package ws

import (
	"context"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Client struct {
	url  string
	conn *websocket.Conn
	mu   sync.Mutex
}

func New(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Connect(ctx context.Context) (<-chan ServerMessage, error) {
	conn, _, err := websocket.Dial(ctx, c.url, nil)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	ch := make(chan ServerMessage, 64)
	go c.readLoop(ctx, ch)
	return ch, nil
}

func (c *Client) Send(msg ClientMessage) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsjson.Write(ctx, conn, msg)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close(websocket.StatusNormalClosure, "bye")
		c.conn = nil
	}
}

func (c *Client) readLoop(ctx context.Context, ch chan<- ServerMessage) {
	defer close(ch)
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		msg, err := ParseServerMessage(data)
		if err != nil {
			continue
		}
		select {
		case ch <- msg:
		case <-ctx.Done():
			return
		}
	}
}
