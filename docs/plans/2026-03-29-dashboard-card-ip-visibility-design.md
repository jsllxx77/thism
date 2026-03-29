# Dashboard Card IP Visibility Design

## Goal

让管理员可以在后台配置首页节点卡片是否显示 IP，并且只影响首页卡片视图，不影响节点表格和节点详情页。

## Context

- `NodeCard` 已经支持 `showIP`，当前默认显示，只有 `guest` 访问模式会隐藏。
- 首页 `Dashboard` 现在直接根据 `accessMode !== "guest"` 传递 `showIP`，没有独立后台开关。
- 后端已经有通用的 `app_settings` 存储模式，以及 `/api/settings/*` 的读写接口风格。
- 设置页已经有多张独立设置卡片，适合追加一个展示偏好卡片，而不是把该选项塞进无关设置里。

## Chosen Approach

### Setting model

新增一组独立的仪表盘展示设置，首个字段为：

- `show_dashboard_card_ip: boolean`

默认值为 `true`，保持当前行为不变。

### Storage

复用 `app_settings` 表，但使用独立 key 持久化，不混入通知设置或权限模型。

这样有几个好处：

- 语义清晰，表示的是展示偏好，不是访问控制。
- 后续可继续扩展首页卡片的显示项，而不污染其他设置结构。
- 默认值与回退逻辑可以集中处理。

### API

新增一组后台设置接口：

- `GET /api/settings/dashboard`
- `PUT /api/settings/dashboard`

读取接口允许 viewer 访问，这样首页和设置页都能拿到相同配置；更新接口仍然只允许 admin。

### UI

在设置页新增一张轻量卡片，放在现有设置卡片区域内，使用当前工程档案 / 控制台视觉系统。

卡片包含：

- 标题和说明文案
- 一个 “首页节点卡片显示 IP” 开关
- 保存按钮
- 成功 / 失败反馈

不改变设置页其他结构和行为。

### Dashboard behavior

首页 `Dashboard` 初始化时请求仪表盘设置，并把 `show_dashboard_card_ip` 与当前 `accessMode` 共同折算为 `NodeCard.showIP`：

- `guest` 仍然始终隐藏
- admin 在设置开启时显示，关闭时隐藏

为避免配置已关闭时页面刷新出现短暂泄露，设置加载完成前卡片先按隐藏处理。

## Non-goals

- 不改变节点表格中的 IP 展示
- 不改变节点详情页中的 IP 展示
- 不改变 guest 访问权限范围
- 不引入更细粒度的节点级显示策略

## Why this approach

- 与现有 `NodeCard` 能力直接对接，改动路径短且明确。
- 与已有 `app_settings` 和设置页卡片结构一致，维护成本低。
- 将“展示偏好”和“访问权限”分离，避免未来行为混乱。

## Testing strategy

- 为 store 增加仪表盘设置默认值与读写往返测试
- 为 API 增加 dashboard settings 的 GET/PUT 接口测试
- 为设置页增加新卡片加载 / 保存测试
- 为首页增加配置开启与关闭时的 IP 展示传递测试
- 保留并继续覆盖 guest 模式下始终隐藏 IP 的行为
