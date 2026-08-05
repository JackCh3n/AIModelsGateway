# AI Models Gateway

统一管理多个 AI 中转站 / 官方站，通过单一网关对外提供 OpenAI 和 Anthropic 兼容接口。支持多 Key 轮询、协议自动转换、模型路由别名、用量统计等功能。

## 功能特性

- **多站点管理**：集中管理多个 AI 服务商（OpenAI 官方 / 中转站），支持启用/禁用、设为默认
- **多 Key 轮询**：每个站点可配置多个 API Key，自动轮询负载均衡
- **协议自动转换**：客户端用 OpenAI 格式请求，后端可对接 Anthropic 站点，反之亦然（支持流式/非流式）
- **模型管理**：支持手动添加 / 一键获取模型（通过 `/v1/models` 接口自动拉取）
- **模型路由别名**：客户端用固定模型名调用，网关自动路由到指定站点的指定模型
- **模型上下文配置**：为每个模型单独配置输入/输出 Token 预算，自动覆盖 `max_tokens`
- **用量统计**：基于 SQLite 持久化存储请求日志，支持分页查看、按站点/模型统计、自动清理 30 天前数据
- **打卡签到**：支持签到送积分的站点，一键打开签到链接并记录打卡时间
- **配置导入导出**：支持 JSON 配置文件导入导出、OpenCode 格式 JSON 粘贴导入
- **暗色主题**：支持白天/夜间主题切换

## 快速开始

### 环境要求

- Go 1.25+

### 编译运行

```bash
# 直接运行
go run . -port 3458

# 编译 + 启动 + 自动打开浏览器（Windows）
go run . -start -port 3458

# 使用 PowerShell 脚本（自动停止旧进程、编译、启动、打开浏览器）
.\build.ps1
```

### cnb.cool 云原生构建与 Release

仓库已托管至 [cnb.cool](https://cnb.cool)，内置 `.cnb.yml` 流水线，功能对齐 GitHub Actions：

- **推送 Tag**（如 `git tag v1.0.0 && git push origin v1.0.0`）：多平台交叉编译 → 自动创建 Release → 上传全部构建产物
- **推送 master 代码**：自动构建并生成时间戳版本 Release
- **手动触发**：在流水线页面点击运行，生成手动 Release

支持 Windows / Linux（含 loong64、ppc64le、riscv64、s390x）/ macOS / FreeBSD 共 12 个平台架构组合。
构建脚本见 `scripts/build-all.sh`、`scripts/gen-release-notes.sh`。

### 访问地址

启动后浏览器打开管理后台：

| 地址 | 说明 |
|------|------|
| `http://127.0.0.1:3458/admin/` | 管理后台 |
| `http://127.0.0.1:3458/v1/chat/completions` | OpenAI 兼容接口 |
| `http://127.0.0.1:3458/v1/messages` | Anthropic 兼容接口 |
| `http://127.0.0.1:3458/v1/models` | 模型列表 |
| `http://127.0.0.1:3458/health` | 健康检查 |

## 客户端接入配置

### OpenAI 格式（Trae / WorkBuddy 等）

```
Base URL: http://127.0.0.1:3458/v1
API Key:  在管理后台 API Keys 页面生成
Model:    all（使用站点默认模型）或指定具体模型名
```

### Anthropic 格式（Claude Code / Cline 等）

```
Base URL: http://127.0.0.1:3458
API Key:  在管理后台 API Keys 页面生成
Model:    all（使用站点默认模型）或指定具体模型名
```

### 指定站点调用

在 URL 路径中加入 `/p/{站点ID}` 即可指定站点：

```
OpenAI:    http://127.0.0.1:3458/v1/chat/completions/p/{站点ID}
Anthropic: http://127.0.0.1:3458/v1/messages/p/{站点ID}
```

## 使用指南

### 添加中转站

1. 进入管理后台 → 中转站管理 → 点击「+ 添加中转站」
2. 填写名称、协议格式（OpenAI / Anthropic）、Base URL
3. 添加 API Key（支持多个 Key 轮询）
4. 添加支持的模型：
   - 手动输入模型名，回车添加
   - 点击「🔌 一键获取模型」通过 `/v1/models` 接口自动拉取（仅 OpenAI 格式）
5. 点击「测试连接」验证配置是否正确
6. 保存

### 导入配置

- **导入配置文件**：支持导入之前导出的 JSON 配置文件
- **导入 OpenCode**：粘贴 OpenCode 格式的 JSON 配置（含 `provider.openai/anthropic`），自动解析并导入多个模型

### 模型路由别名

在「模型路由」页面添加别名，客户端用固定模型名调用网关，网关自动路由到指定站点的指定模型。切换实际模型时只需修改别名，无需改客户端配置。

### 模型上下文配置

为指定模型配置输入/输出 Token 预算。配置后网关转发时自动覆盖请求中的 `max_tokens`，留空则走客户端传的值。

## 项目结构

```
├── main.go         # 入口，启动参数解析
├── server.go       # HTTP 服务、路由注册、鉴权中间件
├── admin.go        # 管理后台 API 接口
├── admin_html.go   # 管理后台前端页面（内嵌 HTML/CSS/JS）
├── proxy.go        # 代理转发核心逻辑、流式处理
├── convert.go      # OpenAI ↔ Anthropic 协议转换
├── store.go        # 数据持久化（JSON 配置文件）
├── db.go           # SQLite 用量日志存储
├── types.go        # 数据结构定义
├── build.ps1       # Windows 编译启动脚本
└── data/           # 运行时数据（config.json + usage.db）
```

## 技术栈

- **后端**：Go 1.25+，纯标准库 + modernc.org/sqlite（纯 Go SQLite 驱动，无需 CGO）
- **前端**：内嵌单页 HTML，无构建依赖
- **存储**：站点/Key/别名配置使用 JSON 文件，用量日志使用 SQLite（WAL 模式）

## 并发性能

网关针对高并发场景做了以下优化，单机可支持 200~500+ 并发请求（实际瓶颈取决于上游 API 限速）：

### 已实现的优化

| 优化项 | 说明 |
|--------|------|
| **读写锁** | 配置读取使用 `sync.RWMutex`，请求路径（鉴权/路由/选 Key）全部走 RLock 并发读，仅管理操作持写锁 |
| **异步用量记录** | 请求路径不再同步写 `config.json`，用量日志通过 4096 缓冲 channel 异步写入 SQLite，批量提交（每 3 秒或满 200 条），不阻塞请求 |
| **HTTP 连接池** | 上游请求复用 TCP 连接：`MaxIdleConnsPerHost=100`（默认仅 2），`MaxIdleConns=500`，减少 TLS 握手开销 |
| **SQLite 连接池** | `SetMaxOpenConns(50)` + WAL 模式，读写不互斥 |
| **Server 超时** | `ReadHeaderTimeout=10s` 防 slowloris，`WriteTimeout=5m` 兼容流式，`IdleTimeout=120s` 及时回收空闲连接 |
| **优雅关闭** | 收到 SIGINT/SIGTERM 后先刷盘待写入日志，再等待活跃连接结束，避免数据丢失 |

### 并发瓶颈分析

- **上游 API 限速**：真正瓶颈是上游服务商的 QPS 限制，多 Key 轮询可分散压力
- **SQLite 写入**：已通过异步批量写入规避，写入不再阻塞请求
- **内存**：每条请求约 10~50KB 内存（含 body 缓冲），1000 并发约需 10~50MB

### Redis 支持（可选）

当前单机架构已能满足大多数场景。如需 **多实例水平扩展** 或 **分布式限流**，可引入 Redis：

| 场景 | Redis 用途 | 实现方式 |
|------|-----------|----------|
| 多实例部署 | 共享配置/用量 | 用 Redis 替代 `config.json` + SQLite 存储用量日志 |
| 分布式限流 | 按 Key/IP 限速 | 用 Redis INCR + EXPIRE 实现滑动窗口限流 |
| 分布式 Key 轮询 | 跨实例均衡 | 用 Redis 计数器替代内存 `keyRotation` |
| 共享 reasoning_content 缓存 | 跨实例缓存思维链 | 用 Redis 替代内存 `reasoningCache` |

引入 Redis 需新增依赖 `github.com/redis/go-redis/v9`，并在 `Settings` 中增加 Redis 连接配置。当前版本未集成 Redis，单机性能已足够；如需启用可按上述方案扩展。

## License

MIT
