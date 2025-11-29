package matchmaker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type memRepo struct {
	mu      sync.Mutex
	pools   map[string]map[string]struct{} // key -> set(address)
	players map[string]string              // address -> key
}

func NewMemoryRepo() Repo {
	return &memRepo{
		pools:   make(map[string]map[string]struct{}),
		players: make(map[string]string),
	}
}

func memKey(pool string, tableSize int) string {
	return fmt.Sprintf("mm:pool:%s:%d", pool, tableSize)
}

func (m *memRepo) Enqueue(ctx context.Context, pool string, tableSize int, address string, ttlSeconds int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memKey(pool, tableSize)
	if _, ok := m.pools[key]; !ok {
		m.pools[key] = make(map[string]struct{})
	}
	m.pools[key][address] = struct{}{}
	m.players[address] = key
	// 简单忽略 TTL，内存版仅供测试
	return nil
}

func (m *memRepo) PopNRandom(ctx context.Context, pool string, tableSize int, n int) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := memKey(pool, tableSize)
	s, ok := m.pools[key]
	if !ok || len(s) < n {
		return []string{}, nil
	}

	// 随机取 n 个
	addrs := make([]string, 0, len(s))
	for a := range s {
		addrs = append(addrs, a)
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(addrs), func(i, j int) { addrs[i], addrs[j] = addrs[j], addrs[i] })

	chosen := addrs[:n]

	// ✅ 清理匹配池（与 Redis 行为对齐）
	delete(m.pools, key)

	// ✅ 删除玩家的反向索引
	for _, a := range chosen {
		delete(m.players, a)
	}

	return chosen, nil
}

func (m *memRepo) Remove(ctx context.Context, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key, ok := m.players[address]
	if !ok {
		return nil
	}
	if s, ok := m.pools[key]; ok {
		delete(s, address)
	}
	delete(m.players, address)
	return nil
}

func (m *memRepo) Count(ctx context.Context, pool string, tableSize int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := memKey(pool, tableSize)
	return int64(len(m.pools[key])), nil
}
