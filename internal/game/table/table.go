package table

import (
	"fmt"
	"time"
)

// 简化版数据结构，和 matchmaker.Room 对接时传入 Players 地址列表
type Table struct {
	ID        string
	Pool      string
	TableSize int
	Players   []string // addresses e.g. "0xAAA"
	CreatedAt time.Time

	// 运行时状态
	Community []Card
	Pot       int64
	State     string
	// seat index -> chips, bet, folded...
	Chips []int64
	Bets  []int64
	Fold  []bool
}

// Card 定义 (suit 0-3, rank 2-14)
type Card struct {
	Suit int `json:"suit"`
	Rank int `json:"rank"`
}

func (c Card) String() string {
	return fmtCard(c)
}

func fmtCard(c Card) string {
	suits := []string{"♣", "♦", "♥", "♠"}
	ranks := map[int]string{
		11: "J",
		12: "Q",
		13: "K",
		14: "A",
	}
	rankStr, ok := ranks[c.Rank]
	if !ok {
		rankStr = fmt.Sprintf("%d", c.Rank)
	}
	suitStr := "?"
	if c.Suit >= 0 && c.Suit < len(suits) {
		suitStr = suits[c.Suit]
	}
	return rankStr + suitStr
}
