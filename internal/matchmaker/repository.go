package matchmaker

import "context"

// Repo 定义对匹配池的抽象操作
type Repo interface {
	// Enqueue 将地址加入指定池（pool+tableSize）
	Enqueue(ctx context.Context, pool string, tableSize int, address string, ttlSeconds int) error
	// PopNRandom 当池内达到 N 人时，随机弹出 N 人（原子）
	PopNRandom(ctx context.Context, pool string, tableSize int, n int) ([]string, error)
	// Remove 将玩家从当前池移除（用于取消）
	Remove(ctx context.Context, address string) error
	// Count 返回池内人数
	Count(ctx context.Context, pool string, tableSize int) (int64, error)
}
