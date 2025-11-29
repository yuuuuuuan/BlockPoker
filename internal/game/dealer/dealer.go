package dealer

import (
	"fmt"
	"math/rand"

	"BlockPoker/internal/game/table"
)

// Dealer 只负责洗牌与发牌（无规则判断）
type Dealer struct {
	deck []table.Card
	rnd  *rand.Rand
}

func NewDealer(seed int64) *Dealer {
	return &Dealer{
		deck: make([]table.Card, 0, 52),
		rnd:  rand.New(rand.NewSource(seed)),
	}
}

// NewDeck 初始化一副牌并洗牌
func (d *Dealer) NewDeck() {
	d.deck = d.makeDeck()
	d.shuffle()
}

func (d *Dealer) makeDeck() []table.Card {
	deck := make([]table.Card, 0, 52)
	for s := 0; s < 4; s++ {
		for r := 2; r <= 14; r++ {
			deck = append(deck, table.Card{Suit: s, Rank: r})
		}
	}
	return deck
}

func (d *Dealer) shuffle() {
	n := len(d.deck)
	for i := 0; i < n; i++ {
		j := d.rnd.Intn(n)
		d.deck[i], d.deck[j] = d.deck[j], d.deck[i]
	}
}

// DealHoleCards 给每个玩家发 2 张底牌，返回 map address -> []Card
func (d *Dealer) DealHoleCards(players []string) map[string][]table.Card {
	out := make(map[string][]table.Card, len(players))
	// 轮流发牌，先玩家 0 一张，...再玩家0 第二张
	for i := 0; i < 2; i++ {
		for _, addr := range players {
			card := d.draw()
			out[addr] = append(out[addr], card)
		}
	}
	return out
}

// DealCommunity 发公共牌 n 张（burn 忽略），追加到 tableCommunity 并返回新增牌
func (d *Dealer) DealCommunity(n int) []table.Card {
	out := make([]table.Card, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, d.draw())
	}
	return out
}

func (d *Dealer) draw() table.Card {
	if len(d.deck) == 0 {
		// should not happen if properly invoked
		d.NewDeck()
	}
	c := d.deck[0]
	d.deck = d.deck[1:]
	return c
}

// fmtCard 用于测试/日志
func fmtCard(c table.Card) string {
	suits := []string{"♣", "♦", "♥", "♠"}
	ranks := map[int]string{
		11: "J", 12: "Q", 13: "K", 14: "A",
	}
	r := fmt.Sprintf("%d", c.Rank)
	if v, ok := ranks[c.Rank]; ok {
		r = v
	}
	return fmt.Sprintf("%s%s", suits[c.Suit], r)
}
