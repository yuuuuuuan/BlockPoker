package websocket

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// GET /ws  (需带 JWT，middleware 已在 main.go 中加入)
func ServeWS(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		addr := c.GetString("address") // JWT middleware 注入

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := &Client{
			Address: addr,
			Conn:    conn,
			Send:    make(chan OutgoingMessage, 32),
			Hub:     hub,
		}

		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
