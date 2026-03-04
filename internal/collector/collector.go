package collector

import (
	"encoding/json"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/thism-dev/thism/internal/models"
)

// Collector gathers system metrics and pushes them to the ThisM server via WebSocket.
type Collector struct {
	serverURL string
	token     string
	name      string
}

// New creates a new Collector.
func New(serverURL, token, name string) *Collector {
	return &Collector{
		serverURL: serverURL,
		token:     token,
		name:      name,
	}
}

// Collect gathers a single snapshot of system metrics.
func (c *Collector) Collect() (*models.MetricsPayload, error) {
	payload := &models.MetricsPayload{
		Type:     "metrics",
		TS:       time.Now().Unix(),
		Services: []models.Service{},
	}

	// CPU — blocks for 1 second for an accurate reading.
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercents) > 0 {
		payload.CPU = cpuPercents[0]
	}

	// Memory.
	vmStat, err := mem.VirtualMemory()
	if err == nil {
		payload.Mem = models.MemStats{
			Used:  vmStat.Used,
			Total: vmStat.Total,
		}
	}

	// Disk — iterate partitions, skip any that fail Usage().
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, p := range partitions {
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}
			payload.Disk = append(payload.Disk, models.DiskStats{
				Mount: p.Mountpoint,
				Used:  usage.Used,
				Total: usage.Total,
			})
		}
	}

	// Network — aggregate across all interfaces.
	ioCounters, err := psnet.IOCounters(false)
	if err == nil && len(ioCounters) > 0 {
		payload.Net = models.NetStats{
			RxBytes: ioCounters[0].BytesRecv,
			TxBytes: ioCounters[0].BytesSent,
		}
	}

	// Processes — collect up to 30 processes.
	procs, err := process.Processes()
	if err == nil {
		for _, p := range procs {
			if len(payload.Processes) >= 30 {
				break
			}
			name, err := p.Name()
			if err != nil {
				continue
			}
			cpuPct, err := p.CPUPercent()
			if err != nil {
				continue
			}
			memInfo, err := p.MemoryInfo()
			if err != nil {
				continue
			}
			var rss uint64
			if memInfo != nil {
				rss = memInfo.RSS
			}
			payload.Processes = append(payload.Processes, models.Process{
				PID:        p.Pid,
				Name:       name,
				CPUPercent: cpuPct,
				MemRSS:     rss,
			})
		}
	}
	if payload.Processes == nil {
		payload.Processes = []models.Process{}
	}

	return payload, nil
}

// Run starts a blocking loop that connects to the server with exponential backoff.
func (c *Collector) Run() {
	backoff := time.Second
	for {
		err := c.connect()
		if err != nil {
			log.Printf("collector: disconnected: %v — retrying in %s", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
		} else {
			// Successful clean exit — reset backoff.
			backoff = time.Second
		}
	}
}

// connect dials the server, then sends metrics every 5 seconds until an error occurs.
func (c *Collector) connect() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}
	u.Path = "/ws/agent"
	q := u.Query()
	q.Set("token", c.token)
	u.RawQuery = q.Encode()

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Reset backoff on successful connection.
	log.Printf("collector: connected to %s", u.String())

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics, err := c.Collect()
		if err != nil {
			return err
		}
		data, err := json.Marshal(metrics)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
	}

	return nil
}
