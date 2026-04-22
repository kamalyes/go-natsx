/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-04-20 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-04-20 00:00:00
 * @FilePath: \go-natsx\errors.go
 * @Description: go-natsx 错误定义
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package natsx

import "errors"

var (
	ErrNotConnected             = errors.New("nats: not connected")
	ErrAlreadyClosed            = errors.New("nats: already closed")
	ErrInvalidSubject           = errors.New("nats: invalid subject")
	ErrInvalidMessage           = errors.New("nats: invalid message")
	ErrPublishFailed            = errors.New("nats: publish failed")
	ErrSubscribeFailed          = errors.New("nats: subscribe failed")
	ErrJetStreamFailed          = errors.New("nats: jetstream operation failed")
	ErrBucketNotFound           = errors.New("nats: kv bucket not found")
	ErrKeyNotFound              = errors.New("nats: kv key not found")
	ErrTimeout                  = errors.New("nats: operation timed out")
	ErrUnavailable              = errors.New("nats: service unavailable")
	ErrGlobalPoolNotInitialized = errors.New("nats: global consumer pool not initialized, call InitWorkerPool first")
)
