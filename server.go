package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  maxMessageSize,
	WriteBufferSize: maxMessageSize,
}

type BroadcastWriter chan []byte

type Server struct {
	Broadcast  BroadcastWriter
	register   chan *Client
	unregister chan *Client
	clients    map[*Client]bool
	handler    func([]byte, *Client) error
}

func NewServer(handler func([]byte, *Client) error) *Server {
	newServer := &Server{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Broadcast:  make(chan []byte),
		clients:    make(map[*Client]bool),
		handler:    handler,
	}
	return newServer
}

func (s *Server) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf("%v method is not allowed", r.Method), 405)
		return
	}
	webSocket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		server:    s,
		webSocket: webSocket,
		Send:      make(chan []byte, maxMessageSize),
	}
	s.register <- client
	go client.write()
	client.read()
}

func (s *Server) Run() {
	go func() {
		for {
			select {
			case client := <-s.register:
				s.clients[client] = true
			case client := <-s.unregister:
				if !s.clients[client] {
					continue
				}
				delete(s.clients, client)
				client.webSocket.Close()
				close(client.Send)
			case message := <-s.Broadcast:
				s.broadcast(message)
			}
		}
	}()
}

func (s *Server) broadcast(bytes []byte) {
	for c := range s.clients {
		select {
		case c.Send <- bytes:
			break
		default:
			s.unregister <- c
		}
	}
}
