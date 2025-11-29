package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1), Hub: hub}
	c2 := &Client{Address: "0xB", Send: make(chan OutgoingMessage, 1), Hub: hub}

	hub.register <- c1
	hub.register <- c2

	msg := OutgoingMessage{
		Event: "room_created",
		Data:  map[string]interface{}{"roomId": "room123"},
	}

	hub.BroadcastToPlayers([]string{"0xA", "0xB"}, msg)

	time.Sleep(20 * time.Millisecond)

	m1 := <-c1.Send
	m2 := <-c2.Send

	assert.Equal(t, "room_created", m1.Event)
	assert.Equal(t, "room_created", m2.Event)
}

func TestHubBroadcastToPlayers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// create clients
	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1), Hub: hub}
	c2 := &Client{Address: "0xB", Send: make(chan OutgoingMessage, 1), Hub: hub}

	// register
	hub.register <- c1
	hub.register <- c2

	msg := OutgoingMessage{
		Event: "room_ready",
		Data:  map[string]interface{}{"roomId": "room123"},
	}

	hub.BroadcastToPlayers([]string{"0xA", "0xB"}, msg)

	time.Sleep(20 * time.Millisecond)

	m1 := <-c1.Send
	m2 := <-c2.Send

	assert.Equal(t, "room_ready", m1.Event)
	assert.Equal(t, "room_ready", m2.Event)
}

func TestHubSendToPlayer(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1), Hub: hub}
	c2 := &Client{Address: "0xB", Send: make(chan OutgoingMessage, 1), Hub: hub}

	hub.register <- c1
	hub.register <- c2

	msg := OutgoingMessage{
		Event: "private_msg",
		Data:  "hello A",
	}

	hub.SendToPlayer("0xA", msg)

	time.Sleep(20 * time.Millisecond)

	received := <-c1.Send

	// ensure A received the private message
	assert.Equal(t, "private_msg", received.Event)
	assert.Equal(t, "hello A", received.Data)

	// ensure B received nothing
	select {
	case <-c2.Send:
		assert.Fail(t, "B should NOT receive anything")
	default:
		// success
	}
}

func TestHubRegisterUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c := &Client{
		Address: "0xA",
		Send:    make(chan OutgoingMessage, 1),
		Hub:     hub,
	}

	hub.register <- c
	time.Sleep(10 * time.Millisecond)

	if _, ok := hub.clients["0xA"]; !ok {
		t.Fatalf("client should be registered")
	}

	hub.unregister <- c
	time.Sleep(10 * time.Millisecond)

	if _, ok := hub.clients["0xA"]; ok {
		t.Fatalf("client should be removed after unregister")
	}
}

func TestHubBroadcast1(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1)}
	c2 := &Client{Address: "0xB", Send: make(chan OutgoingMessage, 1)}

	hub.register <- c1
	hub.register <- c2

	msg := OutgoingMessage{Event: "table_start"}

	hub.BroadcastToPlayers([]string{"0xA", "0xB"}, msg)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, "table_start", (<-c1.Send).Event)
	assert.Equal(t, "table_start", (<-c2.Send).Event)
}

func TestHubSendToPlayer1(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1)}
	hub.register <- c1

	msg := OutgoingMessage{Event: "deal_private"}
	hub.SendToPlayer("0xA", msg)

	time.Sleep(10 * time.Millisecond)

	recv := <-c1.Send
	assert.Equal(t, "deal_private", recv.Event)
}

func BenchmarkHubBroadcast(b *testing.B) {
	hub := NewHub()
	go hub.Run()

	// 创建两个客户端，并给他们的 Send 启动 drain goroutine
	c1 := &Client{Address: "0xA", Send: make(chan OutgoingMessage, 1024), Hub: hub}
	c2 := &Client{Address: "0xB", Send: make(chan OutgoingMessage, 1024), Hub: hub}

	// 所有 Send 都必须有人接收，否则 Hub 会死锁
	go func() {
		for range c1.Send {
		}
	}()
	go func() {
		for range c2.Send {
		}
	}()

	hub.register <- c1
	hub.register <- c2

	b.ResetTimer()
	msg := OutgoingMessage{Event: "bench", Data: nil}

	for i := 0; i < b.N; i++ {
		hub.BroadcastToPlayers([]string{"0xA", "0xB"}, msg)
	}

	// 给 hub 一点时间消化剩余消息
	time.Sleep(50 * time.Millisecond)
}

func BenchmarkSendToPlayer(b *testing.B) {
	hub := NewHub()
	go hub.Run()

	c := &Client{Address: "0xPLAYER", Send: make(chan OutgoingMessage, 1)}
	hub.register <- c

	msg := OutgoingMessage{Event: "private_card"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.SendToPlayer("0xPLAYER", msg)
	}
}
