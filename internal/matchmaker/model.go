package matchmaker

import "time"

// JoinRequest 前端提交的匹配请求
type JoinRequest struct {
	Address   string `json:"address" binding:"required"`
	Pool      string `json:"pool" binding:"required"`      // 例如 "cash-1-2"、"mtt-low"
	TableSize int    `json:"tableSize" binding:"required"` // 2/6/9 等
}

// JoinResponse 返回是否已成桌；若已成桌则给出房间信息
type JoinResponse struct {
	Queued    bool     `json:"queued"`
	RoomID    string   `json:"roomId,omitempty"`
	Players   []string `json:"players,omitempty"`
	Pool      string   `json:"pool"`
	TableSize int      `json:"tableSize"`
}

// CancelRequest 取消匹配
type CancelRequest struct {
	Address string `json:"address" binding:"required"`
}

// Room 组桌结果
type Room struct {
	ID        string
	Pool      string
	TableSize int
	Players   []string
	CreatedAt time.Time
}
