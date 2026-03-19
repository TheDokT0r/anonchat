package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"anonchat/backend/internal/config"
	"anonchat/backend/internal/ratelimit"
	"anonchat/backend/internal/room"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Handler struct {
	mux    *http.ServeMux
	mgr    *room.Manager
	msgLim *ratelimit.MessageLimiter
	ipLim  *ratelimit.IPRoomLimiter
	cfg    config.Config

	connsMu sync.Mutex
	conns   map[string]*wsConn
}

func New(mgr *room.Manager, msgLim *ratelimit.MessageLimiter, ipLim *ratelimit.IPRoomLimiter, cfg config.Config) *Handler {
	h := &Handler{
		mux:    http.NewServeMux(),
		mgr:    mgr,
		msgLim: msgLim,
		ipLim:  ipLim,
		cfg:    cfg,
		conns:  make(map[string]*wsConn),
	}
	h.mux.HandleFunc("/ws", h.handleWS)
	h.mux.HandleFunc("/healthz", h.handleHealthz)
	h.mux.HandleFunc("/readyz", h.handleReadyz)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) Shutdown(ctx context.Context) {
	h.connsMu.Lock()
	conns := make([]*wsConn, 0, len(h.conns))
	for _, c := range h.conns {
		conns = append(conns, c)
	}
	h.connsMu.Unlock()

	for _, c := range conns {
		c.cleanup(ctx)
	}
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) handleReadyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := h.mgr.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("redis unavailable"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *Handler) handleWS(w http.ResponseWriter, r *http.Request) {
	opts := &websocket.AcceptOptions{}
	if len(h.cfg.AllowedOrigins) > 0 && h.cfg.AllowedOrigins[0] == "*" {
		opts.InsecureSkipVerify = true
	} else {
		opts.OriginPatterns = h.cfg.AllowedOrigins
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		slog.Error("websocket accept failed", "error", err)
		return
	}

	ip := extractIP(r)
	connID := generateConnID()

	c := &wsConn{
		conn:    conn,
		handler: h,
		connID:  connID,
		ip:      ip,
	}

	h.connsMu.Lock()
	h.conns[connID] = c
	h.connsMu.Unlock()

	c.serve(r.Context())
}

type wsConn struct {
	conn    *websocket.Conn
	handler *Handler
	connID  string
	ip      string

	mu        sync.Mutex
	roomName  string
	userName  string
	subCancel func()
}

func (c *wsConn) serve(baseCtx context.Context) {
	ctx, cancel := context.WithCancel(baseCtx)
	defer cancel()
	defer c.cleanup(context.Background())

	for {
		var msg map[string]json.RawMessage
		err := wsjson.Read(ctx, c.conn, &msg)
		if err != nil {
			return
		}
		c.handleMessage(ctx, msg)
	}
}

func (c *wsConn) handleMessage(ctx context.Context, msg map[string]json.RawMessage) {
	if _, ok := msg["join"]; ok {
		c.handleJoin(ctx, msg["join"])
		return
	}
	if _, ok := msg["leave"]; ok {
		c.handleLeave(ctx)
		return
	}
	if _, ok := msg["chat"]; ok {
		c.handleChat(ctx, msg["chat"])
		return
	}
	if _, ok := msg["typing"]; ok {
		c.handleTyping(ctx, msg["typing"])
		return
	}
}

func (c *wsConn) handleJoin(ctx context.Context, raw json.RawMessage) {
	var join struct {
		RoomName string `json:"roomName"`
	}
	if err := json.Unmarshal(raw, &join); err != nil {
		c.sendError(ctx, "invalid join message")
		return
	}

	exists, _ := c.handler.mgr.RoomExists(ctx, join.RoomName)
	if !exists && !c.handler.ipLim.Allow(c.ip) {
		c.sendError(ctx, "too many rooms created, slow down")
		return
	}

	if c.roomName != "" {
		c.handleLeave(ctx)
	}

	result, err := c.handler.mgr.Join(ctx, join.RoomName, c.connID, c.ip)
	if err != nil {
		c.sendError(ctx, err.Error())
		return
	}

	c.mu.Lock()
	c.roomName = result.RoomName
	c.userName = result.AssignedName
	c.mu.Unlock()

	resp := map[string]interface{}{
		"roomJoined": map[string]interface{}{
			"roomName":     result.RoomName,
			"assignedName": result.AssignedName,
			"users":        result.Users,
		},
	}
	wsjson.Write(ctx, c.conn, resp)

	c.broadcastPresence(ctx)

	ch, unsub, err := c.handler.mgr.Subscribe(ctx, result.RoomName)
	if err != nil {
		slog.Error("subscribe failed", "error", err)
		return
	}

	c.mu.Lock()
	c.subCancel = unsub
	c.mu.Unlock()

	go func() {
		for data := range ch {
			var msg map[string]json.RawMessage
			if json.Unmarshal(data, &msg) == nil {
				if senderID, ok := msg["_sender"]; ok {
					var sender string
					json.Unmarshal(senderID, &sender)
					if sender == c.connID {
						continue
					}
				}
				delete(msg, "_sender")
				wsjson.Write(ctx, c.conn, msg)
			}
		}
	}()
}

func (c *wsConn) handleLeave(ctx context.Context) {
	c.mu.Lock()
	roomName := c.roomName
	if c.subCancel != nil {
		c.subCancel()
		c.subCancel = nil
	}
	c.roomName = ""
	c.userName = ""
	c.mu.Unlock()

	if roomName == "" {
		return
	}

	c.handler.mgr.Leave(ctx, roomName, c.connID)
	c.broadcastPresenceForRoom(ctx, roomName)
}

func (c *wsConn) handleChat(ctx context.Context, raw json.RawMessage) {
	c.mu.Lock()
	roomName := c.roomName
	userName := c.userName
	c.mu.Unlock()

	if roomName == "" {
		c.sendError(ctx, "not in a room")
		return
	}

	if !c.handler.msgLim.Allow(c.connID) {
		c.sendError(ctx, "slow down")
		return
	}

	var chat struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &chat); err != nil {
		return
	}

	if len(chat.Content) == 0 || len(chat.Content) > 2000 {
		c.sendError(ctx, "message must be 1-2000 characters")
		return
	}

	msg := map[string]interface{}{
		"chat": map[string]interface{}{
			"senderName": userName,
			"content":    chat.Content,
			"timestamp":  time.Now().UnixMilli(),
		},
		"_sender": c.connID,
	}
	data, _ := json.Marshal(msg)
	c.handler.mgr.Publish(ctx, roomName, data)
}

func (c *wsConn) handleTyping(ctx context.Context, raw json.RawMessage) {
	c.mu.Lock()
	roomName := c.roomName
	userName := c.userName
	c.mu.Unlock()

	if roomName == "" {
		return
	}

	var typing struct {
		IsTyping bool `json:"isTyping"`
	}
	if err := json.Unmarshal(raw, &typing); err != nil {
		return
	}

	msg := map[string]interface{}{
		"typing": map[string]interface{}{
			"userName": userName,
			"isTyping": typing.IsTyping,
		},
		"_sender": c.connID,
	}
	data, _ := json.Marshal(msg)
	c.handler.mgr.Publish(ctx, roomName, data)
}

func (c *wsConn) broadcastPresence(ctx context.Context) {
	c.mu.Lock()
	roomName := c.roomName
	c.mu.Unlock()
	c.broadcastPresenceForRoom(ctx, roomName)
}

func (c *wsConn) broadcastPresenceForRoom(ctx context.Context, roomName string) {
	if roomName == "" {
		return
	}
	users, err := c.handler.mgr.GetUserList(ctx, roomName)
	if err != nil {
		return
	}
	msg := map[string]interface{}{
		"presence": map[string]interface{}{
			"users": users,
		},
		"_sender": "",
	}
	data, _ := json.Marshal(msg)
	c.handler.mgr.Publish(ctx, roomName, data)
}

func (c *wsConn) sendError(ctx context.Context, message string) {
	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
		},
	}
	wsjson.Write(ctx, c.conn, resp)
}

func (c *wsConn) cleanup(ctx context.Context) {
	c.handleLeave(ctx)
	c.handler.msgLim.Remove(c.connID)
	c.handler.connsMu.Lock()
	delete(c.handler.conns, c.connID)
	c.handler.connsMu.Unlock()
	c.conn.CloseNow()
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

var connCounter atomic.Uint64

func generateConnID() string {
	id := connCounter.Add(1)
	return fmt.Sprintf("conn-%d-%d", time.Now().UnixNano(), id)
}
