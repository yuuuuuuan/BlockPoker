package engine

import (
	"time"

	"BlockPoker/internal/game/dealer"
	"BlockPoker/internal/game/table"
	"BlockPoker/internal/websocket"
)

// ---------------------
//   ACTION DEFINITION
// ---------------------

type Action struct {
	Player  string
	Payload interface{}
}

// ---------------------
//       ENGINE
// ---------------------

type Engine struct {
	Table      *table.Table
	Dealer     *dealer.Dealer
	Hub        websocket.HubInterface
	actionChan chan Action
}

func NewEngine(t *table.Table, hub websocket.HubInterface) *Engine {
	return &Engine{
		Table:      t,
		Dealer:     dealer.NewDealer(time.Now().UnixNano()),
		Hub:        hub,
		actionChan: make(chan Action, 32), // 防止死锁
	}
}

// Start: 发牌 + 广播 + 启动 action loop
func (e *Engine) Start() {
	e.Dealer.NewDeck()
	e.Table.State = "preflop"

	// 玩家底牌
	holeMap := e.Dealer.DealHoleCards(e.Table.Players)

	// 私牌发给对应玩家
	for addr, cards := range holeMap {
		payload := map[string]any{
			"event":   "deal_hole",
			"table":   e.Table.ID,
			"cards":   cards,
			"you":     addr,
			"state":   e.Table.State,
			"players": e.Table.Players,
		}

		e.Hub.SendToPlayer(addr, websocket.OutgoingMessage{
			Event: "deal_hole",
			Data:  payload,
		})
	}

	// 公共信息广播
	publicInfo := map[string]any{
		"event":   "dealt_public",
		"table":   e.Table.ID,
		"state":   e.Table.State,
		"players": e.Table.Players,
	}

	e.Hub.BroadcastToPlayers(e.Table.Players, websocket.OutgoingMessage{
		Event: "dealt_public",
		Data:  publicInfo,
	})

	// 启动动作处理循环
	go e.actionLoop()
}

// 动作循环：异步读取用户操作
func (e *Engine) actionLoop() {
	for act := range e.actionChan {
		e.handleAction(act)
	}
}

// 分发玩家动作（下注、弃牌、过牌等）
func (e *Engine) handleAction(a Action) {
	// 这里以后你扩展完整德州逻辑
	// 我给你一个演示输出（真实使用要移除）
	msg := websocket.OutgoingMessage{
		Event: "action_ack",
		Data: map[string]any{
			"player":  a.Player,
			"payload": a.Payload,
			"table":   e.Table.ID,
		},
	}
	e.Hub.BroadcastToPlayers(e.Table.Players, msg)
}

// 玩家动作入口（GameManager 调用）
func (e *Engine) EnqueueAction(player string, payload interface{}) {
	if e.actionChan != nil {
		e.actionChan <- Action{
			Player:  player,
			Payload: payload,
		}
	}
}

// --------------------------
//        下一阶段逻辑
// --------------------------

func (e *Engine) NextRound() {
	switch e.Table.State {
	case "preflop":
		cards := e.Dealer.DealCommunity(3)
		e.Table.Community = append(e.Table.Community, cards...)
		e.Table.State = "flop"
		e.broadcastCommunity(cards)

	case "flop":
		cards := e.Dealer.DealCommunity(1)
		e.Table.Community = append(e.Table.Community, cards...)
		e.Table.State = "turn"
		e.broadcastCommunity(cards)

	case "turn":
		cards := e.Dealer.DealCommunity(1)
		e.Table.Community = append(e.Table.Community, cards...)
		e.Table.State = "river"
		e.broadcastCommunity(cards)

	case "river":
		e.Table.State = "showdown"
		e.Hub.BroadcastToPlayers(e.Table.Players, websocket.OutgoingMessage{
			Event: "showdown_start",
			Data:  map[string]any{"table": e.Table.ID},
		})
	}
}

func (e *Engine) broadcastCommunity(cards []table.Card) {
	payload := map[string]any{
		"event":     "community",
		"table":     e.Table.ID,
		"community": e.Table.Community,
		"new":       cards,
		"state":     e.Table.State,
	}

	e.Hub.BroadcastToPlayers(e.Table.Players, websocket.OutgoingMessage{
		Event: "community",
		Data:  payload,
	})
}
