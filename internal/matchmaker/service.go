package matchmaker

import (
	"BlockPoker/internal/websocket"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo        Repo
	playerTTL   int // seconds, 用于防止遗留队列
	hub         HubBroadcaster
	OnRoomReady func(*Room) // ✅ 成桌时调用的回调函数
}

type HubBroadcaster interface {
	BroadcastToPlayers(addrs []string, msg websocket.OutgoingMessage)
}

func NewService(repo Repo, playerTTL int, hub HubBroadcaster) *Service {
	return &Service{repo: repo, playerTTL: playerTTL, hub: hub}
}

// Join 入队并尝试立即成桌（随机）。若可成桌，返回房间；否则返回排队中。
func (s *Service) Join(ctx context.Context, req JoinRequest) (*Room, bool, error) {
	if req.TableSize <= 1 {
		return nil, false, errors.New("invalid tableSize")
	}

	// ❶ 防止重复匹配：检测玩家是否已经在房间中
	if checker, ok := s.repo.(interface {
		GetPlayerRoom(ctx context.Context, address string) (string, error)
	}); ok {
		roomID, _ := checker.GetPlayerRoom(ctx, req.Address)
		if roomID != "" {
			return nil, false, fmt.Errorf("player %s already in room %s", req.Address, roomID)
		}
	}

	// 统一以 pool+tableSize 作为匹配池
	if err := s.repo.Enqueue(ctx, req.Pool, req.TableSize, req.Address, s.playerTTL); err != nil {
		return nil, false, err
	}
	// 判断人数是否满足，满足则原子随机弹出 N 人（包含刚入队者）
	cnt, err := s.repo.Count(ctx, req.Pool, req.TableSize)
	if err != nil {
		return nil, false, err
	}
	if int(cnt) < req.TableSize {
		return nil, true, nil // queued
	}
	addrs, err := s.repo.PopNRandom(ctx, req.Pool, req.TableSize, req.TableSize)
	if err != nil {
		return nil, false, err
	}
	if len(addrs) < req.TableSize {
		// 并发竞争导致人数不足：回退为排队状态
		return nil, true, nil
	}
	room := &Room{
		ID:        uuid.NewString(),
		Pool:      req.Pool,
		TableSize: req.TableSize,
		Players:   addrs,
		CreatedAt: time.Now(),
	}

	//存入 Redis（房间数据）
	if saver, ok := s.repo.(interface {
		SaveRoom(context.Context, *Room, int) error
	}); ok {
		if err := saver.SaveRoom(ctx, room, s.playerTTL); err != nil {
			fmt.Println("⚠️ SaveRoom error:", err)
		}
	}

	//通知所有桌内玩家（通过 WebSocket Hub）
	msg := websocket.OutgoingMessage{
		Event: "matched",
		Data: map[string]any{
			"roomId":    room.ID,
			"pool":      room.Pool,
			"tableSize": room.TableSize,
			"players":   room.Players,
		},
	}
	s.hub.BroadcastToPlayers(addrs, msg)

	// ✅ 启动游戏逻辑
	if s.OnRoomReady != nil {
		go s.OnRoomReady(room)
	}

	return room, false, nil
}

func (s *Service) Cancel(ctx context.Context, address string) error {
	return s.repo.Remove(ctx, address)
}
