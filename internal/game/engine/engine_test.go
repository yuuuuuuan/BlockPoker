package engine

import (
	"BlockPoker/internal/game/dealer"
	"BlockPoker/internal/game/table"
	"BlockPoker/internal/websocket"
	"reflect"
	"testing"
	"time"
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
		broadcasts:   make([]map[string]any, 0),
	}
}

func (h *mockHub) BroadcastToPlayers(addrs []string, msg websocket.OutgoingMessage) {
	// store event name + data
	entry := map[string]any{"event": msg.Event, "data": msg.Data}
	h.broadcasts = append(h.broadcasts, entry)
}

func (h *mockHub) SendToPlayer(addr string, msg websocket.OutgoingMessage) {
	// decode payload into map for assertions
	data := map[string]any{"event": msg.Event, "data": msg.Data}
	h.sentToPlayer[addr] = append(h.sentToPlayer[addr], data)
}

func (h *mockHub) ClientByAddress(addr string) (*websocket.Client, bool) {
	c, ok := h.clients[addr]
	return c, ok
}

func (h *mockHub) Close() {

	h.clients = make(map[string]*websocket.Client)
	h.sentToPlayer = make(map[string][]map[string]any)
	h.broadcasts = make([]map[string]any, 0)
}
func TestEngineStart_DealHoleCards(t *testing.T) {
	players := []string{"0xAAA", "0xBBB"}
	tbl := &table.Table{
		ID:        "room-test",
		Pool:      "default",
		TableSize: 2,
		Players:   players,
		CreatedAt: time.Now(),
	}
	h := newMockHub()
	eng := NewEngine(tbl, h)
	eng.Dealer = dealer.NewDealer(42) // deterministic seed for test
	eng.Start()

	// ensure both players received a deal_hole message
	if len(h.sentToPlayer["0xAAA"]) == 0 || len(h.sentToPlayer["0xBBB"]) == 0 {
		t.Fatalf("expected both players to receive hole cards")
	}

	// check each got 2 cards and cards are not identical (simple uniqueness)
	aCards := h.sentToPlayer["0xAAA"][0]["data"].(map[string]any)["cards"].([]table.Card)
	bCards := h.sentToPlayer["0xBBB"][0]["data"].(map[string]any)["cards"].([]table.Card)

	if len(aCards) != 2 || len(bCards) != 2 {
		t.Fatalf("each player should have 2 cards")
	}

	if reflect.DeepEqual(aCards, bCards) {
		t.Fatalf("player hole cards should not be identical")
	}

	// ensure a public broadcast was made
	found := false
	for _, b := range h.broadcasts {
		if b["event"] == "dealt_public" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected public dealt notification")
	}
}
