package server

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024
)

type Client struct {
	server    *Server
	webSocket *websocket.Conn
	Send      chan []byte
}

func (c *Client) read() {
	defer func() {
		c.server.unregister <- c
	}()
	c.webSocket.SetReadLimit(maxMessageSize)
	c.webSocket.SetReadDeadline(time.Now().Add(pongWait))
	c.webSocket.SetPongHandler(func(string) error {
		c.webSocket.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.webSocket.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		c.server.Event <- RequestEvent{
			Data:   message,
			Client: c,
		}
	}
}

func (c *Client) write() {
	pingTicker := time.NewTicker(pingPeriod)

	defer func() {
		pingTicker.Stop()
		c.server.unregister <- c
	}()

	for {
		select {
		case bytes, ok := <-c.Send:
			if !ok {
				c.webSocket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.webSocket.WriteMessage(websocket.TextMessage, bytes); err != nil {
				return
			}
		case <-pingTicker.C:
			c.webSocket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.webSocket.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
