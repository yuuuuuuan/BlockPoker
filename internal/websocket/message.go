package websocket

type OutgoingMessage struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type IncomingMessage struct {
	From  string      `json:"from"`
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}
