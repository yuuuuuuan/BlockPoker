package matchmaker

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// POST /match/join  body: {address, pool, tableSize}
func (h *Handler) Join(c *gin.Context) {
	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	room, queued, err := h.svc.Join(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if queued {
		c.JSON(http.StatusOK, JoinResponse{
			Queued: true, Pool: req.Pool, TableSize: req.TableSize,
		})
		return
	}
	c.JSON(http.StatusOK, JoinResponse{
		Queued: false, Pool: room.Pool, TableSize: room.TableSize, RoomID: room.ID, Players: room.Players,
	})
}

// POST /match/cancel body: {address}
func (h *Handler) Cancel(c *gin.Context) {
	var req CancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.Cancel(c.Request.Context(), req.Address); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
