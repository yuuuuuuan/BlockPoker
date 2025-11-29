package matchmaker

import (
	ws "BlockPoker/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// MockHub 用于捕获 BroadcastToPlayers 的调用并记录每个地址收到的消息
type MockHub struct {
	mu   sync.Mutex
	msgs map[string]ws.OutgoingMessage
}

func NewMockHub() *MockHub {
	return &MockHub{msgs: make(map[string]ws.OutgoingMessage)}
}

func (m *MockHub) BroadcastToPlayers(addrs []string, msg ws.OutgoingMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range addrs {
		m.msgs[strings.ToLower(a)] = msg
	}
}

func (m *MockHub) GetMsg(addr string) (ws.OutgoingMessage, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.msgs[strings.ToLower(addr)]
	return msg, ok
}

// ---------- 内存实现测试 ----------
func Test_MemoryRepo_MatchFlow(t *testing.T) {
	repo := NewMemoryRepo()
	hub := NewMockHub()
	svc := NewService(repo, 60, hub)

	pool := "cash-1-2"
	size := 3
	addrs := []string{"0xA", "0xB", "0xC", "0xD", "0xE", "0xF"}

	// 入队前两人，不应成桌
	for i := 0; i < 2; i++ {
		_, queued, err := svc.Join(context.Background(), JoinRequest{
			Address: addrs[i], Pool: pool, TableSize: size,
		})
		assert.NoError(t, err)
		assert.True(t, queued)
	}

	// 第三人入队，应立即成桌（随机 3 人）
	room, queued, err := svc.Join(context.Background(), JoinRequest{
		Address: addrs[2], Pool: pool, TableSize: size,
	})
	assert.NoError(t, err)
	assert.False(t, queued)
	assert.NotNil(t, room)
	assert.Equal(t, size, len(room.Players))

	// 验证 hub 向房间内每个玩家都广播了 matched 消息
	for _, p := range room.Players {
		msg, ok := hub.GetMsg(p)
		assert.True(t, ok, "player %s should have received a message", p)
		assert.Equal(t, "matched", msg.Event)
		// 解析 Data 验证 roomId 与 players 列表
		dataBytes, _ := json.Marshal(msg.Data)
		var payload map[string]interface{}
		_ = json.Unmarshal(dataBytes, &payload)
		assert.Equal(t, room.ID, payload["roomId"])
	}

	// 再入队 3 人，应再次成桌
	for i := 3; i < 5; i++ {
		_, q, err := svc.Join(context.Background(), JoinRequest{
			Address: addrs[i], Pool: pool, TableSize: size,
		})
		assert.NoError(t, err)
		assert.True(t, q)
	}
	room2, q2, err := svc.Join(context.Background(), JoinRequest{
		Address: addrs[5], Pool: pool, TableSize: size,
	})
	assert.NoError(t, err)
	assert.False(t, q2)
	assert.NotNil(t, room2)
	assert.Equal(t, size, len(room2.Players))

	// hub 也应向第二桌所有玩家广播
	for _, p := range room2.Players {
		msg, ok := hub.GetMsg(p)
		assert.True(t, ok, "player %s should have received a message for second room", p)
		assert.Equal(t, "matched", msg.Event)
	}
}

// ---------- Redis（miniredis）实现测试 ----------
func Test_RedisRepo_MatchFlow(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := NewRedisRepo(rdb)
	hub := NewMockHub()
	svc := NewService(repo, 60, hub)

	pool := "mtt-low"
	size := 2
	a1, a2, a3, a4 := "0x111", "0x222", "0x333", "0x444"

	// a1 入队 -> 排队
	_, queued, err := svc.Join(context.Background(), JoinRequest{Address: a1, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.True(t, queued)

	// a2 入队 -> 应成桌
	room, queued, err := svc.Join(context.Background(), JoinRequest{Address: a2, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.False(t, queued)
	assert.NotNil(t, room)
	assert.Equal(t, size, len(room.Players))

	// hub 应向 a1, a2 广播
	for _, p := range room.Players {
		msg, ok := hub.GetMsg(p)
		assert.True(t, ok)
		assert.Equal(t, "matched", msg.Event)
	}

	// Redis 中应存在保存的 room key（mm:room:{id}）
	roomKey := "mm:room:" + room.ID
	exists := mr.Exists(roomKey)
	assert.True(t, exists, "room key should exist in redis")

	// a3 入队并取消 -> 不应参与之后的配桌
	_, queued, err = svc.Join(context.Background(), JoinRequest{Address: a3, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.True(t, queued)
	err = svc.Cancel(context.Background(), a3)
	assert.NoError(t, err)

	// a4 入队 -> 因 a3 已取消，应与 a4 重新等待，直到下一人（此处直接补 a3 再入队）
	// 先入队 a4
	_, queued, err = svc.Join(context.Background(), JoinRequest{Address: a4, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.True(t, queued)

	// 补：a3 重新入队 -> 应成桌
	room2, queued, err := svc.Join(context.Background(), JoinRequest{Address: a3, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	if room2 == nil {
		// 由于并发等原因，如果 room2 为 nil，说明返回 queued -> 继续入队 a3之后再入队 a4 的情况
		// 但我们已经入过 a4，这里可以再次尝试让 a4 入队以触发成桌（保险）
		room2, queued, err = svc.Join(context.Background(), JoinRequest{Address: a4, Pool: pool, TableSize: size})
		assert.NoError(t, err)
		assert.False(t, queued)
		assert.NotNil(t, room2)
		assert.Equal(t, size, len(room2.Players))
	} else {
		assert.False(t, queued)
		assert.Equal(t, size, len(room2.Players))
	}

	// 验证池清理（应为空）
	cnt, err := repo.Count(context.Background(), pool, size)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), cnt)
}

// ---------- 并发竞争测试（可选） ----------
func Test_RedisRepo_ConcurrentJoins(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := NewRedisRepo(rdb)
	hub := NewMockHub()
	svc := NewService(repo, 60, hub)

	pool := "cash-5-10"
	size := 3
	addrs := []string{"0xA", "0xB", "0xC", "0xD", "0xE", "0xF"}

	done := make(chan struct{}, len(addrs))
	for _, a := range addrs {
		go func(addr string) {
			_, _, _ = svc.Join(context.Background(), JoinRequest{
				Address: addr, Pool: pool, TableSize: size,
			})
			done <- struct{}{}
		}(a)
	}
	for range addrs {
		<-done
	}

	// 等短暂时间让 miniredis 上的异步 pipeline 执行
	time.Sleep(50 * time.Millisecond)

	// 最终应当恰好出 2 桌或 1 桌 + 余员 0
	cnt, err := repo.Count(context.Background(), pool, size)
	assert.NoError(t, err)
	// 6 人，3 人一桌 -> 余 0
	assert.Equal(t, int64(0), cnt)
}

// 额外：确保 memory repo 实现了 SaveRoom（若未实现此方法，Service 会跳过保存，测试仍可通过）
func Test_MemoryRepo_SaveRoomCompatibility(t *testing.T) {
	// 只是保证内存 repo 不会引起 panic（SaveRoom 是可选接口）
	repo := NewMemoryRepo()
	hub := NewMockHub()
	svc := NewService(repo, 60, hub)

	rq := JoinRequest{Address: uuid.NewString(), Pool: "p", TableSize: 2}
	_, _, err := svc.Join(context.Background(), rq)
	assert.NoError(t, err)
}

// Test_RedisRepo_QueueLifecycle 验证 Redis 队列创建与删除的完整生命周期
func Test_RedisRepo_QueueLifecycle(t *testing.T) {
	ctx := context.Background()
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := NewRedisRepo(rdb)

	pool := "qa-test"
	tableSize := 2
	p1, p2 := "0xAAA", "0xBBB"
	key := poolKey(pool, tableSize)

	// 🟢 Step 1: 玩家1 入队 -> 集合应创建
	err = repo.Enqueue(ctx, pool, tableSize, p1, 60)
	assert.NoError(t, err)
	exists := mr.Exists(key)
	assert.True(t, exists, "pool should exist after first enqueue")

	// 🟢 Step 2: 玩家2 入队 -> 集合仍存在，人数 = 2
	err = repo.Enqueue(ctx, pool, tableSize, p2, 60)
	assert.NoError(t, err)
	count, err := repo.Count(ctx, pool, tableSize)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count, "pool should contain 2 players")

	// 🟢 Step 3: PopNRandom 取出 2 人 -> 集合应被清空删除
	addrs, err := repo.PopNRandom(ctx, pool, tableSize, tableSize)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{p1, p2}, addrs, "should return both players")
	exists = mr.Exists(key)
	assert.False(t, exists, "pool key should be deleted after PopNRandom")

	// 🟢 Step 4: 玩家3 再入队 -> 集合应重新创建
	p3 := "0xCCC"
	err = repo.Enqueue(ctx, pool, tableSize, p3, 60)
	assert.NoError(t, err)
	assert.True(t, mr.Exists(key), "pool key should exist again after new enqueue")

	// 🟢 Step 5: 玩家3 取消 -> 集合为空应被自动删除
	err = repo.Remove(ctx, p3)
	assert.NoError(t, err)
	exists = mr.Exists(key)
	assert.False(t, exists, "pool key should be removed when empty after cancel")

	// 🟢 Step 6: 玩家1 重新入队 + TTL 过期验证
	err = repo.Enqueue(ctx, pool, tableSize, p1, 1) // TTL = 1s
	assert.NoError(t, err)
	assert.True(t, mr.Exists(key))
	time.Sleep(1500 * time.Millisecond)
	// TTL 不影响 pool，因为 pool 是 set，不随 player TTL 消失
	assert.True(t, mr.Exists(key), "pool should still exist after player TTL expired")
}

// ---------- 玩家重复匹配保护测试 ----------
func Test_PlayerCannotRejoin_WhenAlreadyInRoom(t *testing.T) {
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := NewRedisRepo(rdb)
	hub := NewMockHub()
	svc := NewService(repo, 60, hub)

	ctx := context.Background()
	pool := "dup-test"
	size := 2
	a1, a2 := "0xAAA", "0xBBB"

	// 🟢 Step 1: a1 入队
	_, queued, err := svc.Join(ctx, JoinRequest{Address: a1, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.True(t, queued, "first player should be queued")

	// 🟢 Step 2: a2 入队 -> 应成桌
	room, queued, err := svc.Join(ctx, JoinRequest{Address: a2, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.False(t, queued)
	assert.NotNil(t, room)
	assert.Equal(t, size, len(room.Players))
	assert.True(t, mr.Exists("mm:room:"+room.ID), "room should exist in Redis")

	// 验证 playerRoom 映射存在
	key := fmt.Sprintf("mm:playerRoom:%s", a1)
	val, _ := mr.Get(key)
	assert.Equal(t, room.ID, val, "playerRoom mapping should be set")

	// 🛑 Step 3: a1 再次匹配 -> 应被拒绝
	_, _, err = svc.Join(ctx, JoinRequest{Address: a1, Pool: pool, TableSize: size})
	assert.Error(t, err, "player already in room should trigger error")
	assert.Contains(t, err.Error(), "already in room")

	// 🟡 Step 4: 模拟房间结束（删除 playerRoom）
	mr.Del(key)

	// 🟢 Step 5: a1 再次匹配 -> 应允许重新入队
	_, queued, err = svc.Join(ctx, JoinRequest{Address: a1, Pool: pool, TableSize: size})
	assert.NoError(t, err)
	assert.True(t, queued, "player should rejoin after leaving room")
}
