# go-natsx

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kamalyes/go-natsx)](https://github.com/kamalyes/go-natsx)
[![GoDoc](https://godoc.org/github.com/kamalyes/go-natsx?status.svg)](https://godoc.org/github.com/kamalyes/go-natsx)
[![License](https://img.shields.io/github/license/kamalyes/go-natsx)](https://github.com/kamalyes/go-natsx/blob/main/LICENSE)

一个功能丰富、高性能的 Go 语言 NATS 客户端易用性封装库，提供泛型事件订阅、广播、批量流式消费和消费者池管理

## 📖 特性 & 文档导航

| 特性 | 说明 | 文档 |
|:-----|:-----|:----:|
| 🔌 **客户端封装** | 基于 `*nats.Conn` 的轻量封装，不管理连接生命周期 | [📘 快速开始](docs/QUICKSTART.md) |
| 📨 **消息发布** | 泛型事件发布、带重试发布、请求-响应模式 | [📙 消息发布](docs/PUBLISH.md) |
| 📬 **普通订阅** | QueueSubscribe 负载均衡模式，泛型自动反序列化 | [📗 事件订阅](docs/SUBSCRIBE.md) |
| 📡 **广播订阅** | 所有订阅者都收到消息，适合事件通知 | [📕 广播订阅](docs/BROADCAST.md) |
| 📦 **批量流式消费** | JetStream Pull 模式批量拉取，支持批量大小和等待时间 | [📓 批量消费](docs/STREAM-BATCH.md) |
| 🏊 **消费者池** | 局部消费者池和全局消费者池，基于 go-toolbox WorkerPool | [🏊 消费者池](docs/CONSUMER-POOL.md) |
| ⚙️ **可配置参数** | 重试次数、批量大小、等待时间、ACK 超时等 | [⚙️ 配置选项](docs/OPTIONS.md) |
| 🔗 **JetStream** | 原生 JetStream 支持，持久化消息和流处理 | [🔗 JetStream](docs/JETSTREAM.md) |
| 🛡️ **错误处理** | 结构化错误定义，支持错误检查 | [🛡️ 错误处理](docs/ERROR-HANDLING.md) |

> 📖 **完整文档**：查看 [文档中心](docs/README.md) 了解所有功能和学习路径

## 📦 安装

```bash
go get github.com/kamalyes/go-natsx
```

## 🚀 快速开始

```go
import (
    "github.com/nats-io/nats.go"
    natsx "github.com/kamalyes/go-natsx"
)

// 1. 创建 NATS 连接（由调用方管理连接生命周期）
conn, _ := nats.Connect("nats://127.0.0.1:4222")

// 2. 创建客户端
client, _ := natsx.NewClient(conn)
defer client.Close()

// 3. 发布事件
natsx.PublishEvent(client, "user.created", &UserCreated{Name: "张三"})

// 4. 订阅事件（泛型自动反序列化）
natsx.Subscribe[UserCreated](client, "user.created", "order-service", func(evt *UserCreated) error {
    fmt.Println("User created:", evt.Name)
    return nil
})
```

> 💡 **详细教程**：查看 [📘 快速入门文档](docs/QUICKSTART.md) 了解完整的安装和使用步骤

## 🏗️ 核心特性

### 泛型事件订阅

```go
// 普通订阅 - QueueSubscribe 负载均衡
natsx.Subscribe[OrderEvent](client, "order.created", "payment-service", func(evt *OrderEvent) error {
    return processPayment(evt)
})

// 广播订阅 - 所有订阅者都收到消息
natsx.SubscribeBroadcast[UserEvent](client, "user.updated", func(evt *UserEvent) error {
    return refreshCache(evt)
})

// 批量流式消费 - JetStream Pull 模式
natsx.SubscribeStreamBatch[LogEvent](client, "logs.batch", "analytics", func(evts []*LogEvent) error {
    return batchInsert(evts)
}, natsx.WithBatchSize(100), natsx.WithMaxWait(5*time.Second))
```

### 消费者池

```go
// 全局消费者池 - 所有订阅共享
client.InitWorkerPool(10, 1000)
natsx.Subscribe[Event](client, "topic", "svc", handler, natsx.WithIntoGlobalPool())

// 局部消费者池 - 每个订阅独立
natsx.Subscribe[Event](client, "topic", "svc", handler, natsx.WithLocalPoolSize(5, 200))
```

### JetStream 支持

```go
client.EnableJetStream()

// 发布持久化消息
ack, _ := client.PublishJetStream(ctx, "order.created", data)

// 订阅 JetStream 消息（自动 ACK）
natsx.Subscribe[OrderEvent](client, "order.created", "svc", handler,
    natsx.WithMaxAckWait(30*time.Second),
    natsx.WithMsgMaxRetry(5),
)
```

> 📖 **详细说明**：查看 [🏊 消费者池](docs/CONSUMER-POOL.md) 和 [🔗 JetStream](docs/JETSTREAM.md) 了解更多配置

## 🧪 测试

```bash
# 启动 NATS 服务器（启用 JetStream）
nats-server -js

# 运行测试
go test ./... -v
go test ./... -cover
go test -v -run TestSubscribe
```

## 📚 相关资源

- 📖 [完整文档中心](docs/README.md) - 所有功能文档和学习路径
- 🐛 [问题反馈](https://github.com/kamalyes/go-natsx/issues) - 报告 bug 或提出建议
- 💬 [讨论区](https://github.com/kamalyes/go-natsx/discussions) - 技术交流

## 📦 依赖

- [nats.go](https://github.com/nats-io/nats.go) - NATS 官方 Go 客户端
- [go-toolbox](https://github.com/kamalyes/go-toolbox) - WorkerPool 和重试工具
- [go-logger](https://github.com/kamalyes/go-logger) - 结构化日志
- [json-iterator](https://github.com/json-iterator/go) - 高性能 JSON 序列化

## 🤝 贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m '✨ feat: Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 📋 Git Commit Emoji 规范

<details>
<summary>点击展开 Emoji 规范表</summary>

| Emoji | 类型 | 说明 |
|:-----:|------|------|
| ✨ | feat | 新功能 |
| 🐛 | fix | 修复 bug |
| 📝 | docs | 文档更新 |
| ♻️ | refactor | 代码重构 |
| ⚡ | perf | 性能优化 |
| ✅ | test | 测试相关 |
| 🔧 | chore | 配置/构建 |
| 🚀 | deploy | 部署发布 |
| 🔒 | security | 安全修复 |
| 🔥 | remove | 删除代码 |

**示例：** `git commit -m "✨ feat(subscribe): 新增批量流式消费"`

</details>

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 👨‍💻 作者

Kamal Yang ([@kamalyes](https://github.com/kamalyes))
