/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-04-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-04-23 00:00:00
 * @FilePath: \go-natsx\subscribe_test.go
 * @Description: go-natsx 订阅功能单元测试
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package natsx

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

// TestNormalizeConsumerName 测试消费者名称规范化
func TestNormalizeConsumerName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"dots to underscores", "user.login.testing", "user_login_testing"},
		{"no dots", "user_login", "user_login"},
		{"single dot", "user.login", "user_login"},
		{"empty string", "", ""},
		{"multiple consecutive dots", "user..login", "user__login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeConsumerName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultSubscribeOptions 测试默认订阅选项
func TestDefaultSubscribeOptions(t *testing.T) {
	opts := DefaultSubscribeOptions()
	assert.False(t, opts.IsListenBroadcast)
	assert.False(t, opts.IsIntoGlobalPool)
	assert.Equal(t, 1, opts.LocalPoolSize)
	assert.Equal(t, 100, opts.LocalPoolQueueSize)
	assert.Equal(t, 100, opts.BatchSize)
	assert.Equal(t, 10*time.Second, opts.MaxWait)
	assert.Equal(t, uint64(3), opts.MsgMaxRetry)
	assert.Equal(t, 1*time.Second, opts.MsgRetryInterval)
	assert.Equal(t, 30*time.Second, opts.MaxAckWait)
	assert.False(t, opts.ConsumeFastest)
	assert.False(t, opts.EnabledFlowControl)
}

// TestSubscribeOptions_WithListenBroadcast 测试广播选项
func TestSubscribeOptions_WithListenBroadcast(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithListenBroadcast()(&opts)
	assert.True(t, opts.IsListenBroadcast)
}

// TestSubscribeOptions_WithIntoGlobalPool 测试全局消费者池选项
func TestSubscribeOptions_WithIntoGlobalPool(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithIntoGlobalPool()(&opts)
	assert.True(t, opts.IsIntoGlobalPool)
}

// TestSubscribeOptions_WithLocalPoolSize 测试局部消费者池大小选项
func TestSubscribeOptions_WithLocalPoolSize(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithLocalPoolSize(5, 200)(&opts)
	assert.Equal(t, 5, opts.LocalPoolSize)
	assert.Equal(t, 200, opts.LocalPoolQueueSize)
}

// TestSubscribeOptions_WithBatchSize 测试批量大小选项
func TestSubscribeOptions_WithBatchSize(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithBatchSize(50)(&opts)
	assert.Equal(t, 50, opts.BatchSize)
}

// TestSubscribeOptions_WithMaxWait 测试最大等待时间选项
func TestSubscribeOptions_WithMaxWait(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithMaxWait(5 * time.Second)(&opts)
	assert.Equal(t, 5*time.Second, opts.MaxWait)
}

// TestSubscribeOptions_WithMsgMaxRetry 测试消息最大重试次数选项
func TestSubscribeOptions_WithMsgMaxRetry(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithMsgMaxRetry(10)(&opts)
	assert.Equal(t, uint64(10), opts.MsgMaxRetry)
}

// TestSubscribeOptions_WithMsgRetryInterval 测试消息重试间隔选项
func TestSubscribeOptions_WithMsgRetryInterval(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithMsgRetryInterval(2 * time.Second)(&opts)
	assert.Equal(t, 2*time.Second, opts.MsgRetryInterval)
}

// TestSubscribeOptions_WithMaxAckWait 测试最大 ACK 等待时间选项
func TestSubscribeOptions_WithMaxAckWait(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithMaxAckWait(60 * time.Second)(&opts)
	assert.Equal(t, 60*time.Second, opts.MaxAckWait)
}

// TestSubscribeOptions_WithIdleHeartbeat 测试心跳时间选项
func TestSubscribeOptions_WithIdleHeartbeat(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithIdleHeartbeat(3 * time.Second)(&opts)
	assert.Equal(t, 3*time.Second, opts.IdleHeartbeat)
}

// TestSubscribeOptions_WithEnableFlowControl 测试流控选项
func TestSubscribeOptions_WithEnableFlowControl(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithEnableFlowControl()(&opts)
	assert.True(t, opts.EnabledFlowControl)
}

// TestSubscribeOptions_WithConsumeFastest 测试尽快消费选项
func TestSubscribeOptions_WithConsumeFastest(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithConsumeFastest(true)(&opts)
	assert.True(t, opts.ConsumeFastest)

	WithConsumeFastest(false)(&opts)
	assert.False(t, opts.ConsumeFastest)
}

// TestSubscribeOptions_Chained 测试链式组合多个选项
func TestSubscribeOptions_Chained(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithBatchSize(50)(&opts)
	WithMsgMaxRetry(5)(&opts)
	WithMsgRetryInterval(2 * time.Second)(&opts)
	WithMaxAckWait(60 * time.Second)(&opts)
	WithLocalPoolSize(3, 50)(&opts)

	assert.Equal(t, 50, opts.BatchSize)
	assert.Equal(t, uint64(5), opts.MsgMaxRetry)
	assert.Equal(t, 2*time.Second, opts.MsgRetryInterval)
	assert.Equal(t, 60*time.Second, opts.MaxAckWait)
	assert.Equal(t, 3, opts.LocalPoolSize)
	assert.Equal(t, 50, opts.LocalPoolQueueSize)
}

// TestSubscribeOptions_WithContextInjector 测试消息级上下文注入器选项
func TestSubscribeOptions_WithContextInjector(t *testing.T) {
	opts := DefaultSubscribeOptions()
	assert.Nil(t, opts.ContextInjector)

	type ctxKey struct{}
	inj := func(ctx context.Context, msg *nats.Msg) context.Context {
		return context.WithValue(ctx, ctxKey{}, "injected")
	}
	WithContextInjector(inj)(&opts)
	assert.NotNil(t, opts.ContextInjector)
}

// TestSubscribeOptions_WithUnlimitedDelivery 测试无限重投模式
func TestSubscribeOptions_WithUnlimitedDelivery(t *testing.T) {
	opts := DefaultSubscribeOptions()
	assert.Equal(t, uint64(3), opts.MsgMaxRetry)

	WithUnlimitedDelivery()(&opts)
	assert.Equal(t, uint64(0), opts.MsgMaxRetry, "unlimited delivery should set MsgMaxRetry to 0")

	// 与 WithMsgMaxRetry 的覆盖顺序：后调用者生效
	WithMsgMaxRetry(5)(&opts)
	assert.Equal(t, uint64(5), opts.MsgMaxRetry)
	WithUnlimitedDelivery()(&opts)
	assert.Equal(t, uint64(0), opts.MsgMaxRetry)
}

// TestSubscribeOptions_WithRetryBackoff 测试指数退避选项
func TestSubscribeOptions_WithRetryBackoff(t *testing.T) {
	opts := DefaultSubscribeOptions()
	assert.Nil(t, opts.RetryBackoff)

	WithRetryBackoff(Backoff{Base: time.Second, Max: 30 * time.Second})(&opts)
	assert.NotNil(t, opts.RetryBackoff)
	assert.Equal(t, time.Second, opts.RetryBackoff.Base)
	assert.Equal(t, 30*time.Second, opts.RetryBackoff.Max)
}

// TestBackoffDelayFor 测试指数退避延迟计算
func TestBackoffDelayFor(t *testing.T) {
	backoff := Backoff{Base: 2 * time.Second, Max: 30 * time.Second, Factor: 2.0}

	assert.Equal(t, 2*time.Second, backoff.delayFor(1), "first delivery failure should return Base")
	assert.Equal(t, 4*time.Second, backoff.delayFor(2), "second failure should double")
	assert.Equal(t, 8*time.Second, backoff.delayFor(3))
	assert.Equal(t, 30*time.Second, backoff.delayFor(10), "should cap at Max")
	assert.Equal(t, 30*time.Second, backoff.delayFor(100), "should stay capped far beyond Max")

	// Factor 未设置时默认 2.0
	defaultFactor := Backoff{Base: time.Second, Max: 10 * time.Second}
	assert.Equal(t, 4*time.Second, defaultFactor.delayFor(3), "default factor should be 2.0")

	// Base 未设置时禁用退避
	noBase := Backoff{Max: time.Second}
	assert.Equal(t, time.Duration(0), noBase.delayFor(5), "zero Base should disable backoff")

	// 抖动模式下延迟不超过无抖动值
	jittered := Backoff{Base: 10 * time.Second, Max: 10 * time.Second, Jitter: true}
	for i := 0; i < 100; i++ {
		delay := jittered.delayFor(3)
		assert.GreaterOrEqual(t, delay, time.Duration(0))
		assert.LessOrEqual(t, delay, 10*time.Second, "jittered delay must not exceed capped value")
	}
}

// TestDeliveryCount 测试投递次数读取（非 JetStream 消息返回 0）
func TestDeliveryCount(t *testing.T) {
	assert.Equal(t, uint64(0), DeliveryCount(nil), "nil msg should return 0")

	msg := &nats.Msg{Subject: "test.subject", Data: []byte(`{}`)}
	// Core NATS 消息没有 JetStream 元数据（reply 为空且非 ACK subject）
	assert.Equal(t, uint64(0), DeliveryCount(msg), "core NATS msg without metadata should return 0")
}

// TestRetryDelayPriority 测试延迟计算优先级：退避策略 > 固定间隔
func TestRetryDelayPriority(t *testing.T) {
	msg := &nats.Msg{Subject: "test.subject"}

	// 均未设置：立即重投
	assert.Equal(t, time.Duration(0), retryDelay(SubscribeOptions{}, msg))

	// 仅固定间隔
	assert.Equal(t, 3*time.Second, retryDelay(SubscribeOptions{MsgRetryInterval: 3 * time.Second}, msg))

	// 退避优先于固定间隔
	subOpts := SubscribeOptions{
		MsgRetryInterval: 3 * time.Second,
		RetryBackoff:     &Backoff{Base: time.Second, Max: time.Minute},
	}
	assert.Equal(t, time.Second, retryDelay(subOpts, msg), "backoff should take precedence over fixed interval")
}

// TestDeriveMessageContext_InjectorAndDeadline 测试消息级 ctx 派生：注入器生效 + AckWait 对齐 deadline
func TestDeriveMessageContext_InjectorAndDeadline(t *testing.T) {
	type ctxKey struct{}

	subOpts := DefaultSubscribeOptions()
	subOpts.MaxAckWait = 50 * time.Millisecond
	subOpts.ContextInjector = func(ctx context.Context, msg *nats.Msg) context.Context {
		return context.WithValue(ctx, ctxKey{}, msg.Subject)
	}

	msg := &nats.Msg{Subject: "test.subject"}
	derived, cancel := deriveMessageContext(context.Background(), subOpts, msg)
	defer cancel()

	// 注入器的值可见
	assert.Equal(t, "test.subject", derived.Value(ctxKey{}))
	// deadline 与 AckWait 对齐（近似断言，容忍调度误差）
	dl, ok := derived.Deadline()
	assert.True(t, ok, "message ctx should carry a deadline aligned with MaxAckWait")
	assert.WithinDuration(t, time.Now().Add(50*time.Millisecond), dl, 20*time.Millisecond)

	// 超时后 ctx 取消（模拟「处理慢于重投窗口」的快速失败）
	time.Sleep(60 * time.Millisecond)
	assert.ErrorIs(t, derived.Err(), context.DeadlineExceeded)
}

// TestDeriveMessageContext_BaseContextPropagation 测试订阅级 base ctx 的取消向下传播
func TestDeriveMessageContext_BaseContextPropagation(t *testing.T) {
	subOpts := DefaultSubscribeOptions()
	subOpts.MaxAckWait = 0 // 无 AckWait 时退化为 WithCancel，仅随 base ctx 取消

	baseCtx, baseCancel := context.WithCancel(context.Background())
	derived, cancel := deriveMessageContext(baseCtx, subOpts, nil)
	defer cancel()

	baseCancel()
	assert.ErrorIs(t, derived.Err(), context.Canceled)
}

// TestSubscribe_InjectorReceivesMessage 测试注入器收到真实消息（Core NATS 路径）
func TestSubscribe_InjectorReceivesMessage(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.sub.injector")
	var injectedSubject atomic.Value
	var received atomic.Int32

	err := Subscribe(context.Background(), client, subject, "testing",
		func(ctx context.Context, evt *TestEvent) error {
			received.Add(1)
			return nil
		},
		WithContextInjector(func(ctx context.Context, msg *nats.Msg) context.Context {
			injectedSubject.Store(msg.Subject)
			return ctx
		}),
	)
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "hello"})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), received.Load())
	assert.Equal(t, subject, injectedSubject.Load(), "injector should receive the raw nats message")
}

// TestSubscribe_HandlerContextCancelledOnTimeout 测试 handler ctx 超时快速失败（Core NATS 路径退化为超时熔断）
func TestSubscribe_HandlerContextCancelledOnTimeout(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.sub.deadline")
	var ctxErr atomic.Value

	err := Subscribe(context.Background(), client, subject, "testing",
		func(ctx context.Context, evt *TestEvent) error {
			<-ctx.Done() // 模拟慢处理，等待库派生的 deadline 触发
			ctxErr.Store(ctx.Err())
			return ctx.Err()
		},
		WithMaxAckWait(100*time.Millisecond),
	)
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "slow"})
	assert.NoError(t, err)

	time.Sleep(300 * time.Millisecond)
	assert.ErrorIs(t, ctxErr.Load().(error), context.DeadlineExceeded,
		"handler ctx should be cancelled after MaxAckWait deadline")
}

// TestSubscribe_BroadcastOverridesPoolSettings 测试广播模式在 Subscribe 内部覆盖池设置
func TestSubscribe_BroadcastOverridesPoolSettings(t *testing.T) {
	opts := DefaultSubscribeOptions()
	WithListenBroadcast()(&opts)
	WithIntoGlobalPool()(&opts)
	WithLocalPoolSize(10, 500)(&opts)

	assert.True(t, opts.IsListenBroadcast)
	assert.True(t, opts.IsIntoGlobalPool, "option function sets global pool, Subscribe() overrides it internally")
	assert.Equal(t, 10, opts.LocalPoolSize, "option function sets pool size, Subscribe() overrides it internally")
}

// TestSubscribe_GlobalPoolNotInitialized 测试全局消费者池未初始化
func TestSubscribe_GlobalPoolNotInitialized(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string
	}

	err := Subscribe(context.Background(), client, "test.event", "testing", func(ctx context.Context, evt *TestEvent) error {
		return nil
	}, WithIntoGlobalPool())
	assert.ErrorIs(t, err, ErrGlobalPoolNotInitialized)
}

// TestSubscribe_NotConnected 测试未连接时订阅
func TestSubscribe_NotConnected(t *testing.T) {
	client := &Client{logger: NewDefaultLogger()}

	type TestEvent struct {
		Name string
	}

	err := Subscribe(context.Background(), client, "test.event", "testing", func(ctx context.Context, evt *TestEvent) error {
		return nil
	})
	assert.ErrorIs(t, err, ErrNotConnected)
}

// TestSubscribe_Success_CoreNATS 测试 Core NATS 普通订阅成功
func TestSubscribe_Success_CoreNATS(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.sub")

	err := Subscribe(context.Background(), client, subject, "testing", func(ctx context.Context, evt *TestEvent) error {
		return nil
	})
	assert.NoError(t, err)
}

// TestSubscribe_ReceiveMessage 测试订阅并接收消息
func TestSubscribe_ReceiveMessage(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.sub.recv")
	var received atomic.Int32

	err := Subscribe(context.Background(), client, subject, "testing", func(ctx context.Context, evt *TestEvent) error {
		received.Add(1)
		return nil
	})
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "hello"})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), received.Load())
}

// TestSubscribeBroadcast_Success_CoreNATS 测试 Core NATS 广播订阅成功
func TestSubscribeBroadcast_Success_CoreNATS(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.broadcast")

	err := SubscribeBroadcast(context.Background(), client, subject, func(ctx context.Context, evt *TestEvent) error {
		return nil
	})
	assert.NoError(t, err)
}

// TestSubscribeBroadcast_ReceiveMessage 测试广播订阅接收消息
func TestSubscribeBroadcast_ReceiveMessage(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.broadcast.recv")
	var received atomic.Int32

	err := SubscribeBroadcast(context.Background(), client, subject, func(ctx context.Context, evt *TestEvent) error {
		received.Add(1)
		return nil
	})
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "hello"})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), received.Load())
}

// TestSubscribeBroadcast_MultipleSubscribers 测试广播模式下多个订阅者都收到消息
func TestSubscribeBroadcast_MultipleSubscribers(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.broadcast.multi")
	var received1, received2 atomic.Int32

	err := SubscribeBroadcast(context.Background(), client, subject, func(ctx context.Context, evt *TestEvent) error {
		received1.Add(1)
		return nil
	})
	assert.NoError(t, err)

	err = SubscribeBroadcast(context.Background(), client, subject, func(ctx context.Context, evt *TestEvent) error {
		received2.Add(1)
		return nil
	})
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "hello"})
	assert.NoError(t, err)

	time.Sleep(300 * time.Millisecond)
	assert.Equal(t, int32(1), received1.Load(), "subscriber 1 should receive message")
	assert.Equal(t, int32(1), received2.Load(), "subscriber 2 should receive message")
}

// TestSubscribeStreamBatch_JetStreamNotEnabled 测试未启用 JetStream 时批量订阅
func TestSubscribeStreamBatch_JetStreamNotEnabled(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string
	}

	err := SubscribeStreamBatch(context.Background(), client, "test.event", "testing", func(ctx context.Context, evts []*TestEvent) error {
		return nil
	})
	assert.ErrorIs(t, err, ErrJetStreamFailed)
}

// TestSubscribeStreamBatch_GlobalPoolNotInitialized 测试批量订阅全局池未初始化
func TestSubscribeStreamBatch_GlobalPoolNotInitialized(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string
	}

	err := SubscribeStreamBatch(context.Background(), client, "test.event", "testing", func(ctx context.Context, evts []*TestEvent) error {
		return nil
	}, WithIntoGlobalPool())
	assert.ErrorIs(t, err, ErrGlobalPoolNotInitialized)
}

// TestSubscribeStreamBatch_Success 测试批量流式消费成功
func TestSubscribeStreamBatch_Success(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	streamName := "TEST_BATCH_SUB"
	ensureStream(t, client, streamName)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := streamName + ".test"
	var received atomic.Int32

	err := SubscribeStreamBatch(context.Background(), client, subject, "testing", func(ctx context.Context, evts []*TestEvent) error {
		received.Add(int32(len(evts)))
		return nil
	}, WithBatchSize(10), WithMaxWait(2*time.Second))
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := client.PublishJetStream(context.Background(), subject, []byte(`{"name":"test"}`))
		assert.NoError(t, err)
	}

	time.Sleep(3 * time.Second)
	assert.Greater(t, received.Load(), int32(0), "should receive at least one message")
}

// TestSubscribe_WithGlobalPool 测试使用全局消费者池订阅
func TestSubscribe_WithGlobalPool(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	client.InitWorkerPool(5, 100)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.global.pool")
	var received atomic.Int32

	err := Subscribe(context.Background(), client, subject, "testing", func(ctx context.Context, evt *TestEvent) error {
		received.Add(1)
		return nil
	}, WithIntoGlobalPool())
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "hello"})
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), received.Load())
}

// TestSubscribe_MultipleOptions 测试多个选项组合
func TestSubscribe_MultipleOptions(t *testing.T) {
	client, conn := newConnectedClient(t)
	defer client.Close()
	defer conn.Close()

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := uniqueSubject("test.opts")

	err := Subscribe(context.Background(), client, subject, "testing", func(ctx context.Context, evt *TestEvent) error {
		return nil
	},
		WithLocalPoolSize(3, 50),
		WithMsgMaxRetry(5),
		WithMsgRetryInterval(2*time.Second),
		WithMaxAckWait(60*time.Second),
	)
	assert.NoError(t, err)
}

// TestSubscribe_WithJetStream 测试启用 JetStream 时订阅
func TestSubscribe_WithJetStream(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	streamName := "TEST_SUB_JS"
	ensureStream(t, client, streamName)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := streamName + ".test"

	err := Subscribe(context.Background(), client, subject, "testing", func(ctx context.Context, evt *TestEvent) error {
		return nil
	}, WithMaxAckWait(30*time.Second))
	assert.NoError(t, err)
}

// TestSubscribeBroadcast_WithJetStream 测试启用 JetStream 时广播订阅
func TestSubscribeBroadcast_WithJetStream(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	streamName := "TEST_BROADCAST_JS"
	ensureStream(t, client, streamName)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := streamName + ".test"

	err := SubscribeBroadcast(context.Background(), client, subject, func(ctx context.Context, evt *TestEvent) error {
		return nil
	}, WithMaxAckWait(30*time.Second), WithIdleHeartbeat(5*time.Second))
	assert.NoError(t, err)
}

// TestSubscribe_ErrPermanent_TerminatesMessage 测试 ErrPermanent 哨兵错误触发 Term（不再重投）
func TestSubscribe_ErrPermanent_TerminatesMessage(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	streamName := "TEST_PERMANENT_JS"
	ensureStream(t, client, streamName)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := streamName + ".permanent"
	var attempts atomic.Int32

	err := Subscribe(context.Background(), client, subject, "testing_permanent",
		func(ctx context.Context, evt *TestEvent) error {
			attempts.Add(1)
			// 模拟业务上无法匹配的场景：声明永久性失败，库应 Term 终止而非 Nak 重投
			return fmt.Errorf("%w: order not found for %s", ErrPermanent, evt.Name)
		},
		WithMaxAckWait(5*time.Second),
		WithUnlimitedDelivery(), // 无限重投模式下 ErrPermanent 仍应 Term
	)
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "orphan"})
	assert.NoError(t, err)

	// 等待足够多的潜在重投窗口（若有 Nak 重投，attempts 会持续增长）
	time.Sleep(2 * time.Second)
	assert.LessOrEqual(t, attempts.Load(), int32(1),
		"ErrPermanent should terminate the message, no redelivery expected (attempts=%d)", attempts.Load())
}

// TestSubscribe_TemporaryError_RetriesWithBackoff 测试临时错误按退避策略 Nak 重投
func TestSubscribe_TemporaryError_RetriesWithBackoff(t *testing.T) {
	client, conn := newConnectedClientWithJS(t)
	defer client.Close()
	defer conn.Close()

	streamName := "TEST_BACKOFF_JS"
	ensureStream(t, client, streamName)

	type TestEvent struct {
		Name string `json:"name"`
	}

	subject := streamName + ".backoff"
	var attempts atomic.Int32
	var firstDelivery atomic.Int64

	err := Subscribe(context.Background(), client, subject, "testing_backoff",
		func(ctx context.Context, evt *TestEvent) error {
			if attempts.Add(1) == 1 {
				firstDelivery.Store(time.Now().UnixMilli())
				return errors.New("transient: db lock timeout") // 临时错误：应 Nak 重投
			}
			return nil // 第二次成功
		},
		WithMaxAckWait(5*time.Second),
		WithRetryBackoff(Backoff{Base: 200 * time.Millisecond, Max: time.Second}),
	)
	assert.NoError(t, err)

	err = PublishEvent(client, subject, &TestEvent{Name: "retry-me"})
	assert.NoError(t, err)

	// 等待重投 + 二次成功
	time.Sleep(2 * time.Second)
	assert.GreaterOrEqual(t, attempts.Load(), int32(2),
		"transient error should be redelivered via Nak (attempts=%d)", attempts.Load())
}
