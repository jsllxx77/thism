# 并发控制面修复实施计划

## 1. 目的与边界

本计划针对 `thism` 的并发安全与生命周期风险，服务于以下控制面链路：

```text
API 创建自更新任务
  -> Hub 选择、替换并写入 Agent 连接
  -> Agent 持久化并执行自更新
  -> Agent 回报状态
  -> 服务端持久化、汇总并广播
```

深审依赖包括会影响这条链路正确性或全局可用性的 SQLite、告警 Dispatcher、dashboard 广播、后台 worker 和 shutdown。一般鉴权、SSRF、供应链、指标计算和 UI 细节不在本计划内，除非它们直接造成并发、生命周期、背压或控制面问题。

当前部署基线：单实例服务端、最多 50 个 Agent、每 5 秒上报、最多 2 个 dashboard 订阅者、最多 50 个节点批量更新，故障时 50 个 Agent 可在 60 秒内重连并重放状态。

本计划是整改计划，不是动态验证结果。当前实现中的问题在正式审查报告中必须按 `Confirmed`、`High-confidence static` 和 `Needs validation` 分类，不能把本计划中的目标状态误写成现状。

## 2. 不可违反的不变量

### 2.1 投递与执行

- 对每个 `(JobID, nodeID)`，服务端允许重投，但 Agent 的更新副作用最多执行一次。
- Agent 按 `JobID` 持久去重；重复命令只回放 `accepted`、进行中或终态，不重新下载、替换或重启。
- 不同 `JobID` 的更新不得并发执行；自动更新和远程更新共享同一个版本切换互斥边界。
- `os.Rename` 替换当前二进制是不可逆提交点。此前可以取消并清理临时文件；此后必须继续 `Exec` 或用 SHA256 恢复，不能把取消当作副作用未发生。

### 2.2 连接代际

- 每次注册为一个 `nodeID` 分配单调连接代际。
- 新连接成为 `current` 与旧连接被 fence/关闭是同一身份切换。
- 只有 current 代际可以提交指标、命令状态、heartbeat 和在线/离线事件。
- 命令发送与连接替换在 Hub 内有明确线性化点：提交给当时的 current writer，或明确失败。
- stale 连接不得删除新连接、广播错误离线状态或继续提交控制面状态。
- Agent 在新连接上重放持久状态，不能依赖旧 socket 继续回报。

### 2.3 状态所有权

对每个 `(JobID, nodeID)` 分开保存两个维度：

- `delivery`：服务端拥有，`pending -> sending -> sent | send_failed`。
- `execution`：Agent 拥有，`accepted -> downloading -> verifying -> restarting -> succeeded | failed | unknown`。

`delivery` 不覆盖 `execution`。执行状态只能合法单调推进，重复同值幂等；`succeeded`、`failed` 和 `unknown` 为 sticky 终态。`unknown` 只能由制品 SHA256 证据或操作员显式处置解析。

状态检查、CAS/合法转换和持久化必须在同一事务边界内完成。dashboard 只能广播提交后的规范化状态，不能广播被拒绝或尚未提交的原始事件。

### 2.4 序列、流身份和恢复

- 每个 node 使用服务端分配且持久化的连续 command sequence，同时保留 `JobID`。
- Agent 只接受 `seq = retired_watermark + 1`。
- `seq` 更大时返回 `sequence_gap(expected)`，不缓存、不执行；服务端按序重投缺失项。
- 当前序列只有在持久终态或同序列 cancel tombstone 后才能推进 `retired_watermark`。
- 已分配任务不可直接删除；取消必须使用原序列的持久 tombstone 收口。
- 每个 node 使用跨备份稳定、服务端和 Agent 两端持久化的 `command_stream_id`。
- 握手核对 stream ID 和 Agent 水位完成前，服务端禁止控制命令。
- 服务端数据库回滚时，以 Agent 水位为同一 stream 的不可回退下界，事务性提升 `next_seq`，记录 `rollback-reconciled` 审计区间，绝不复用序列。
- stream 缺失或不匹配、本地防重放状态损坏时，节点进入隔离；不得自动重置 Agent。

## 3. 当前关键实现热点

这些是实施阶段的优先检查和改造入口，不代表每一项已经完成动态确认：

- `internal/hub/hub.go`
  - `Register`、`Unregister` 和 `Run` 的连接身份生命周期。
  - `SendToAgent` 当前在选择连接后释放 Hub 锁，再获取连接写锁并执行网络写入；需要补上相对连接替换的线性化语义。
  - 广播队列、慢订阅者和 Hub 关闭所有权。
- `internal/api/api.go`
  - `handleCreateAgentUpdateJob` / `handleCreateAgentUpdates` 的 online check-then-act、投递状态落库顺序和批量发送。
  - `handleAgentCommandStatus` 的状态合并、存储错误处理和广播时序。
  - Agent WebSocket 注册、初始配置写入、heartbeat、状态回报、注销和 dashboard 订阅。
- `internal/store/store.go`
  - `UpdateUpdateJobTargetStatus` 当前无条件覆盖状态。
  - `FinalizeUpdateJobsForNodeVersion` 依赖版本字符串，需改为基于 stream、序列和 SHA256 的恢复事实。
  - update job schema、事务分配序列、tombstone、审计和回滚追平。
- `internal/collector/collector.go`
  - `dispatchAgentCommand` 只有进程内 `updateInProgress`，没有持久 JobID 去重。
  - 自动更新路径直接调用 `selfUpdateFunc`，绕过远程更新互斥。
  - 状态回调绑定命令到达时的旧 WebSocket。
  - `runSelfUpdate` 当前使用 `context.Background()`、2 分钟 HTTP 总超时、无阶段租约和无 Rename 前取消边界。
  - `os.Rename`、版本文件写入、状态记录和 `syscall.Exec` 的崩溃恢复窗口。
- `internal/alerting/dispatcher.go`、`cmd/server/main.go`
  - Dispatcher 的 drop alert goroutine、关闭等待和网络发送是否被 WaitGroup/Context 覆盖。
  - 指标 retention、rollup 和 HTTP shutdown 的后台循环是否绑定主生命周期，是否在 Store 关闭前停止。

## 4. 阶段总览与依赖

| 阶段 | 模块边界 | 目标 | 前置依赖 | 建议责任角色 |
| --- | --- | --- | --- | --- |
| 0 | 审计基线与协议冻结 | 把不变量、状态图、现状证据和回归基线固定下来 | 无 | 技术负责人/并发审查负责人 |
| 1 | API / Hub / Store | 先消除服务端错发、stale 事件和状态回退 | 阶段 0；模型与 schema 设计 | 服务端控制面维护角色 |
| 2 | Agent | 实现持久去重、序列接收、恢复、租约和提交点取消 | 阶段 1 的协议字段；本地状态格式 | Agent 维护角色 |
| 3 | Dispatcher / 后台生命周期 | 保证慢消费者、告警、后台 worker 和 shutdown 不拖垮控制面 | 阶段 1 的 Hub/Store 生命周期接口 | 平台/运行时维护角色 |
| 4 | 测试基础设施与服务验证 | 把竞态、故障和重连基线变成稳定回归 | 阶段 1-3 的可注入接口 | 测试基础设施/QA 角色 |
| 5 | 迁移、灰度与发布 | 处理旧节点、旧记录和恢复场景，按发布门槛逐步启用 | 阶段 1-4 通过验收 | 发布负责人/项目负责人 |

每阶段完成后都应保持源代码可构建、已有测试可运行；协议迁移不得让旧 Agent 在未完成握手和状态恢复前继续接收控制命令。

## 5. 阶段 0：冻结协议与证据基线

### 工作项

1. 在模型层定义并冻结：
   - `command_stream_id`、`seq`、`retired_watermark`。
   - 连接代际和 current/stale 事件身份。
   - delivery、execution、observation、cancel intent/tombstone 的边界。
   - `unknown`、`control-quarantined`、`control-blocked`、`sequence_gap` 和 `expired/already_processed` 的协议语义。
2. 绘制现有 API、Hub、Agent、Store、Dispatcher 和 shutdown 的时序图，标出锁、channel、goroutine、网络 I/O 和持久化边界。
3. 固定旧行为的回归基线：现有测试结果、`go test -race ./...` 结果、服务启动方式和可观测日志字段。
4. 记录每个潜在发现的代码路径、参与者、触发顺序、违反的不变量、影响和证据类型。

### 验收标准

- 每个后续状态转换都能映射到唯一的所有者和持久化边界。
- 不存在“先实现字段、后决定语义”的未决协议字段。
- 报告可以明确区分代码确定性证明、稳定复现和待验证风险。

## 6. 阶段 1：服务端 API / Hub / Store

### 6.1 Hub 连接代际与发送线性化

**建议责任角色：** 服务端控制面维护角色。

**工作项：**

1. 为每个 node 持久化或可恢复地维护连接代际；注册时完成 current 替换和旧连接 fencing。
2. 将命令发送与连接替换纳入同一 Hub 串行化协议。建议使用明确的发送请求/结果边界或 per-agent writer admission，而不是让调用者持有一个可能已过期的连接指针。
3. 对待发送控制命令绑定 node、stream 和代际；切换前未提交的命令明确返回失败，允许同一 `JobID` 重投。
4. 关闭 stale socket 时不得阻塞 Hub 全局事件循环；避免 stale writer 在切换后继续排队。
5. 为 heartbeat、指标、命令状态和在线/离线事件统一加入代际校验。
6. 增加 Hub 的 context/Close 生命周期，关闭 Register、Unregister、Send 和 Broadcast 的入口，避免 channel 在服务关闭后永久阻塞。

**前置依赖：** 阶段 0 的代际模型和发送结果语义。

**验收标准：**

- 新连接替换旧连接后，旧连接的注销、状态、指标和 heartbeat 都被忽略。
- 命令不会在替换窗口中静默落到未知代际；结果只能是提交到 current writer 或明确失败。
- 同一 node 的两个连接不会并发成为有效 current。
- 关闭 Hub 后，所有入口快速返回明确错误，Hub goroutine 能在约定 shutdown deadline 内退出。

### 6.2 API 投递与状态合并

**建议责任角色：** 服务端 API/控制面维护角色。

**工作项：**

1. 将 `IsOnline` 后再 `SendToAgent` 的 check-then-act 改为 Hub 原子发送请求；在线检查只用于展示或初步筛选，不能作为投递正确性的依据。
2. 创建任务时，在数据库事务中写入 job、node target、stream/sequence 元数据和初始 `delivery=pending`。
3. 为投递尝试实现 `pending -> sending -> sent | send_failed`，并保留错误、代际和尝试时间；发送不确定时允许同一 `JobID` 重投。
4. 将 Agent 状态回报路由到带 stream、seq、generation 和 JobID 的 CAS/合法转换函数。
5. 拒绝迟到 `dispatched`、重复或非法 execution 转换；重复同值必须幂等。
6. 只有事务提交后的规范化状态才可广播；存储失败或转换被拒绝时不得广播原始事件。
7. 任务汇总区分 delivery 失败、execution 失败、`unknown`、取消和观察停滞。

**验收标准：**

- `accepted` 先提交后，迟到的 `dispatched` 不会回退状态。
- `send_failed` 不会覆盖 Agent 已提交的 `accepted` 或更后 execution。
- 同一状态重复回报不会重复改变 `updated_at` 以外的事实，也不会造成错误告警风暴。
- dashboard 看到的每个状态都能在提交后的 Store 快照中找到。
- API 发送成功但状态落库失败、连接替换发生在发送中间、发送结果不确定等场景都有明确 delivery 事实和重投路径。

### 6.3 Store schema、序列和回滚

**建议责任角色：** SQLite/Store 维护角色。

**工作项：**

1. 增加每 node 的 `command_stream_id`、`next_seq`、`retired_watermark` 和控制隔离状态。
2. 增加任务的 `seq`、delivery/execution/observation 字段、目标 SHA256、状态时间和代际/stream 关联。
3. 在一个数据库事务中分配序列并创建任务；取消已分配命令时创建同序列 tombstone，禁止出现永久缺口。
4. 为执行状态实现 CAS/合法转换和 sticky 终态；将 `unknown` 解析限定为 SHA256 证据或审计授权。
5. 实现握手 reconciliation：
   - stream 相同且服务端序列落后时，按 Agent 水位事务性提升 `next_seq`。
   - 记录紧凑 `rollback-reconciled` 审计区间。
   - 使用 Agent 保留的详情回补可证明事实，不从缺失数据推断成功。
   - stream 缺失/不匹配时将 node 隔离。
6. 实现 180 天/1024 条详细记录的有界清理；`in_progress`、`unknown` 和审计记录不因 TTL 静默删除。
7. 记录 `rollback-reconciled`、流重绑定、强制采用、unknown 处置和新 node 注册的审计事件。

**验收标准：**

- 并发分配不会产生重复序列或孤立任务。
- 服务端永远不会复用 Agent 已退休或已见过的序列。
- `seq > expected`、旧序列、详情已清理序列的行为分别符合 `sequence_gap`、状态回放和 `expired/already_processed` 语义。
- 数据库回滚测试能证明水位只前进、不回退，且审计区间可查询。
- 任何 schema 迁移失败都不会留下已分配但无任务/tombstone 的序列。

## 7. 阶段 2：Agent 持久执行协议

### 7.1 本地状态与 stream handshake

**建议责任角色：** Agent 维护角色。

**工作项：**

1. 增加原子、带校验的本地状态存储，至少包含：
   - `command_stream_id`。
   - `retired_watermark`。
   - 最近 1024 条 JobID/seq/状态/目标 SHA256/消息记录。
   - `in_progress` 的阶段、`last_progress_at`、提交点和 cancel intent。
   - `unknown` 和 control quarantine/block 状态。
2. 使用临时文件、同步、原子替换和损坏检测，避免部分写入被当成有效水位。
3. Agent 启动时校验本地状态；已有 node 凭据但状态缺失、损坏或无法原子恢复时进入 `control-quarantined`。
4. 新 WebSocket 连接先完成 stream/watermark handshake，再允许控制命令；握手期间仍可上报基础监控。
5. 新连接建立后重放窗口内详细状态和当前隔离状态，不依赖旧 socket。

**验收标准：**

- WebSocket 重连、`syscall.Exec`、进程重启和主机重启后，stream、watermark 和任务状态保持一致。
- 状态文件损坏时不自动从服务端重建水位，不接收更新命令。
- 受信备份恢复、显式采用服务端流和重新注册均生成审计事件，并对无法证明的任务进入 `unknown`。
- 两端防重放证据同时丢失时，系统明确隔离，不声称恢复原 at-most-once 保证。

### 7.2 串行命令接收与 JobID 去重

**建议责任角色：** Agent 更新执行维护角色。

**工作项：**

1. 所有远程和自动更新入口统一进入一个全局版本切换执行器，移除自动更新绕过 `updateMu/updateInProgress` 的路径。
2. 收到 `seq = watermark + 1` 时，在任何下载/替换副作用前原子持久化 `(seq, JobID, SHA256, accepted)`。
3. 收到窗口内重复 JobID/seq 时回放精确状态；收到已退休但详情清理的序列时回 `expired/already_processed`。
4. 收到未来序列时回 `sequence_gap(expected)`，不缓存、不执行。
5. 不同 JobID 继续严格串行；当前任务未进入终态或 tombstone 前，不接受下一个副作用任务。
6. 统一远程更新与自动更新的 JobID/source/sequence 记录方式；若自动更新不走服务端 sequence，必须至少走同一持久执行器和全局互斥，并在协议文档中明确其序列边界。

**验收标准：**

- 重放同一 JobID 100 次最多产生一次下载、Rename 和 Exec 副作用。
- 两个不同 JobID 并发到达时只有一个进入执行，其余得到明确可重试状态，不会并行替换二进制。
- 自动更新与远程更新在所有阶段互斥。
- 进程重启后重复命令只回放状态，不重新执行已完成或 `unknown` 任务。

### 7.3 阶段租约、取消和 Rename 提交点

**建议责任角色：** Agent 更新执行维护角色。

**工作项：**

1. 为执行任务创建 cancel context，并把它绑定到下载、读响应、临时文件处理和验证阶段。
2. 持久化阶段和 `last_progress_at`；阶段推进续租。
3. 默认租约从以下值开始，并允许按实际制品调整：下载 2 分钟、验证/落盘 1 分钟、重启确认 2 分钟。
4. 取消请求写入同一 seq 的持久 cancel intent：
   - Rename 前：取消 context、删除临时文件、写 `cancelled` tombstone、退休序列。
   - Rename 后：拒绝取消，继续 Exec 和恢复核对。
5. Rename 前阶段超时写 execution `failed`；Rename 后阶段超时核对当前可执行文件 SHA256，匹配则 `succeeded`，否则 `unknown` 并设置 `control-blocked`。
6. 版本文件写入、状态持久化和 Exec 失败必须走保守恢复路径，不能直接当成“未执行失败”。

**验收标准：**

- 下载或验证阶段取消可以终止请求并清理临时文件，不推进错误水位。
- Rename 之后的取消请求不会删除或回滚已提交二进制。
- 模拟 Rename 后立即崩溃/Exec 失败时，重启能按 SHA256 收敛到 `succeeded` 或 `unknown`。
- Agent 长时间无进展时能自判失活；服务端只产生 `observation=stalled`，不篡改 execution。
- 首个未处置 `unknown` 立即阻断该 node 后续更新，处置后才恢复。

## 8. 阶段 3：Dispatcher、dashboard 与后台生命周期

### 8.1 Dispatcher 与背压

**建议责任角色：** 平台/运行时维护角色。

**工作项：**

1. 确认控制命令不经过可丢弃的 dashboard/告警队列；控制命令失败必须进入 delivery 状态。
2. Dispatcher 队列保持有界；队列满时只丢弃允许丢弃的观测/告警工作，并记录可查询的 drop 事实。
3. `sendDropAlert` 等异步 goroutine 纳入 WaitGroup/Context，关闭时不得在 Store 关闭后继续发送或写库。
4. 慢 dashboard 只影响自身订阅者；指标和 `node_status` 可 latest-wins 丢弃，状态事实通过快照和任务详情恢复。
5. keepalive、Agent 读循环和 Hub 其他订阅者不得受 dashboard 背压阻塞。

**验收标准：**

- 慢 dashboard 不阻塞 Agent WebSocket 读循环、Hub 其他订阅者或控制命令发送。
- 丢弃指标不会改变已持久化任务状态；dashboard 重连后快照可恢复当前事实。
- Dispatcher 关闭后不存在未跟踪的 drop alert goroutine 或使用已关闭 Store 的 worker。

### 8.2 服务 shutdown 顺序

**建议责任角色：** 服务端运行时维护角色。

**工作项：**

1. 用主 context 统一管理 HTTP server、Hub、Dispatcher、metrics retention pruner、rollup 和其他后台循环。
2. 设计明确关闭顺序：停止接收新控制任务 → fence 新连接/停止 Hub intake → 停止 Agent/dashboard worker → 等待 Dispatcher 和后台任务 → HTTP graceful shutdown → 最后关闭 Store/GeoIP 等共享资源。
3. 为每个 wait 设置 deadline；无法及时结束的操作记录结构化错误和残留资源，不无限等待。
4. 所有 channel send、ticker、timer、keepalive goroutine 和 websocket writer 都有终止条件。
5. 将 shutdown 期间的状态持久化与广播规则写入协议：已提交事实保留，未提交发送明确失败。

**验收标准：**

- 在连接、更新下载、告警发送、慢 dashboard 和 rollup 同时运行时发起 shutdown，服务在约定 deadline 内返回。
- Store 关闭后没有 worker 继续访问它。
- 重复调用 Close/Shutdown 幂等；不会 double close channel、重复 Wait 或死锁。
- shutdown 不会把已执行任务误标为未执行，也不会静默丢失已提交 execution 状态。

## 9. 阶段 4：测试基础设施与验证矩阵

### 9.1 必须新增的确定性测试

**建议责任角色：** 测试基础设施/QA 角色，与各模块维护角色共同维护。

1. **Hub/连接代际**
   - 新连接替换旧连接时，旧连接注销不得删除新连接。
   - 发送与替换并发时，命令只能进入 current writer 或明确失败。
   - stale 连接的 status、metrics、heartbeat 和 offline 事件全部被忽略。
2. **投递与状态合并**
   - `accepted` 先到、`dispatched` 后到时不回退。
   - send failure 与 execution 状态并发时，两个维度独立。
   - 终态 sticky，非法转换和原始未提交广播均被拒绝。
3. **JobID/sequence**
   - 重复 JobID、连接重连、进程重启和 Exec 后重放只产生一次副作用。
   - future sequence 返回 gap 且不缓存；缺失序列补齐后才允许下一序列。
   - cancel tombstone 收口并推进水位；取消不能制造缺口。
   - 数据库回滚时不复用序列，审计区间正确。
4. **Agent 状态恢复**
   - `in_progress` + 当前 SHA256 匹配补成功。
   - `in_progress` + 不匹配进入 unknown 并冻结。
   - 本地状态损坏进入 `control-quarantined`，不自动重建水位。
   - unknown 处置必须有证据/审计，处置前拒绝新更新。
5. **租约与取消**
   - Rename 前取消清理临时文件并写 tombstone。
   - Rename 后取消被拒绝，继续 Exec/核对。
   - 阶段租约续租和超时只由 Agent 改 execution；服务端 stalled 只改 observation。
6. **背压与 shutdown**
   - 慢 dashboard 不阻塞控制面，快照可恢复。
   - Dispatcher drop alert、ticker、keepalive、Hub 和 HTTP server 都能在 shutdown 中退出。
   - Store 关闭后无后台访问。

### 9.2 验证命令与执行顺序

1. 运行受影响包的定向测试。
2. 运行全 Go 包：

   ```bash
   go test ./...
   ```

3. 运行竞态检测：

   ```bash
   go test -race ./...
   ```

4. 执行定向并发/故障测试，使用可控 fake clock、fake socket、可阻塞 writer、故障注入 Store 和可观测 goroutine 生命周期，避免依赖 `time.Sleep` 竞态。
5. 必要时启动服务进行服务级测试，覆盖 50 Agent 在 60 秒内重连、50 节点批量更新、慢 dashboard、进程重启和 shutdown。
6. 保存测试命令、提交版本、日志、失败重现条件和资源统计。无法稳定复现的竞态归入 `Needs validation`，不计入 confirmed findings。

## 10. 阶段 5：迁移、灰度和发布

### 10.1 迁移顺序

1. 先做 additive schema migration：stream、sequence、delivery/execution、审计和本地 Agent 状态字段。
2. 服务端先支持握手、快照和双维度读取，但在旧 Agent 未完成 stream/watermark 核对前禁止发送控制命令。
3. Agent 发布包含本地状态存储、JobID 去重和协议握手；旧 Agent 只能继续监控，不能接收新的控制命令。
4. 对已有节点逐个完成 stream 绑定和水位核对；缺失/不匹配节点进入 `control-quarantined` 或 `control-blocked`，由操作员选择恢复、采用或重新注册。
5. 迁移已有 update job 时保留原 JobID、目标 SHA256 和历史状态；无法证明执行结果的任务转为 `unknown`，不能批量标记成功。
6. 协议和旧字段稳定运行后，再移除无条件状态覆盖、版本字符串自动成功和旧发送路径。

### 10.2 50-Agent 发布阻断建议

以下是建议，不替代项目负责人或发布负责人的最终风险接受：

- **建议阻断发布：** 任何确认的 Critical/High，尤其是命令可发给错误/过期 Agent、旧连接可覆盖新连接、同一 JobID 可重复执行更新、序列可复用、数据库回滚可绕过 Agent 水位、或 shutdown 可造成全局控制面不可用。
- **建议保持阻断：** 已有完整确定性触发路径但尚未动态复现的 Critical/High `High-confidence static` 风险，除非完成针对性验证并证明有等价缓解措施。
- **至少不应阻断但必须进入计划：** 仅影响 dashboard 实时性、可由快照恢复的指标丢失，或仅在远高于 50-Agent 基线才出现的容量问题；前提是没有反向阻塞控制面。
- **需要发布前通过：** 相关现有测试、`go test -race ./...`、连接替换/状态回退/JobID 重放/序列缺口/取消与 shutdown 定向测试，以及 50-Agent/60 秒重连场景。
- **Needs validation：** 单独列出复现条件、所需故障注入和决策期限；不把它们计入 confirmed finding，但若潜在影响为 Critical，应由发布负责人明确风险接受或暂缓发布。

## 11. 每项修复计划的固定模板

每一个 `Critical/High` 和其余确认发现都应按以下字段记录：

- **发现/不变量：** 违反了哪条控制面或生命周期不变量。
- **代码路径：** 文件、函数和关键调用顺序。
- **触发时序：** 并发参与者、锁/channel/network I/O/事务边界和具体 interleaving。
- **影响：** 错发、重复副作用、状态损坏、全局阻塞、消息丢失或资源泄漏。
- **证据：** `Confirmed`、`High-confidence static` 或 `Needs validation`，以及复现命令/日志。
- **修复阶段：** 阶段 1-5 中的目标阶段。
- **建议责任角色：** 服务端控制面、Agent、Store、运行时或测试基础设施角色；不写虚构姓名。
- **前置依赖：** schema、协议、迁移、测试替身或其他修复。
- **验收标准：** 可观察、不依赖主观判断的行为条件。
- **测试：** 单元、race、故障注入、服务级或 50-Agent 验证。
- **发布建议：** 是否阻断当前基线，以及项目负责人/发布负责人需要确认的风险。

## 12. 完成定义

本计划全部完成至少需要满足：

- 控制命令具备 current 代际发送线性化和明确失败语义。
- Agent 具备跨重连、Exec、进程和主机重启的 JobID 去重与 SHA256 恢复。
- 服务端 delivery/execution 分离，状态转换单调、终态粘滞、广播基于提交事实。
- 每 node 的 sequence 连续、缺口可恢复、取消可收口、数据库回滚不复用序列。
- stream mismatch、本地状态损坏和 unknown 都 fail closed 且可审计。
- 慢 dashboard、Dispatcher 和后台 worker 不阻塞控制面；shutdown 有界且无残留 worker 访问 Store。
- 相关现有测试、`go test -race ./...`、定向故障测试和 50-Agent 重连验证通过，或每个例外都有明确的发布负责人风险接受。

## 13. 非目标

- 本计划不直接修复源代码。
- 本计划不创建 GitHub issues。
- 本计划不替代安全专项、产品需求或一般性能压测。
- 本计划不把 at-most-once 副作用协议表述为原子 exactly-once。
- 本计划不替项目负责人或发布负责人做最终发布决定。
