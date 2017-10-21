package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/buger/jsonparser"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 512
)

var (
	newLine = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is the structure of each client connected to a hub
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan *Message
	key  string
}

// readPump isthe message receiver
func (c *Client) readPump() {
	defer c.conn.Close()

	c.conn.SetReadLimit(maxMsgSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Println(err)
			}
			break
		}

		c.hub.broadcast <- &Message{
			broadcast: bytes.TrimSpace(bytes.Replace(msg, newLine, space, -1)),
			senderKey: c.key,
		}
	}
}

// writePump is the client message sender
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer ticker.Stop()
	defer c.conn.Close()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if c.key == msg.senderKey {
				if err := w.Close(); err != nil {
					return
				}
				continue
			}
			w.Write(msg.broadcast)
			go writeToCache(c.hub.docID, msg.broadcast)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// serveWS handles each wsclient connection
func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan *Message),
		key:  r.Header.Get("Sec-WebSocket-Key"),
	}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// writeToCache writes each change to a data store
func writeToCache(id string, body []byte) {
	// TODO: Add user key?
	log.Println(string(body))
	msg, err := jsonparser.GetString(body, "body")
	if err != nil {
		log.Println("ERR:", err.Error())
		return
	}

	var chng DocumentChanges
	err = db.Get(&chng, `
		INSERT INTO document_changes(document_id, body_state, updated)
		VALUES($1, $2, NOW())
		RETURNING *
	`, id, msg)
	if err != nil {
		log.Println(err)
	}
}

// getDoc get the full document
func getDoc(data interface{}, id string) error {
	return db.Get(data, `
		SELECT id, title, body_state AS body, created, updated
		FROM documents
		INNER JOIN document_changes AS dc ON dc.document_id=$1
		WHERE id=$1 ORDER BY updated DESC LIMIT 1
	`, id)
}
