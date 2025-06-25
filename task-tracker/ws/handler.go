package ws

import (
	"log"
	"net/http"
	"strconv"
	"task-tracker/common"
)

func HandleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	userID, err := common.ValidateToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := common.NewWSConn(w, r)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	client := &Client{
		Conn:   conn,
		UserID: strconv.Itoa(userID),
		Send:   make(chan []byte, 256),
	}

	WsHub.Register <- client

	go read(client)
	go write(client)
}

func read(c *Client) {
	defer func() {
		WsHub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			log.Println("WebSocket read error:", err)
			break
		}
	}
}

func write(c *Client) {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(msg); err != nil {
			log.Println("WebSocket write error:", err)
			break
		}
	}
}
