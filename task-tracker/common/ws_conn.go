package common

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSConn struct {
	*websocket.Conn
}

func NewWSConn(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WSConn{conn}, nil
}

func (ws *WSConn) WriteMessage(data []byte) error {
	return ws.Conn.WriteMessage(websocket.TextMessage, data)
}
