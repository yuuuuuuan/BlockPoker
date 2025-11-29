package auth

import (
	"BlockPoker/config"
	"encoding/hex"
	"fmt"
	"strings"

	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
}

type Handler struct {
	nonceStore map[string]bool
}

// 工厂方法：创建 handler
func NewHandler() *Handler {
	return &Handler{
		nonceStore: make(map[string]bool),
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "bad request"})
		return
	}

	// 检查 nonce 是否有效
	if !h.nonceStore[req.Nonce] {
		c.JSON(400, gin.H{"error": "invalid nonce"})
		return
	}
	delete(h.nonceStore, req.Nonce) // 只允许一次

	// -------------------
	// 恢复签名者地址 (核心)
	// -------------------
	//msg := req.Nonce
	msg := "Sign this message to authenticate with BlockPoker. Nonce: " + req.Nonce
	//msgBytes, _ := hex.DecodeString(msg)
	//log.Printf("Verifying login for address=%s with nonce=%s", req.Address, msgBytes)

	// 构造与 MetaMask personal_sign 完全一致的消息
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	//log.Printf("%s", prefix)
	hash := crypto.Keccak256Hash([]byte(prefix))

	// 处理 signature
	sig := req.Signature
	if len(sig) >= 2 && sig[:2] == "0x" {
		sig = sig[2:]
	}

	//log.Printf("signature=%s", sig)
	sigBytes, _ := hex.DecodeString(sig)

	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	//log.Printf("signature bytes=%x", sigBytes)
	// 修正 V 值
	// 恢复公钥
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		c.JSON(400, gin.H{"error": "signature verify failed"})
		return
	}
	//log.Printf("pubKey=%x", crypto.FromECDSAPub(pubKey))
	recovered := crypto.PubkeyToAddress(*pubKey).Hex()

	//log.Printf("Login attempt: claimed=%s, recovered=%s", req.Address, recovered)

	if !strings.EqualFold(recovered, req.Address) {
		c.JSON(401, gin.H{"error": "signature mismatch"})
		return
	}

	// -----------------------------
	// ✓ 签名验证成功 → 生成 JWT
	// -----------------------------
	claims := jwt.MapClaims{
		"sub": req.Address,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtStr, err := token.SignedString([]byte(config.C.JWT.Secret))
	if err != nil {
		c.JSON(500, gin.H{"error": "jwt generation failed"})
		return
	}

	c.JSON(200, gin.H{
		"jwt": jwtStr,
	})
}
