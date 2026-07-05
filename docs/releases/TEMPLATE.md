# {{VERSION}}

## English

Release {{VERSION}}.

### Changes

- Write a short user-facing summary of the release.
- Replace these bullets before tagging.

### Assets

- Signed linux/amd64 and linux/arm64 agent binaries are attached.
- Signature and version sidecar files are attached for each agent binary.
- Container images are published to GHCR as {{VERSION}}, sha-{{SHORT_SHA}}, and latest.

## 中文

发布 {{VERSION}}。

### 更新内容

- 用中文面向用户概括本次发布内容。
- 推送标签前替换这些示例条目，不要直接复制英文 commit 标题。

### 发布产物

- 已附带签名的 linux/amd64 和 linux/arm64 agent 二进制文件。
- 每个 agent 二进制文件都附带对应的签名和版本 sidecar 文件。
- 容器镜像已发布到 GHCR，标签包括 {{VERSION}}、sha-{{SHORT_SHA}} 和 latest。
