package websocket

import (
	"log"
	"sync"
)

type HubInterface interface {
	BroadcastToPlayers(addrs []string, msg OutgoingMessage)
	ClientByAddress(addr string) (*Client, bool)
	SendToPlayer(addr string, msg OutgoingMessage)
	Close()
}

type Hub struct {
	clients    map[string]*Client // address -> client
	register   chan *Client
	unregister chan *Client
	broadcast  chan broadcastReq
	sendOne    chan sendReq
	incoming   chan IncomingMessage
	OnIncoming func(IncomingMessage)
	quit       chan struct{}
	mu         sync.RWMutex
}

type broadcastReq struct {
	Addresses []string
	Message   OutgoingMessage
}

type sendReq struct {
	Address string
	Message OutgoingMessage
}

// type incomingReq struct {
// 	From    string
// 	Message IncomingMessage
// }

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan broadcastReq),
		sendOne:    make(chan sendReq),
		incoming:   make(chan IncomingMessage),
		quit:       make(chan struct{}),
	}
}

func (h *Hub) Run() {

	log.Println("Hub started")

	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.Address] = c
			log.Printf("Hub.register -> %s (当前连接数: %d)", c.Address, len(h.clients))

			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c.Address]; ok {
				delete(h.clients, c.Address)
				log.Printf("Hub.unregister -> %s (当前连接数: %d)", c.Address, len(h.clients))

				close(c.Send)
			}
			h.mu.Unlock()

		case req := <-h.broadcast:
			for _, addr := range req.Addresses {
				if client, ok := h.clients[addr]; ok {
					client.Send <- req.Message
				}
			}

		case req := <-h.sendOne:
			if client, ok := h.clients[req.Address]; ok {
				//client.Send <- req.Message
				select {
				case client.Send <- req.Message:
				default:
					// optional: 丢弃 / 记录日志 / 将消息转移到慢队列
				}

			}

		case req := <-h.incoming:
			// !!!! 这里把玩家消息统一转发给游戏层（Engine / GameManager）
			if h.OnIncoming != nil {
				h.OnIncoming(req)
			}

		case <-h.quit:
			for _, c := range h.clients {
				close(c.Send)
			}
		}
	}
}

// Broadcast to multiple players
func (h *Hub) BroadcastToPlayers(addrs []string, msg OutgoingMessage) {
	log.Println("BroadcastToPlayers successed")
	h.broadcast <- broadcastReq{
		Addresses: addrs,
		Message:   msg,
	}
}

// Send to a single player (safe concurrent)
func (h *Hub) SendToPlayer(addr string, msg OutgoingMessage) {
	log.Println("SendToPlayer successed")
	h.sendOne <- sendReq{
		Address: addr,
		Message: msg,
	}
}

// Lookup for a player client by address
func (h *Hub) ClientByAddress(addr string) (*Client, bool) {
	c, ok := h.clients[addr]
	return c, ok
}

func (h *Hub) Close() {
	close(h.quit)
}
