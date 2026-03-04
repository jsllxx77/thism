# ThisM — 轻量级自托管服务器监控工具 设计文档

**日期：** 2026-03-04
**项目名：** ThisM
**定位：** 面向技术爱好者/开发者的轻量级自托管服务器监控解决方案

---

## 1. 整体架构

ThisM 由两个独立的 Go 二进制组成：

### `thisM-server`（主控端）
- HTTP/WebSocket 服务器，监听单一端口（默认 `:8080`）
- 内嵌编译后的 React 前端静态文件（`embed.FS`）
- 管理所有 Agent 连接、存储历史数据到 SQLite、提供 REST API
- 提供 Web UI 和 API 两种访问入口

### `thisM-agent`（被控端）
- 在每台被监控的服务器上运行
- 通过 WebSocket 长连接主动连接主控
- 定期采集本机指标并推送，同时响应主控下发的指令
- 断线后自动重连（指数退避）

### 数据流

```
被控机                          主控机
thisM-agent  ─── WebSocket ──▶  thisM-server ──▶ SQLite
                                      │
                                      ▼
                              React Web UI (embed)
                              REST API
```

连接鉴权：Agent 注册时使用预设的 Token，主控验证后建立会话。

---

## 2. 核心组件与数据模型

### 主控端模块划分

| 模块 | 职责 |
|------|------|
| `ws/hub` | 管理所有 Agent WebSocket 连接，维护在线状态 |
| `api` | REST API 路由，提供节点列表、历史数据、服务状态等接口 |
| `store` | SQLite 数据访问层，封装所有读写操作 |
| `collector` | 接收 Agent 推送的指标数据，写入 store |
| `frontend` | embed 静态文件服务 |

### SQLite 表结构

```sql
-- 节点注册信息
nodes (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  token      TEXT NOT NULL UNIQUE,
  ip         TEXT,
  os         TEXT,
  arch       TEXT,
  created_at INTEGER,
  last_seen  INTEGER
)

-- 时序指标快照（滚动保留，默认 30 天）
metrics (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id     TEXT NOT NULL,
  ts          INTEGER NOT NULL,
  cpu_percent REAL,
  mem_used    INTEGER,
  mem_total   INTEGER,
  disk_used   INTEGER,
  disk_total  INTEGER,
  net_rx      INTEGER,
  net_tx      INTEGER
)

-- 进程快照（仅保留最新一次）
processes (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id     TEXT NOT NULL,
  ts          INTEGER NOT NULL,
  pid         INTEGER,
  name        TEXT,
  cpu_percent REAL,
  mem_rss     INTEGER
)

-- 服务健康检测
service_checks (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  node_id      TEXT NOT NULL,
  name         TEXT NOT NULL,
  check_type   TEXT,
  target       TEXT,
  status       TEXT,
  last_checked INTEGER
)
```

### Agent 推送数据包格式（JSON over WebSocket）

```json
{
  "type": "metrics",
  "ts": 1709500000,
  "cpu": 23.5,
  "mem": { "used": 2048, "total": 8192 },
  "disk": [{ "mount": "/", "used": 10240, "total": 51200 }],
  "net": { "rx_bytes": 1024, "tx_bytes": 512 },
  "processes": [{ "pid": 123, "name": "nginx", "cpu": 0.1, "mem": 64 }],
  "services": [{ "name": "nginx", "status": "running" }]
}
```

---

## 3. API 设计

### REST API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/nodes` | 获取所有节点列表及在线状态 |
| `GET` | `/api/nodes/:id/metrics` | 获取节点历史指标（支持时间范围查询） |
| `GET` | `/api/nodes/:id/processes` | 获取节点当前进程列表 |
| `GET` | `/api/nodes/:id/services` | 获取节点服务健康状态 |
| `POST` | `/api/nodes/register` | Agent 注册（返回分配的 node_id） |

### WebSocket 端点

| 端点 | 说明 |
|------|------|
| `WS /ws/agent` | Agent 专用连接端点（需 Token 鉴权） |
| `WS /ws/dashboard` | 前端实时数据订阅端点 |

**实时更新机制：** 前端通过 `/ws/dashboard` 订阅，主控收到 Agent 推送后同时广播给所有在线前端客户端。

---

## 4. 错误处理与部署

### 连接稳定性
- Agent 断线后以指数退避策略自动重连（1s → 2s → 4s → 最大 60s）
- 主控标记节点为 `offline` 并保留历史数据，重连后自动恢复 `online`
- 前端 WebSocket 断线后自动重连，UI 显示"连接中"状态

### 数据老化
- 主控每天定期清理超过保留期（默认 30 天）的 metrics 记录
- 进程快照仅保留最新一次，不累积历史

### 安全
- Agent 连接必须携带有效 Token（Header 或 QueryParam），无效直接拒绝
- Web UI 支持 Bearer Token 保护（可配置）
- 所有敏感配置通过环境变量或配置文件传入，不硬编码

### 部署方式

```bash
# 主控（单命令启动）
./thisM-server --port 8080 --db ./data.db --token <admin-token>

# Agent（被控机上）
./thisM-agent --server wss://your-host:8080 --token <node-token> --name "prod-1"
```

支持通过 systemd 或 Docker 运行，提供示例配置文件。

---

## 5. 前端视觉设计

**技术栈：** React + TypeScript + Tailwind CSS + shadcn/ui + Recharts

**设计风格：** 深色主题为主（可切换浅色），玻璃拟态卡片 + 细边框，整体参考 Vercel / Linear 的极简科技感。

### 响应式布局策略

| 断点 | 布局 |
|------|------|
| PC（>1024px） | 侧边导航 + 主内容区，节点卡片 3-4 列网格 |
| 平板（768-1024px） | 侧边栏可折叠，节点卡片 2 列 |
| 移动端（<768px） | 底部 Tab 导航，节点卡片单列全宽 |

### 页面结构

```
├── 总览页（Dashboard）
│   ├── 顶部汇总条：在线节点数、全局平均 CPU/内存（大数字 + sparkline）
│   └── 节点卡片列表：深色磨砂玻璃背景，CPU/内存进度条，在线状态脉冲指示灯
│
├── 节点详情页
│   ├── CPU / 内存 / 网络：Recharts 渐变面积折线图，时间范围选择（1h/6h/24h/7d）
│   ├── 磁盘：环形进度图 + 挂载点列表
│   ├── 进程列表：虚拟滚动表格，支持按 CPU/内存排序
│   └── 服务健康：状态徽章（绿色 Running / 红色 Dead / 灰色 Unknown）
│
└── 设置页
    ├── Token 管理（生成/撤销）
    └── 数据保留策略配置
```

**离线节点：** 卡片整体降低不透明度 + 灰色调，角标显示"最后在线时间"。

---

## 6. 监控指标范围

- **系统基础：** CPU 使用率、内存（used/total）、磁盘（各挂载点 used/total）、网络流量（rx/tx）
- **进程：** 进程列表（PID、名称、CPU%、内存 RSS），支持排序
- **服务健康：** 关键服务存活检测（systemd 服务 / 端口探测）

---

## 7. 技术选型汇总

| 层级 | 技术 |
|------|------|
| 主控端语言 | Go |
| 数据库 | SQLite（via `modernc.org/sqlite` 纯 Go 实现，零 CGo） |
| WebSocket | `gorilla/websocket` |
| HTTP 路由 | `chi` 或标准库 `net/http` |
| 前端框架 | React + TypeScript |
| UI 组件库 | shadcn/ui + Tailwind CSS |
| 图表库 | Recharts |
| 前端构建 | Vite |
| Agent 系统指标采集 | `gopsutil` |
