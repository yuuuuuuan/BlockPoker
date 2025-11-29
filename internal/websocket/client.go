package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Address string
	Conn    *websocket.Conn
	Send    chan OutgoingMessage
	Hub     *Hub
}

const (
	writeWait  = 10 * time.Second    // 单次写超时
	pongWait   = 60 * time.Second    // 读超时
	pingPeriod = (pongWait * 9) / 10 // 心跳发送周期
	//maxMessageSize = 1024 * 4            // 最大4KB
)

// 写协程
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod) // 心跳
	defer func() {
		ticker.Stop()
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	for {
		select {

		// 有消息待发
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub关闭Send，通知前端
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(msg); err != nil {
				return
			}

		// 定时发送 ping 维持连接健康
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// 读协程（此处暂不处理消息，可扩展）
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		var msg IncomingMessage
		if err := c.Conn.ReadJSON(&msg); err != nil {
			return
		}

		c.Hub.incoming <- IncomingMessage{
			From:  c.Address,
			Event: msg.Event,
			Data:  msg.Data,
		}
	}
}
