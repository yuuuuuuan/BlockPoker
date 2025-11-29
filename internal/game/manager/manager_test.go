package manager

import (
	"sync"
	"testing"
	"time"

	"BlockPoker/internal/matchmaker"
	"BlockPoker/internal/websocket"
)

// mockHub 实现 HubInterface，记录消息
type mockHub struct {
	sentToPlayer map[string][]map[string]any
	clients      map[string]*websocket.Client
	broadcasts   []map[string]any
}

func newMockHub() *mockHub {
	return &mockHub{
		sentToPlayer: make(map[string][]map[string]any),
		clients:      make(map[string]*websocket.Client),
		broadcasts:   make([]map[string]any, 0),
	}
}

func (h *mockHub) BroadcastToPlayers(addrs []string, msg websocket.OutgoingMessage) {
	// store event name + data
	entry := map[string]any{"event": msg.Event, "data": msg.Data}
	h.broadcasts = append(h.broadcasts, entry)
}

func (h *mockHub) ClientByAddress(addr string) (*websocket.Client, bool) {
	c, ok := h.clients[addr]
	return c, ok
}

func (h *mockHub) SendToPlayer(addr string, msg websocket.OutgoingMessage) {
	// decode payload into map for assertions
	data := map[string]any{"event": msg.Event, "data": msg.Data}
	h.sentToPlayer[addr] = append(h.sentToPlayer[addr], data)
}

func (h *mockHub) Close() {

	h.clients = make(map[string]*websocket.Client)
	h.sentToPlayer = make(map[string][]map[string]any)
	h.broadcasts = make([]map[string]any, 0)
}

// ✅ TestGameManagerStartRoom: 主测试用例
func TestGameManagerStartRoom(t *testing.T) {
	mockHub := newMockHub()
	mgr := NewGameManager(mockHub)

	room := &matchmaker.Room{
		ID:        "room-1",
		Pool:      "default",
		TableSize: 2,
		Players:   []string{"0xA", "0xB"},
		CreatedAt: time.Now(),
	}

	if err := mgr.StartRoom(room); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 等待异步启动引擎
	time.Sleep(50 * time.Millisecond)

	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if len(mgr.engines) != 1 {
		t.Fatalf("expected 1 engine, got %d", len(mgr.engines))
	}

	if _, ok := mgr.engines["room-1"]; !ok {
		t.Fatalf("expected engine for room-1 to exist")
	}
}

// ✅ TestGameManagerDuplicateRoom: 重复房间应报错
func TestGameManagerDuplicateRoom(t *testing.T) {
	mockHub := &mockHub{}
	mgr := NewGameManager(mockHub)

	room := &matchmaker.Room{
		ID:        "r1",
		Pool:      "default",
		TableSize: 2,
		Players:   []string{"P1", "P2"},
		CreatedAt: time.Now(),
	}

	if err := mgr.StartRoom(room); err != nil {
		t.Fatalf("unexpected error first start: %v", err)
	}
	time.Sleep(10 * time.Millisecond)

	if err := mgr.StartRoom(room); err == nil {
		t.Fatalf("expected error for duplicate room, got nil")
	}
}

// ✅ TestGameManagerConcurrency: 并发安全
func TestGameManagerConcurrency(t *testing.T) {
	mockHub := &mockHub{}
	mgr := NewGameManager(mockHub)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			room := &matchmaker.Room{
				ID:        "r" + string(rune(i+'A')),
				Pool:      "p",
				TableSize: 2,
				Players:   []string{"X", "Y"},
				CreatedAt: time.Now(),
			}
			_ = mgr.StartRoom(room)
		}(i)
	}
	wg.Wait()

	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	if len(mgr.engines) == 0 {
		t.Fatalf("expected some engines created")
	}
}
