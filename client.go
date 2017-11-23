package socks

import (
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
		mt, message, err := c.webSocket.ReadMessage()
		if mt == -1 {
			return
		}
		c.server.Event <- RequestEvent{
			Data:   message,
			Client: c,
			Error:  err,
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
				c.send(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.send(websocket.TextMessage, bytes); err != nil {
				return
			}
		case <-pingTicker.C:
			if err := c.send(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) send(messageType int, bytes []byte) error {
	c.webSocket.SetWriteDeadline(time.Now().Add(writeWait))
	return c.webSocket.WriteMessage(messageType, bytes)
}
