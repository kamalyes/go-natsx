/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-04-23 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-04-23 00:00:00
 * @FilePath: \go-natsx\test_helper_test.go
 * @Description: go-natsx 测试辅助工具
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package natsx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	defaultNATSURL = "nats://127.0.0.1:4222"
)

// natsURL 返回 NATS 服务器地址，优先从环境变量读取
func natsURL() string {
	if url := os.Getenv("NATS_TEST_URL"); url != "" {
		return url
	}
	return defaultNATSURL
}

// newConnectedClient 创建并连接到真实 NATS 服务器的客户端
// 如果连接失败则跳过测试
func newConnectedClient(t *testing.T) (*Client, *nats.Conn) {
	t.Helper()

	conn, err := nats.Connect(natsURL(),
		nats.Timeout(5*time.Second),
		nats.ReconnectWait(1*time.Second),
		nats.MaxReconnects(3),
	)
	if err != nil {
		t.Skipf("NATS server not available at %s: %v", natsURL(), err)
	}

	client, err := NewClient(conn)
	if err != nil {
		conn.Close()
		t.Fatalf("Failed to create client: %v", err)
	}

	return client, conn
}

// newConnectedClientWithJS 创建启用 JetStream 的客户端
func newConnectedClientWithJS(t *testing.T) (*Client, *nats.Conn) {
	t.Helper()

	client, conn := newConnectedClient(t)

	if err := client.EnableJetStream(); err != nil {
		client.Close()
		conn.Close()
		t.Skipf("JetStream not available (start nats-server with -js flag): %v", err)
	}

	return client, conn
}

// ensureStream 确保流存在
func ensureStream(t *testing.T, client *Client, streamName string) {
	t.Helper()

	js := client.JetStream()
	if js == nil {
		t.Fatal("JetStream not enabled")
	}

	_, err := js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{streamName + ".>"},
			Storage:  nats.MemoryStorage,
		})
		if err != nil {
			t.Skipf("Failed to create stream %s (JetStream may not be fully configured): %v", streamName, err)
		}
	}
}

// uniqueSubject 生成唯一的测试 Subject
func uniqueSubject(prefix string) string {
	return fmt.Sprintf("%s.%d", prefix, time.Now().UnixNano())
}
