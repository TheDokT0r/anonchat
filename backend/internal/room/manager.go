package room

import (
	"context"
	"errors"
	"strings"

	"anonchat/backend/internal/names"
	"anonchat/backend/internal/redisclient"
)

var (
	ErrRoomFull    = errors.New("room is full")
	ErrInvalidRoom = errors.New("invalid room name")
)

type JoinResult struct {
	RoomName     string
	AssignedName string
	Users        []string
}

type Manager struct {
	store    redisclient.Store
	pubsub   redisclient.PubSub
	maxUsers int64
}

func NewManager(store redisclient.Store, pubsub redisclient.PubSub, maxUsers int64) *Manager {
	return &Manager{
		store:    store,
		pubsub:   pubsub,
		maxUsers: maxUsers,
	}
}

func normalizeRoomName(name string) (string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || len(name) > 50 {
		return "", ErrInvalidRoom
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return "", ErrInvalidRoom
		}
	}
	return name, nil
}

func (m *Manager) Join(ctx context.Context, roomName, connID, ip string) (*JoinResult, error) {
	normalized, err := normalizeRoomName(roomName)
	if err != nil {
		return nil, err
	}

	count, err := m.store.UserCount(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if count >= m.maxUsers {
		return nil, ErrRoomFull
	}

	existingUsers, err := m.store.GetUsers(ctx, normalized)
	if err != nil {
		return nil, err
	}
	existingNames := make([]string, 0, len(existingUsers))
	for _, name := range existingUsers {
		existingNames = append(existingNames, name)
	}

	assignedName := names.Generate(existingNames)

	if err := m.store.AddUser(ctx, normalized, connID, assignedName); err != nil {
		return nil, err
	}

	allUsers, err := m.store.GetUsers(ctx, normalized)
	if err != nil {
		return nil, err
	}
	userNames := make([]string, 0, len(allUsers))
	for _, name := range allUsers {
		userNames = append(userNames, name)
	}

	return &JoinResult{
		RoomName:     normalized,
		AssignedName: assignedName,
		Users:        userNames,
	}, nil
}

func (m *Manager) Leave(ctx context.Context, roomName, connID string) error {
	_, err := m.store.RemoveUser(ctx, roomName, connID)
	return err
}

func (m *Manager) GetUserName(ctx context.Context, roomName, connID string) (string, error) {
	users, err := m.store.GetUsers(ctx, roomName)
	if err != nil {
		return "", err
	}
	return users[connID], nil
}

func (m *Manager) GetUserList(ctx context.Context, roomName string) ([]string, error) {
	users, err := m.store.GetUsers(ctx, roomName)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(users))
	for _, name := range users {
		result = append(result, name)
	}
	return result, nil
}

func (m *Manager) Publish(ctx context.Context, roomName string, data []byte) error {
	return m.pubsub.Publish(ctx, roomName, data)
}

func (m *Manager) Subscribe(ctx context.Context, roomName string) (<-chan []byte, func(), error) {
	return m.pubsub.Subscribe(ctx, roomName)
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.store.Ping(ctx)
}

func (m *Manager) RoomExists(ctx context.Context, roomName string) (bool, error) {
	normalized, err := normalizeRoomName(roomName)
	if err != nil {
		return false, err
	}
	return m.store.RoomExists(ctx, normalized)
}
