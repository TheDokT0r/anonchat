package ws

import "encoding/json"

// Client → Server

type JoinMsg struct {
	RoomName string `json:"roomName"`
}

type LeaveMsg struct{}

type ChatSendMsg struct {
	Content string `json:"content"`
}

type TypingSendMsg struct {
	IsTyping bool `json:"isTyping"`
}

type ClientMessage struct {
	Join   *JoinMsg       `json:"join,omitempty"`
	Leave  *LeaveMsg      `json:"leave,omitempty"`
	Chat   *ChatSendMsg   `json:"chat,omitempty"`
	Typing *TypingSendMsg `json:"typing,omitempty"`
}

// Server → Client

type RoomJoinedData struct {
	RoomName     string   `json:"roomName"`
	AssignedName string   `json:"assignedName"`
	Users        []string `json:"users"`
}

type ChatData struct {
	SenderName string `json:"senderName"`
	Content    string `json:"content"`
	Timestamp  int64  `json:"timestamp"`
}

type PresenceData struct {
	Users []string `json:"users"`
}

type TypingData struct {
	UserName string `json:"userName"`
	IsTyping bool   `json:"isTyping"`
}

type ErrorData struct {
	Message string `json:"message"`
}

type ServerMessage struct {
	RoomJoined *RoomJoinedData `json:"roomJoined,omitempty"`
	Chat       *ChatData       `json:"chat,omitempty"`
	Presence   *PresenceData   `json:"presence,omitempty"`
	Typing     *TypingData     `json:"typing,omitempty"`
	Error      *ErrorData      `json:"error,omitempty"`
}

func ParseServerMessage(data []byte) (ServerMessage, error) {
	var msg ServerMessage
	err := json.Unmarshal(data, &msg)
	return msg, err
}
