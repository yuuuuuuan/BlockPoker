package dealer

import (
	"testing"
	"time"

	"BlockPoker/internal/game/table"
)

// 工具：检查是否有重复牌
func hasDuplicates(cards []table.Card) bool {
	seen := make(map[string]bool)
	for _, c := range cards {
		k := cardKey(c)
		if seen[k] {
			return true
		}
		seen[k] = true
	}
	return false
}

func cardKey(c table.Card) string {
	return string(rune(c.Suit)) + ":" + string(rune(c.Rank))
}

// ✅ 测试牌组初始化
func TestNewDeck(t *testing.T) {
	d := NewDealer(time.Now().UnixNano())
	d.NewDeck()

	if len(d.deck) != 52 {
		t.Fatalf("expected 52 cards, got %d", len(d.deck))
	}
	if hasDuplicates(d.deck) {
		t.Fatalf("deck should not contain duplicates")
	}

	// 检查花色和点数完整性
	suits := make(map[int]bool)
	ranks := make(map[int]bool)
	for _, c := range d.deck {
		suits[c.Suit] = true
		ranks[c.Rank] = true
	}
	if len(suits) != 4 {
		t.Fatalf("expected 4 suits, got %d", len(suits))
	}
	if len(ranks) != 13 {
		t.Fatalf("expected 13 ranks, got %d", len(ranks))
	}
}

// ✅ 测试洗牌效果（概率性验证）
func TestShuffleChangesOrder(t *testing.T) {
	d1 := NewDealer(42)
	d1.NewDeck()
	d2 := NewDealer(42)
	d2.NewDeck()

	// 因为种子相同，所以序列应相同
	for i := range d1.deck {
		if d1.deck[i] != d2.deck[i] {
			t.Fatalf("expected identical decks for same seed")
		}
	}

	// 新种子应生成不同序列
	d3 := NewDealer(99)
	d3.NewDeck()
	diff := false
	for i := range d1.deck {
		if d1.deck[i] != d3.deck[i] {
			diff = true
			break
		}
	}
	if !diff {
		t.Fatalf("expected deck with different seed to differ")
	}
}

// ✅ 测试底牌发放逻辑
func TestDealHoleCards(t *testing.T) {
	d := NewDealer(1)
	d.NewDeck()
	players := []string{"A", "B", "C"}
	hands := d.DealHoleCards(players)

	// 每个玩家应有 2 张牌
	for _, addr := range players {
		if len(hands[addr]) != 2 {
			t.Fatalf("player %s should have 2 cards, got %d", addr, len(hands[addr]))
		}
	}

	// 所有发出的牌不应重复
	all := []table.Card{}
	for _, h := range hands {
		all = append(all, h...)
	}
	if hasDuplicates(all) {
		t.Fatalf("hole cards contain duplicates")
	}

	// 发完后牌堆应少 6 张
	if len(d.deck) != 52-6 {
		t.Fatalf("expected remaining deck 46, got %d", len(d.deck))
	}
}

// ✅ 测试公共牌发放逻辑
func TestDealCommunity(t *testing.T) {
	d := NewDealer(2)
	d.NewDeck()

	flop := d.DealCommunity(3)
	turn := d.DealCommunity(1)
	river := d.DealCommunity(1)

	if len(flop) != 3 || len(turn) != 1 || len(river) != 1 {
		t.Fatalf("expected 3+1+1 cards, got %d %d %d", len(flop), len(turn), len(river))
	}

	all := append(append(flop, turn...), river...)
	if hasDuplicates(all) {
		t.Fatalf("community cards contain duplicates")
	}
	if len(d.deck) != 52-5 {
		t.Fatalf("expected 47 remaining, got %d", len(d.deck))
	}
}

// ✅ 测试自动补牌机制
func TestDrawResetsDeck(t *testing.T) {
	d := NewDealer(3)
	d.NewDeck()
	// 手动抽光牌
	for i := 0; i < 52; i++ {
		d.draw()
	}
	// 再抽一张应触发自动 NewDeck()
	card := d.draw()
	if (card.Rank < 2 || card.Rank > 14) || (card.Suit < 0 || card.Suit > 3) {
		t.Fatalf("invalid card returned after deck reset")
	}
}
