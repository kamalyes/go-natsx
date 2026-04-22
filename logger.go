/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-04-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-04-23 00:00:00
 * @FilePath: \go-natsx\logger.go
 * @Description: go-natsx 日志接口，直接复用 go-logger
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package natsx

import (
	"time"

	"github.com/kamalyes/go-logger"
)

// NewDefaultLogger 创建默认配置的NATS日志器
func NewDefaultLogger() logger.ILogger {
	return logger.NewLogger().
		WithLevel(logger.INFO).
		WithPrefix("[NATSX] ").
		WithShowCaller(false).
		WithColorful(true).
		WithTimeFormat(time.RFC3339)
}
