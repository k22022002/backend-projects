package ws

import (
	"fmt"
	"sync"
	"task-tracker/common"
)

type Client struct {
	Conn   *common.WSConn
	UserID string
	Send   chan []byte
}

type Hub struct {
	Clients    sync.Map
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan common.NotificationJob
}

var WsHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan common.NotificationJob),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients.Store(client.UserID, client)
		case client := <-h.Unregister:
			h.Clients.Delete(client.UserID)
			close(client.Send)
		case msg := <-h.Broadcast:
			key := fmt.Sprintf("%d", msg.TaskID)
			if val, ok := h.Clients.Load(key); ok {
				client := val.(*Client)
				client.Send <- []byte(msg.Message)
			}
		}
	}
}
