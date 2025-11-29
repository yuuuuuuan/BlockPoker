package matchmaker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisRepo struct {
	rdb *redis.Client
}

func NewRedisRepo(rdb *redis.Client) Repo {
	return &redisRepo{rdb: rdb}
}

// key 约定：
//
//	set: mm:pool:{pool}:{tableSize}         -> Set(address,...)
//	kv : mm:player:{address}                -> value "pool:tableSize" (便于取消时定位池)
//	ttl 辅助: 对 player key 设置 TTL，避免长期遗留
func poolKey(pool string, tableSize int) string {
	return fmt.Sprintf("mm:pool:%s:%d", pool, tableSize)
}
func playerKey(addr string) string {
	return fmt.Sprintf("mm:player:%s", addr)
}

func (r *redisRepo) Enqueue(ctx context.Context, pool string, tableSize int, address string, ttlSeconds int) error {
	p := r.rdb.Pipeline()
	p.SAdd(ctx, poolKey(pool, tableSize), address)
	p.Set(ctx, playerKey(address), fmt.Sprintf("%s:%d", pool, tableSize), time.Duration(ttlSeconds)*time.Second)
	_, err := p.Exec(ctx)
	return err
}

func (r *redisRepo) PopNRandom(ctx context.Context, pool string, tableSize int, n int) ([]string, error) {
	key := poolKey(pool, tableSize)
	// Redis 3.2+ 支持 SPOP COUNT，一次随机弹出 n 个元素并从集合删除（原子）
	res, err := r.rdb.SPopN(ctx, key, int64(n)).Result()
	if err != nil {
		return nil, err
	}
	// 清理 playerKey
	if len(res) > 0 {
		p := r.rdb.Pipeline()
		for _, addr := range res {
			p.Del(ctx, playerKey(addr))
		}
		_, _ = p.Exec(ctx)
	}
	return res, nil
}

func (r *redisRepo) Remove(ctx context.Context, address string) error {
	// 读取 player:... kv
	kv, err := r.rdb.Get(ctx, playerKey(address)).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	// 解析 "pool:tableSize" 格式，安全拆分
	parts := strings.SplitN(kv, ":", 2)
	if len(parts) != 2 {
		// 如果格式不对，仍删除 playerKey 并返回
		_ = r.rdb.Del(ctx, playerKey(address)).Err()
		return nil
	}
	pool := parts[0]
	sizeStr := parts[1]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		_ = r.rdb.Del(ctx, playerKey(address)).Err()
		return nil
	}

	poolK := poolKey(pool, size)
	playerK := playerKey(address)

	// Lua 脚本：删除 playerKey、从集合中移除成员；若集合空则删除集合
	// KEYS[1] = playerKey, KEYS[2] = poolKey, ARGV[1] = address
	script := `
        redis.call("DEL", KEYS[1])
        redis.call("SREM", KEYS[2], ARGV[1])
        if redis.call("SCARD", KEYS[2]) == 0 then
            redis.call("DEL", KEYS[2])
        end
        return 1
    `
	if err := r.rdb.Eval(ctx, script, []string{playerK, poolK}, address).Err(); err != nil {
		// 如果 Eval 不被支持（某些 miniredis 版本可能不完整），回退到非原子实现
		// 但仍尽量安全地执行：先 SREM，再 DEL playerKey，再检测并删除空集
		p := r.rdb.Pipeline()
		p.SRem(ctx, poolK, address)
		p.Del(ctx, playerK)
		if _, execErr := p.Exec(ctx); execErr != nil {
			return execErr
		}
		// 再次确认集合是否空
		if n, _ := r.rdb.SCard(ctx, poolK).Result(); n == 0 {
			_ = r.rdb.Del(ctx, poolK).Err()
		}
	}

	return nil
}

func (r *redisRepo) SaveRoom(ctx context.Context, room *Room, ttlSeconds int) error {
	key := fmt.Sprintf("mm:room:%s", room.ID)
	data, _ := json.Marshal(room)
	p := r.rdb.Pipeline()
	p.Set(ctx, key, data, time.Duration(ttlSeconds)*time.Second)
	for _, addr := range room.Players {
		p.Set(ctx, fmt.Sprintf("mm:playerRoom:%s", addr), room.ID, time.Duration(ttlSeconds)*time.Second)
	}
	_, err := p.Exec(ctx)
	return err
}

func (r *redisRepo) Count(ctx context.Context, pool string, tableSize int) (int64, error) {
	return r.rdb.SCard(ctx, poolKey(pool, tableSize)).Result()
}

func (r *redisRepo) GetPlayerRoom(ctx context.Context, address string) (string, error) {
	key := fmt.Sprintf("mm:playerRoom:%s", address)
	val, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}
