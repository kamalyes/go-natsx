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
	"sync/atomic"
	"testing"
	"time"

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

	err := Subscribe(client, "test.event", "testing", func(evt *TestEvent) error {
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

	err := Subscribe(client, "test.event", "testing", func(evt *TestEvent) error {
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

	err := Subscribe(client, subject, "testing", func(evt *TestEvent) error {
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

	err := Subscribe(client, subject, "testing", func(evt *TestEvent) error {
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

	err := SubscribeBroadcast(client, subject, func(evt *TestEvent) error {
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

	err := SubscribeBroadcast(client, subject, func(evt *TestEvent) error {
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

	err := SubscribeBroadcast(client, subject, func(evt *TestEvent) error {
		received1.Add(1)
		return nil
	})
	assert.NoError(t, err)

	err = SubscribeBroadcast(client, subject, func(evt *TestEvent) error {
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

	err := SubscribeStreamBatch(client, "test.event", "testing", func(evts []*TestEvent) error {
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

	err := SubscribeStreamBatch(client, "test.event", "testing", func(evts []*TestEvent) error {
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

	err := SubscribeStreamBatch(client, subject, "testing", func(evts []*TestEvent) error {
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

	err := Subscribe(client, subject, "testing", func(evt *TestEvent) error {
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

	err := Subscribe(client, subject, "testing", func(evt *TestEvent) error {
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

	err := Subscribe(client, subject, "testing", func(evt *TestEvent) error {
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

	err := SubscribeBroadcast(client, subject, func(evt *TestEvent) error {
		return nil
	}, WithMaxAckWait(30*time.Second), WithIdleHeartbeat(5*time.Second))
	assert.NoError(t, err)
}
