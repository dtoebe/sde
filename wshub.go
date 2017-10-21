package main

import (
	"fmt"
)

// Hub is the structure of each ws document connection
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	docID      string
}

// Message is the structure of each edit in the document as it comes through the ws
type Message struct {
	broadcast []byte
	senderKey string
}

// hubMap is a map of each ws connection key: document id, val: the ws connection
var hubMap map[string]*Hub

// newHub create new ws connection for the doc
func newHub(id string) *Hub {
	return &Hub{
		broadcast:  make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		docID:      id,
	}
}

// *Hub.run runs the new ws connection
func (h *Hub) run() {
	for {
		select {
		// register a new client to a specific ws connection
		case client := <-h.register:
			h.clients[client] = true
			client.send <- &Message{
				broadcast: []byte(fmt.Sprintf(`{"id": "%s"}`, client.key)),
			}
		// de-register existing client
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		// send message to each client
		case message := <-h.broadcast:
			for client := range h.clients {
				// don't send to creator of the broadcast
				if client.key == message.senderKey {
					continue
				}
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
