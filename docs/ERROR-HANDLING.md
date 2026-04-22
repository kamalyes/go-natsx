# 错误处理

go-natsx 定义了结构化的错误变量，支持 `errors.Is` 检查

## 错误变量

| 错误 | 说明 |
|:-----|:-----|
| `ErrNotConnected` | 未连接到 NATS 服务器 |
| `ErrAlreadyClosed` | 客户端已关闭 |
| `ErrInvalidSubject` | Subject 无效（空字符串） |
| `ErrInvalidMessage` | 消息无效 |
| `ErrPublishFailed` | 消息发布失败 |
| `ErrSubscribeFailed` | 订阅失败 |
| `ErrJetStreamFailed` | JetStream 操作失败 |
| `ErrBucketNotFound` | KV Bucket 未找到 |
| `ErrKeyNotFound` | KV Key 未找到 |
| `ErrTimeout` | 操作超时 |
| `ErrUnavailable` | 服务不可用 |
| `ErrGlobalPoolNotInitialized` | 全局消费者池未初始化 |

## 错误检查

### 使用 errors.Is

```go
err := natsx.Subscribe[Event](client, "topic", "svc", handler, natsx.WithIntoGlobalPool())
if err != nil {
    if errors.Is(err, natsx.ErrGlobalPoolNotInitialized) {
        client.InitWorkerPool(10, 1000)
        // 重试订阅
    }
}
```

### 发布错误

```go
err := client.Publish(ctx, "order.created", data)
if err != nil {
    switch {
    case errors.Is(err, natsx.ErrNotConnected):
        log.Println("NATS 连接未建立")
    case errors.Is(err, natsx.ErrInvalidSubject):
        log.Println("Subject 不能为空")
    case errors.Is(err, natsx.ErrPublishFailed):
        log.Println("发布失败:", err)
    }
}
```

### 订阅错误

```go
err := natsx.Subscribe[Event](client, "topic", "svc", handler)
if err != nil {
    switch {
    case errors.Is(err, natsx.ErrNotConnected):
        log.Println("NATS 连接未建立")
    case errors.Is(err, natsx.ErrSubscribeFailed):
        log.Println("订阅失败:", err)
    case errors.Is(err, natsx.ErrGlobalPoolNotInitialized):
        log.Println("请先调用 InitWorkerPool")
    }
}
```

### JetStream 错误

```go
err := client.PublishJetStream(ctx, "topic", data)
if err != nil {
    if errors.Is(err, natsx.ErrJetStreamFailed) {
        log.Println("JetStream 未启用")
    }
}
```

## 消息处理错误

在 JetStream 模式下，`handleFunc` 返回的错误会影响消息确认：

```go
natsx.Subscribe[OrderEvent](client, "order.created", "svc",
    func(evt *OrderEvent) error {
        if err := processOrder(evt); err != nil {
            // 返回 error → 消息 NAK，等待重试
            return err
        }
        // 返回 nil → 消息 ACK
        return nil
    },
    natsx.WithMsgMaxRetry(3),
    natsx.WithMsgRetryInterval(time.Second),
)
```

### 反序列化错误

消息反序列化失败时：
- Core NATS 模式：记录错误日志，消息被丢弃
- JetStream 模式：消息 NAK，等待重试

### 超过重试次数

超过 `MsgMaxRetry` 后，消息被 Term，不再重试：

```go
nakMsg(c, msg, msgMaxRetry, msgRetryInterval)
// 内部逻辑：
// 1. 检查消息投递次数
// 2. 如果超过 msgMaxRetry → msg.Term()
// 3. 否则 → msg.NakWithDelay(msgRetryInterval)
```
