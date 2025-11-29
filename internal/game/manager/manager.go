package manager

import (
	"fmt"
	"sync"

	"BlockPoker/internal/game/engine"
	"BlockPoker/internal/game/table"
	"BlockPoker/internal/matchmaker"
	"BlockPoker/internal/websocket"
)

// GameManager 管理所有对局
type GameManager struct {
	mu           sync.RWMutex
	engines      map[string]*engine.Engine // roomID → engine
	playerToRoom map[string]string         // player address → roomID
	hub          websocket.HubInterface
}

func NewGameManager(hub websocket.HubInterface) *GameManager {
	return &GameManager{
		engines:      make(map[string]*engine.Engine),
		playerToRoom: make(map[string]string),
		hub:          hub,
	}
}

// StartRoom 创建桌子并启动 engine
func (m *GameManager) StartRoom(r *matchmaker.Room) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.engines[r.ID]; ok {
		return fmt.Errorf("engine for room %s exists", r.ID)
	}

	t := &table.Table{
		ID:        r.ID,
		Pool:      r.Pool,
		TableSize: r.TableSize,
		Players:   r.Players,
		CreatedAt: r.CreatedAt,
		Chips:     make([]int64, r.TableSize),
		Bets:      make([]int64, r.TableSize),
		Fold:      make([]bool, r.TableSize),
	}

	eng := engine.NewEngine(t, m.hub)
	m.engines[r.ID] = eng

	// ⭐ 建立玩家地址 → 房间 ID 映射
	for _, p := range r.Players {
		m.playerToRoom[p] = r.ID
	}

	// 异步进入游戏流程
	go eng.Start()

	return nil
}

// HandlePlayerMessage 统一入口（来自 Hub.Incoming）
func (m *GameManager) HandlePlayerMessage(msg websocket.IncomingMessage) {
	m.mu.RLock()
	roomID := m.playerToRoom[msg.From]
	eng := m.engines[roomID]
	m.mu.RUnlock()

	if eng == nil {
		return
	}

	switch msg.Event {

	case "player_action":
		// 交给 Engine（下注、跟注、弃牌等）
		eng.EnqueueAction(msg.From, msg.Data)

	case "chat":
		// 桌内聊天广播
		m.hub.BroadcastToPlayers(
			eng.Table.Players,
			websocket.OutgoingMessage{
				Event: "chat",
				Data: map[string]any{
					"from": msg.From,
					"text": msg.Data,
				},
			},
		)
	}
}
