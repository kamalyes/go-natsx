/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-04-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-04-23 00:00:00
 * @FilePath: \go-natsx\subscribe.go
 * @Description: go-natsx 泛型事件订阅
 *
 * 提供三种订阅模式：
 *   - Subscribe：普通事件订阅（QueueSubscribe 负载均衡），泛型自动反序列化
 *   - SubscribeBroadcast：广播事件订阅（Subscribe 模式），所有订阅者都收到消息
 *   - SubscribeStreamBatch：批量流式消费（JetStream Pull 模式），支持批量拉取
 *
 * 消费者池：
 *   - 局部消费者池：每个订阅创建独立 WorkerPool 处理消息
 *   - 全局消费者池：共享 Client 级别的 WorkerPool，通过 InitWorkerPool 初始化
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package natsx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/kamalyes/go-toolbox/pkg/syncx"
	"github.com/nats-io/nats.go"
)

// SubscribeOptions 订阅选项
type SubscribeOptions struct {
	IsListenBroadcast  bool          // 是否广播方式监听
	IsIntoGlobalPool   bool          // 是否进入全局消费者池中消费
	LocalPoolSize      int           // 局部消费者池大小
	LocalPoolQueueSize int           // 局部消费者池队列大小
	BatchSize          int           // 批量消费的最大数量
	MaxWait            time.Duration // 批量消费最大等待消息时间
	ConsumeFastest     bool          // 批量消费时是否尽快消费
	MsgMaxRetry        uint64        // 消息消费失败最大重试次数
	MsgRetryInterval   time.Duration // 消息消费重试的时间间隔
	MaxAckWait         time.Duration // 消息最长消费时间
	IdleHeartbeat      time.Duration // 消费者心跳时间
	EnabledFlowControl bool          // 是否开启流控机制
}

// DefaultSubscribeOptions 返回默认订阅选项
func DefaultSubscribeOptions() SubscribeOptions {
	return SubscribeOptions{
		LocalPoolSize:      1,
		LocalPoolQueueSize: 100,
		BatchSize:          100,
		MaxWait:            10 * time.Second,
		MsgMaxRetry:        3,
		MsgRetryInterval:   1 * time.Second,
		MaxAckWait:         30 * time.Second,
	}
}

// ApplySubOptsFunc 订阅选项函数
type ApplySubOptsFunc func(opt *SubscribeOptions)

// WithListenBroadcast 设置广播模式
func WithListenBroadcast() ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.IsListenBroadcast = true
	}
}

// WithIntoGlobalPool 设置进入全局消费者池
func WithIntoGlobalPool() ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.IsIntoGlobalPool = true
	}
}

// WithLocalPoolSize 设置局部消费者池大小
func WithLocalPoolSize(size, queueSize int) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.LocalPoolSize = size
		opt.LocalPoolQueueSize = queueSize
	}
}

// WithBatchSize 设置批量消费大小
func WithBatchSize(batchSize int) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.BatchSize = batchSize
	}
}

// WithMaxWait 设置批量消费最大等待时间
func WithMaxWait(maxWait time.Duration) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.MaxWait = maxWait
	}
}

// WithMsgMaxRetry 设置消息最大重试次数
func WithMsgMaxRetry(msgMaxRetry uint64) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.MsgMaxRetry = msgMaxRetry
	}
}

// WithMsgRetryInterval 设置消息重试间隔
func WithMsgRetryInterval(msgRetryInterval time.Duration) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.MsgRetryInterval = msgRetryInterval
	}
}

// WithMaxAckWait 设置消息最长消费时间
func WithMaxAckWait(maxAckWait time.Duration) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.MaxAckWait = maxAckWait
	}
}

// WithIdleHeartbeat 设置消费者心跳时间
func WithIdleHeartbeat(idleHeartbeat time.Duration) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.IdleHeartbeat = idleHeartbeat
	}
}

// WithEnableFlowControl 开启流控
func WithEnableFlowControl() ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.EnabledFlowControl = true
	}
}

// WithConsumeFastest 设置是否尽快消费
func WithConsumeFastest(consumeFastest bool) ApplySubOptsFunc {
	return func(opt *SubscribeOptions) {
		opt.ConsumeFastest = consumeFastest
	}
}

// Subscribe 普通事件订阅（QueueSubscribe 负载均衡模式）
// 同一 queue 组内只有一个消费者收到某条消息，适合任务分发场景
// 泛型 T 自动反序列化消息体
func Subscribe[T any](c *Client, eventName, subscriberName string, handleFunc func(evt *T) error, opts ...ApplySubOptsFunc) error {
	subOpts := DefaultSubscribeOptions()
	for _, opt := range opts {
		opt(&subOpts)
	}

	if subOpts.IsListenBroadcast {
		subOpts.LocalPoolSize = 1
		subOpts.IsIntoGlobalPool = false
	}

	if subOpts.IsIntoGlobalPool && c.WorkerPool() == nil {
		return ErrGlobalPoolNotInitialized
	}

	c.mu.RLock()
	conn := c.conn
	js := c.js
	c.mu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	enabledJS := js != nil

	var localPool *syncx.WorkerPool
	if !subOpts.IsIntoGlobalPool {
		localPool = syncx.NewWorkerPool(subOpts.LocalPoolSize, subOpts.LocalPoolQueueSize)
	}

	if !enabledJS {
		if subOpts.IsListenBroadcast {
			_, err := conn.Subscribe(eventName, func(msg *nats.Msg) {
				dispatchConsumer(c, msg, handleFunc, enabledJS, subOpts.IsIntoGlobalPool, localPool, subOpts.MsgMaxRetry, subOpts.MsgRetryInterval)
			})
			if err != nil {
				if localPool != nil {
					_ = localPool.Close()
				}
				return fmt.Errorf("%w: %v", ErrSubscribeFailed, err)
			}
			return nil
		}

		_, err := conn.QueueSubscribe(eventName, normalizeConsumerName(eventName+"_"+subscriberName), func(msg *nats.Msg) {
			dispatchConsumer(c, msg, handleFunc, enabledJS, subOpts.IsIntoGlobalPool, localPool, subOpts.MsgMaxRetry, subOpts.MsgRetryInterval)
		})
		if err != nil {
			if localPool != nil {
				_ = localPool.Close()
			}
			return fmt.Errorf("%w: %v", ErrSubscribeFailed, err)
		}
		return nil
	}

	var natsOpts []nats.SubOpt
	if subOpts.MaxAckWait > 0 {
		natsOpts = append(natsOpts, nats.AckWait(subOpts.MaxAckWait))
	}
	if subOpts.IdleHeartbeat > 0 {
		subOpts.EnabledFlowControl = true
		natsOpts = append(natsOpts, nats.IdleHeartbeat(subOpts.IdleHeartbeat))
	}
	if subOpts.EnabledFlowControl {
		natsOpts = append(natsOpts, nats.EnableFlowControl())
	}
	natsOpts = append(natsOpts, nats.ManualAck())

	if subOpts.IsListenBroadcast {
		_, err := js.Subscribe(eventName, func(msg *nats.Msg) {
			dispatchConsumer(c, msg, handleFunc, enabledJS, subOpts.IsIntoGlobalPool, localPool, subOpts.MsgMaxRetry, subOpts.MsgRetryInterval)
		}, natsOpts...)
		if err != nil {
			if localPool != nil {
				_ = localPool.Close()
			}
			return fmt.Errorf("%w: %v", ErrSubscribeFailed, err)
		}
		return nil
	}

	_, err := js.QueueSubscribe(eventName, normalizeConsumerName(eventName+"_"+subscriberName), func(msg *nats.Msg) {
		dispatchConsumer(c, msg, handleFunc, enabledJS, subOpts.IsIntoGlobalPool, localPool, subOpts.MsgMaxRetry, subOpts.MsgRetryInterval)
	}, natsOpts...)
	if err != nil {
		if localPool != nil {
			_ = localPool.Close()
		}
		return fmt.Errorf("%w: %v", ErrSubscribeFailed, err)
	}

	c.logger.Info("Subscribed", "subject", eventName, "subscriber", subscriberName, "broadcast", subOpts.IsListenBroadcast, "global_pool", subOpts.IsIntoGlobalPool)
	return nil
}

// SubscribeBroadcast 广播事件订阅
// 所有订阅者都收到消息，适合事件通知、状态同步场景
func SubscribeBroadcast[T any](c *Client, eventName string, handleFunc func(evt *T) error, opts ...ApplySubOptsFunc) error {
	return Subscribe[T](c, eventName, "", handleFunc, append(opts, WithListenBroadcast())...)
}

// SubscribeStreamBatch 批量流式消费（JetStream Pull 模式）
// 基于 JetStream PullSubscribe 实现批量拉取消息
func SubscribeStreamBatch[T any](c *Client, eventName, subscriberName string, handleFunc func(evts []*T) error, opts ...ApplySubOptsFunc) error {
	subOpts := DefaultSubscribeOptions()
	for _, opt := range opts {
		opt(&subOpts)
	}

	if subOpts.IsListenBroadcast {
		subOpts.LocalPoolSize = 1
		subOpts.IsIntoGlobalPool = false
	}
	if subOpts.IsIntoGlobalPool && c.WorkerPool() == nil {
		return ErrGlobalPoolNotInitialized
	}
	if subOpts.BatchSize <= 0 {
		subOpts.BatchSize = 100
	}
	if subOpts.MaxWait <= 0 {
		subOpts.MaxWait = 10 * time.Second
	}

	c.mu.RLock()
	js := c.js
	c.mu.RUnlock()

	if js == nil {
		return ErrJetStreamFailed
	}

	var localPool *syncx.WorkerPool
	if !subOpts.IsIntoGlobalPool {
		localPool = syncx.NewWorkerPool(subOpts.LocalPoolSize, subOpts.LocalPoolQueueSize)
	}

	sub, err := js.PullSubscribe(eventName, normalizeConsumerName(eventName+"_"+subscriberName))
	if err != nil {
		if localPool != nil {
			_ = localPool.Close()
		}
		return fmt.Errorf("%w: %v", ErrSubscribeFailed, err)
	}

	c.addSub(sub)

	go func() {
		defer sub.Unsubscribe()
		if localPool != nil {
			defer localPool.Close()
		}

		for {
			var messages []*nats.Msg
			start := time.Now()
			for len(messages) < subOpts.BatchSize {
				timeout := subOpts.MaxWait - time.Since(start)
				if timeout <= 0 {
					break
				}

				batch, fetchErr := sub.Fetch(subOpts.BatchSize-len(messages), nats.MaxWait(timeout))
				if fetchErr != nil {
					if errors.Is(fetchErr, context.DeadlineExceeded) || errors.Is(fetchErr, nats.ErrTimeout) {
						continue
					}
					c.logger.Error("Fetch batch stream error", "event", eventName, "subscriber", subscriberName, "error", fetchErr)
					time.Sleep(5 * time.Second)
					continue
				}

				if len(batch) > 0 {
					messages = append(messages, batch...)
				}

				if subOpts.ConsumeFastest {
					break
				}
			}

			if len(messages) > 0 {
				dispatchBatchConsumer(c, messages, handleFunc, subOpts.IsIntoGlobalPool, localPool, subOpts.MsgMaxRetry, subOpts.MsgRetryInterval)
			}
		}
	}()

	c.logger.Info("Stream batch subscribed", "subject", eventName, "subscriber", subscriberName, "batch_size", subOpts.BatchSize)
	return nil
}

// dispatchConsumer 分发单条消息到消费者池
func dispatchConsumer[T any](c *Client, msg *nats.Msg, handleFunc func(*T) error, isManualAck bool, isIntoGlobalPool bool, localPool *syncx.WorkerPool, msgMaxRetry uint64, msgRetryInterval time.Duration) {
	task := func() {
		var event T
		if err := jsoniter.Unmarshal(msg.Data, &event); err != nil {
			c.logger.Error("Unmarshal nats msg failed", "error", err)
			if isManualAck {
				nakMsg(c, msg, msgMaxRetry, msgRetryInterval)
			}
			return
		}

		if err := handleFunc(&event); err != nil {
			if isManualAck {
				nakMsg(c, msg, msgMaxRetry, msgRetryInterval)
			}
			return
		}

		if isManualAck {
			if err := msg.Ack(); err != nil {
				c.logger.Error("Nats msg ack error", "error", err)
			}
		}
	}

	if isIntoGlobalPool {
		if pool := c.WorkerPool(); pool != nil {
			_ = pool.SubmitNonBlocking(task)
		}
		return
	}

	if localPool != nil {
		_ = localPool.SubmitNonBlocking(task)
	}
}

// dispatchBatchConsumer 分发批量消息到消费者池
func dispatchBatchConsumer[T any](c *Client, messages []*nats.Msg, handleFunc func([]*T) error, isIntoGlobalPool bool, localPool *syncx.WorkerPool, msgMaxRetry uint64, msgRetryInterval time.Duration) {
	task := func() {
		_ = handleStreamBatch(c, messages, handleFunc, msgMaxRetry, msgRetryInterval)
	}

	if isIntoGlobalPool {
		if pool := c.WorkerPool(); pool != nil {
			_ = pool.SubmitNonBlocking(task)
		}
		return
	}

	if localPool != nil {
		_ = localPool.SubmitNonBlocking(task)
	}
}

// handleStreamBatch 处理批量流式消息
func handleStreamBatch[T any](c *Client, messages []*nats.Msg, handleFunc func([]*T) error, msgMaxRetry uint64, msgRetryInterval time.Duration) error {
	var (
		events        []*T
		validMessages []*nats.Msg
	)

	for _, msg := range messages {
		event := new(T)
		if err := jsoniter.Unmarshal(msg.Data, event); err != nil {
			c.logger.Error("Unmarshal nats msg failed", "error", err)
			nakMsg(c, msg, msgMaxRetry, msgRetryInterval)
			continue
		}
		events = append(events, event)
		validMessages = append(validMessages, msg)
	}

	if len(events) == 0 {
		return nil
	}

	if err := handleFunc(events); err != nil {
		for _, msg := range validMessages {
			nakMsg(c, msg, msgMaxRetry, msgRetryInterval)
		}
		return err
	}

	for _, msg := range validMessages {
		if err := msg.Ack(); err != nil {
			c.logger.Error("Nats msg ack error", "error", err)
		}
	}

	return nil
}

// nakMsg NAK 消息，支持重试
func nakMsg(c *Client, msg *nats.Msg, msgMaxRetry uint64, msgRetryInterval time.Duration) {
	if msgMaxRetry > 0 {
		if deliveryCount, err := msg.Metadata(); err == nil && deliveryCount.NumDelivered > msgMaxRetry {
			if err := msg.Term(); err != nil {
				c.logger.Error("Term msg failed", "error", err)
			}
			return
		}
	}

	var err error
	if msgRetryInterval > 0 {
		err = msg.NakWithDelay(msgRetryInterval)
	} else {
		err = msg.Nak()
	}
	if err != nil {
		c.logger.Error("Nats msg nak error", "error", err)
	}
}

// normalizeConsumerName 规范化消费者名称
func normalizeConsumerName(name string) string {
	return strings.ReplaceAll(name, ".", "_")
}
