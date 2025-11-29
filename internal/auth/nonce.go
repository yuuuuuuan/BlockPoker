package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

func generateNonce() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (h *Handler) PostNonce(c *gin.Context) {
	nonce, err := generateNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	// 保存到 Redis 或内存（绑定 walletAddress）
	// 防止重放
	h.nonceStore[nonce] = true

	c.JSON(200, gin.H{"nonce": nonce})
}

func (h *Handler) GetNonce(c *gin.Context) {
	nonce, err := generateNonce()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}

	// 保存到 Redis 或内存（绑定 walletAddress）
	// 防止重放
	h.nonceStore[nonce] = true

	c.JSON(200, gin.H{"nonce": nonce})
}
