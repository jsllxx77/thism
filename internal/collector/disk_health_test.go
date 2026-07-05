package collector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thism-dev/thism/internal/models"
)

func TestCollectDiskHealthEnumeratesSysfsAndNVMeHealth(t *testing.T) {
	tempDir := t.TempDir()
	sysBlock := filepath.Join(tempDir, "sys", "block")
	devRoot := filepath.Join(tempDir, "dev")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatalf("mkdir dev root: %v", err)
	}
	writeSysfsFile(t, sysBlock, "nvme0n1", "queue/rotational", "0\n")
	writeSysfsFile(t, sysBlock, "nvme0n1", "size", "2000\n")
	writeSysfsFile(t, sysBlock, "nvme0n1", "device/model", "Fast NVMe\n")
	writeSysfsFile(t, sysBlock, "nvme0n1", "device/serial", "NVME123\n")
	writeSysfsFile(t, sysBlock, "nvme0n1", "device/firmware_rev", "1.2.3\n")
	writeSysfsFile(t, sysBlock, "sda", "queue/rotational", "1\n")
	writeSysfsFile(t, sysBlock, "sda", "size", "4000\n")
	writeSysfsFile(t, sysBlock, "sda", "device/model", "Bulk SATA\n")
	writeSysfsFile(t, sysBlock, "loop0", "queue/rotational", "0\n")
	if err := os.WriteFile(filepath.Join(devRoot, "nvme0"), nil, 0o600); err != nil {
		t.Fatalf("write nvme controller placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devRoot, "nvme0n1"), nil, 0o600); err != nil {
		t.Fatalf("write nvme namespace placeholder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devRoot, "sda"), nil, 0o600); err != nil {
		t.Fatalf("write sata placeholder: %v", err)
	}

	originalSysBlockPath := diskHealthSysBlockPath
	originalDevRootPath := diskHealthDevRootPath
	originalReadNVMeHealthLogFunc := readNVMeHealthLogFunc
	originalReadATAHealthFunc := readATAHealthFunc
	defer func() {
		diskHealthSysBlockPath = originalSysBlockPath
		diskHealthDevRootPath = originalDevRootPath
		readNVMeHealthLogFunc = originalReadNVMeHealthLogFunc
		readATAHealthFunc = originalReadATAHealthFunc
	}()

	diskHealthSysBlockPath = sysBlock
	diskHealthDevRootPath = devRoot
	readNVMeHealthLogFunc = func(controllerPath string) (nvmeHealthLog, error) {
		if controllerPath != filepath.Join(devRoot, "nvme0") {
			t.Fatalf("expected controller path %s, got %s", filepath.Join(devRoot, "nvme0"), controllerPath)
		}
		temp := 42.0
		lifeUsed := 7.0
		spare := 91.0
		return nvmeHealthLog{
			CriticalWarning: 0,
			TemperatureC:    &temp,
			LifeUsedPercent: &lifeUsed,
			AvailableSpare:  &spare,
			PowerOnHours:    1234,
			UnsafeShutdowns: 5,
			MediaErrors:     0,
		}, nil
	}
	readATAHealthFunc = func(devicePath string) (ataHealthLog, error) {
		if devicePath != filepath.Join(devRoot, "sda") {
			t.Fatalf("expected ata path %s, got %s", filepath.Join(devRoot, "sda"), devicePath)
		}
		temp := 38.0
		lifeUsed := 12.0
		return ataHealthLog{
			TemperatureC:          &temp,
			LifeUsedPercent:       &lifeUsed,
			PowerOnHours:          4321,
			ReallocatedSectors:    0,
			PendingSectors:        0,
			OfflineUncorrectable:  0,
			InterfaceCRCErrors:    0,
			ReportedUncorrectable: 0,
		}, nil
	}

	disks := collectDiskHealth()
	if len(disks) != 2 {
		t.Fatalf("expected nvme and sata disks, got %#v", disks)
	}

	nvme := disks[0]
	if nvme.Name != "nvme0n1" || nvme.Path != filepath.Join(devRoot, "nvme0n1") || nvme.Type != models.DiskHealthTypeNVMe {
		t.Fatalf("unexpected nvme disk identity: %#v", nvme)
	}
	if nvme.Status != models.DiskHealthStatusOK || nvme.TemperatureC == nil || *nvme.TemperatureC != 42 || nvme.LifeUsedPercent == nil || *nvme.LifeUsedPercent != 7 || nvme.AvailableSparePercent == nil || *nvme.AvailableSparePercent != 91 {
		t.Fatalf("unexpected nvme health: %#v", nvme)
	}
	if nvme.SupportStatus != models.DiskHealthSupportSupported {
		t.Fatalf("expected nvme physical health support, got %#v", nvme)
	}
	if nvme.SizeBytes != 1024000 || nvme.PowerOnHours != 1234 || nvme.UnsafeShutdowns != 5 {
		t.Fatalf("unexpected nvme counters: %#v", nvme)
	}

	sata := disks[1]
	if sata.Name != "sda" || sata.Type != models.DiskHealthTypeATA || sata.Status != models.DiskHealthStatusOK {
		t.Fatalf("expected sata disk to report embedded SMART health, got %#v", sata)
	}
	if sata.SupportStatus != models.DiskHealthSupportSupported {
		t.Fatalf("expected sata physical health support, got %#v", sata)
	}
	if sata.TemperatureC == nil || *sata.TemperatureC != 38 || sata.LifeUsedPercent == nil || *sata.LifeUsedPercent != 12 || sata.PowerOnHours != 4321 {
		t.Fatalf("unexpected sata health: %#v", sata)
	}
}

func TestCollectDiskHealthReportsVirtualDiskObservableState(t *testing.T) {
	tempDir := t.TempDir()
	sysBlock := filepath.Join(tempDir, "sys", "block")
	devRoot := filepath.Join(tempDir, "dev")
	procMounts := filepath.Join(tempDir, "proc", "mounts")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatalf("mkdir dev root: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(procMounts), 0o755); err != nil {
		t.Fatalf("mkdir proc root: %v", err)
	}
	writeSysfsFile(t, sysBlock, "vda", "queue/rotational", "0\n")
	writeSysfsFile(t, sysBlock, "vda", "ro", "1\n")
	writeSysfsFile(t, sysBlock, "vda", "size", "4096\n")
	if err := os.WriteFile(filepath.Join(devRoot, "vda"), nil, 0o600); err != nil {
		t.Fatalf("write virtual disk placeholder: %v", err)
	}
	if err := os.WriteFile(procMounts, []byte("/dev/vda1 / ext4 ro,relatime 0 0\n"), 0o644); err != nil {
		t.Fatalf("write proc mounts: %v", err)
	}

	originalSysBlockPath := diskHealthSysBlockPath
	originalDevRootPath := diskHealthDevRootPath
	originalProcMountsPath := diskHealthProcMountsPath
	defer func() {
		diskHealthSysBlockPath = originalSysBlockPath
		diskHealthDevRootPath = originalDevRootPath
		diskHealthProcMountsPath = originalProcMountsPath
	}()

	diskHealthSysBlockPath = sysBlock
	diskHealthDevRootPath = devRoot
	diskHealthProcMountsPath = procMounts

	disks := collectDiskHealth()
	if len(disks) != 1 {
		t.Fatalf("expected one virtual disk, got %#v", disks)
	}
	disk := disks[0]
	if disk.Name != "vda" || disk.Type != models.DiskHealthTypeVirtual {
		t.Fatalf("unexpected virtual disk identity: %#v", disk)
	}
	if disk.Status != models.DiskHealthStatusUnsupported || disk.SupportStatus != models.DiskHealthSupportUnsupported {
		t.Fatalf("expected virtual disk physical health to be unsupported, got %#v", disk)
	}
	if !disk.ReadOnly {
		t.Fatalf("expected read-only state from sysfs/proc mounts, got %#v", disk)
	}
	if len(disk.Mounts) != 1 || disk.Mounts[0].Mount != "/" || disk.Mounts[0].FSType != "ext4" || !disk.Mounts[0].ReadOnly {
		t.Fatalf("expected observable mount state, got %#v", disk.Mounts)
	}
	if disk.SizeBytes != 2097152 {
		t.Fatalf("expected virtual disk size to remain observable, got %#v", disk)
	}
}

func TestCollectDiskHealthTreatsCloudNVMeAsVirtualHealth(t *testing.T) {
	tempDir := t.TempDir()
	sysBlock := filepath.Join(tempDir, "sys", "block")
	devRoot := filepath.Join(tempDir, "dev")
	writeSysfsFile(t, sysBlock, "nvme1n1", "queue/rotational", "0\n")
	writeSysfsFile(t, sysBlock, "nvme1n1", "device/model", "Amazon Elastic Block Store\n")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatalf("mkdir dev root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devRoot, "nvme1n1"), nil, 0o600); err != nil {
		t.Fatalf("write cloud nvme placeholder: %v", err)
	}

	originalSysBlockPath := diskHealthSysBlockPath
	originalDevRootPath := diskHealthDevRootPath
	originalReadNVMeHealthLogFunc := readNVMeHealthLogFunc
	defer func() {
		diskHealthSysBlockPath = originalSysBlockPath
		diskHealthDevRootPath = originalDevRootPath
		readNVMeHealthLogFunc = originalReadNVMeHealthLogFunc
	}()

	diskHealthSysBlockPath = sysBlock
	diskHealthDevRootPath = devRoot
	readNVMeHealthLogFunc = func(string) (nvmeHealthLog, error) {
		t.Fatal("cloud NVMe disks must not use hardware NVMe health passthrough")
		return nvmeHealthLog{}, nil
	}

	disks := collectDiskHealth()
	if len(disks) != 1 {
		t.Fatalf("expected one cloud nvme disk, got %#v", disks)
	}
	if disks[0].Type != models.DiskHealthTypeVirtual || disks[0].SupportStatus != models.DiskHealthSupportUnsupported || disks[0].Status != models.DiskHealthStatusUnsupported {
		t.Fatalf("expected cloud nvme physical health to be unsupported, got %#v", disks[0])
	}
}

func TestCollectDiskHealthMarksPhysicalHealthAccessDegraded(t *testing.T) {
	tempDir := t.TempDir()
	sysBlock := filepath.Join(tempDir, "sys", "block")
	devRoot := filepath.Join(tempDir, "dev")
	writeSysfsFile(t, sysBlock, "sda", "queue/rotational", "1\n")
	writeSysfsFile(t, sysBlock, "sda", "ro", "0\n")
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatalf("mkdir dev root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devRoot, "sda"), nil, 0o600); err != nil {
		t.Fatalf("write sata placeholder: %v", err)
	}

	originalSysBlockPath := diskHealthSysBlockPath
	originalDevRootPath := diskHealthDevRootPath
	originalReadATAHealthFunc := readATAHealthFunc
	defer func() {
		diskHealthSysBlockPath = originalSysBlockPath
		diskHealthDevRootPath = originalDevRootPath
		readATAHealthFunc = originalReadATAHealthFunc
	}()

	diskHealthSysBlockPath = sysBlock
	diskHealthDevRootPath = devRoot
	readATAHealthFunc = func(string) (ataHealthLog, error) {
		return ataHealthLog{}, errATASmartUnsupported
	}

	disks := collectDiskHealth()
	if len(disks) != 1 {
		t.Fatalf("expected one sata disk, got %#v", disks)
	}
	if disks[0].Status != models.DiskHealthStatusUnknown || disks[0].SupportStatus != models.DiskHealthSupportDegraded {
		t.Fatalf("expected degraded physical health access, got %#v", disks[0])
	}
	if !strings.Contains(disks[0].Message, "SMART") {
		t.Fatalf("expected degraded support message to mention SMART failure, got %#v", disks[0])
	}
}

func TestCollectDiskHealthFollowsSysfsBlockSymlinks(t *testing.T) {
	tempDir := t.TempDir()
	sysBlock := filepath.Join(tempDir, "sys", "block")
	realDevice := filepath.Join(tempDir, "devices", "pci0000:00", "sda")
	devRoot := filepath.Join(tempDir, "dev")
	if err := os.MkdirAll(sysBlock, 0o755); err != nil {
		t.Fatalf("mkdir sys block: %v", err)
	}
	if err := os.MkdirAll(devRoot, 0o755); err != nil {
		t.Fatalf("mkdir dev root: %v", err)
	}
	writeSysfsFile(t, filepath.Dir(realDevice), "sda", "queue/rotational", "1\n")
	writeSysfsFile(t, filepath.Dir(realDevice), "sda", "size", "8000\n")
	if err := os.Symlink(realDevice, filepath.Join(sysBlock, "sda")); err != nil {
		t.Fatalf("symlink sys block device: %v", err)
	}
	if err := os.WriteFile(filepath.Join(devRoot, "sda"), nil, 0o600); err != nil {
		t.Fatalf("write sata placeholder: %v", err)
	}

	originalSysBlockPath := diskHealthSysBlockPath
	originalDevRootPath := diskHealthDevRootPath
	originalReadATAHealthFunc := readATAHealthFunc
	defer func() {
		diskHealthSysBlockPath = originalSysBlockPath
		diskHealthDevRootPath = originalDevRootPath
		readATAHealthFunc = originalReadATAHealthFunc
	}()

	diskHealthSysBlockPath = sysBlock
	diskHealthDevRootPath = devRoot
	readATAHealthFunc = func(string) (ataHealthLog, error) {
		return ataHealthLog{}, nil
	}

	disks := collectDiskHealth()
	if len(disks) != 1 {
		t.Fatalf("expected symlinked block device to be collected, got %#v", disks)
	}
	if disks[0].Name != "sda" || disks[0].SizeBytes != 4096000 {
		t.Fatalf("unexpected symlinked disk stats: %#v", disks[0])
	}
}

func TestParseNVMeHealthLogDerivesStatus(t *testing.T) {
	raw := make([]byte, 512)
	raw[0] = 0x01
	temperatureKelvin := uint16(343)
	raw[1] = byte(temperatureKelvin)
	raw[2] = byte(temperatureKelvin >> 8)
	raw[3] = 8
	raw[4] = 10
	raw[5] = 99
	writeUint128Low64(raw[128:144], 2222)
	writeUint128Low64(raw[144:160], 3)
	writeUint128Low64(raw[160:176], 4)

	log := parseNVMeHealthLog(raw)
	if log.TemperatureC == nil || *log.TemperatureC < 69.8 || *log.TemperatureC > 69.9 {
		t.Fatalf("expected parsed celsius temperature near 69.85, got %#v", log.TemperatureC)
	}
	if log.LifeUsedPercent == nil || *log.LifeUsedPercent != 99 {
		t.Fatalf("expected life used 99, got %#v", log.LifeUsedPercent)
	}
	if log.AvailableSpare == nil || *log.AvailableSpare != 8 {
		t.Fatalf("expected spare 8, got %#v", log.AvailableSpare)
	}
	if log.PowerOnHours != 2222 || log.UnsafeShutdowns != 3 || log.MediaErrors != 4 {
		t.Fatalf("unexpected nvme counters: %#v", log)
	}

	stats := diskHealthFromNVMeLog(models.DiskHealthStats{Name: "nvme0n1", Type: models.DiskHealthTypeNVMe}, log)
	if stats.Status != models.DiskHealthStatusCritical {
		t.Fatalf("expected critical status from warning bit, got %#v", stats)
	}
}

func TestParseATAHealthLogDerivesStatus(t *testing.T) {
	raw := make([]byte, 512)
	writeATAAttribute(raw, 9, 99, 0, 2222)
	writeATAAttribute(raw, 194, 70, 0, 43)
	writeATAAttribute(raw, 5, 100, 10, 2)
	writeATAAttribute(raw, 197, 100, 10, 1)
	writeATAAttribute(raw, 198, 100, 10, 0)
	writeATAAttribute(raw, 199, 100, 10, 3)
	writeATAAttribute(raw, 233, 88, 10, 12)

	log := parseATAHealthLog(raw)
	if log.TemperatureC == nil || *log.TemperatureC != 43 {
		t.Fatalf("expected parsed ata temperature 43, got %#v", log.TemperatureC)
	}
	if log.LifeUsedPercent == nil || *log.LifeUsedPercent != 12 {
		t.Fatalf("expected parsed ata life used 12, got %#v", log.LifeUsedPercent)
	}
	if log.PowerOnHours != 2222 || log.ReallocatedSectors != 2 || log.PendingSectors != 1 || log.InterfaceCRCErrors != 3 {
		t.Fatalf("unexpected ata counters: %#v", log)
	}

	stats := diskHealthFromATAHealthLog(models.DiskHealthStats{Name: "sda", Type: models.DiskHealthTypeATA}, log)
	if stats.Status != models.DiskHealthStatusCritical {
		t.Fatalf("expected critical status from pending sectors, got %#v", stats)
	}
	if stats.MediaErrors != 3 || stats.ReallocatedSectors != 2 || stats.PendingSectors != 1 || stats.InterfaceCRCErrors != 3 {
		t.Fatalf("expected ata error counters to be exposed, got %#v", stats)
	}
}

func writeSysfsFile(t *testing.T, root, device, relativePath, value string) {
	t.Helper()
	path := filepath.Join(root, device, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeATAAttribute(target []byte, id byte, current byte, threshold byte, raw uint64) {
	for index := 2; index+12 <= 362; index += 12 {
		if target[index] == 0 {
			target[index] = id
			target[index+1] = 0x03
			target[index+3] = current
			for rawIndex := 0; rawIndex < 6; rawIndex++ {
				target[index+5+rawIndex] = byte(raw >> (8 * rawIndex))
			}
			thresholdOffset := 362 + (index - 2)
			if thresholdOffset+12 <= len(target) {
				target[thresholdOffset] = id
				target[thresholdOffset+1] = threshold
			}
			return
		}
	}
}

func writeUint128Low64(target []byte, value uint64) {
	for index := 0; index < 8; index++ {
		target[index] = byte(value >> (8 * index))
	}
}
