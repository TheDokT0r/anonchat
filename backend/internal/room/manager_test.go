package room_test

import (
	"context"
	"testing"

	"anonchat/backend/internal/room"
)

// mockStore implements redisclient.Store for unit testing.
type mockStore struct {
	rooms map[string]map[string]string // room -> connID -> displayName
}

func newMockStore() *mockStore {
	return &mockStore{rooms: make(map[string]map[string]string)}
}

func (m *mockStore) AddUser(_ context.Context, rm, connID, name string) error {
	if m.rooms[rm] == nil {
		m.rooms[rm] = make(map[string]string)
	}
	m.rooms[rm][connID] = name
	return nil
}

func (m *mockStore) RemoveUser(_ context.Context, rm, connID string) (bool, error) {
	delete(m.rooms[rm], connID)
	empty := len(m.rooms[rm]) == 0
	if empty {
		delete(m.rooms, rm)
	}
	return empty, nil
}

func (m *mockStore) GetUsers(_ context.Context, rm string) (map[string]string, error) {
	if m.rooms[rm] == nil {
		return map[string]string{}, nil
	}
	return m.rooms[rm], nil
}

func (m *mockStore) UserCount(_ context.Context, rm string) (int64, error) {
	return int64(len(m.rooms[rm])), nil
}

func (m *mockStore) RoomExists(_ context.Context, rm string) (bool, error) {
	_, ok := m.rooms[rm]
	return ok, nil
}

func (m *mockStore) Ping(_ context.Context) error { return nil }

// mockPubSub implements redisclient.PubSub for unit testing.
type mockPubSub struct {
	published [][]byte
}

func (m *mockPubSub) Publish(_ context.Context, _ string, data []byte) error {
	m.published = append(m.published, data)
	return nil
}

func (m *mockPubSub) Subscribe(_ context.Context, _ string) (<-chan []byte, func(), error) {
	ch := make(chan []byte, 64)
	return ch, func() { close(ch) }, nil
}

func TestJoin_NewRoom(t *testing.T) {
	store := newMockStore()
	pubsub := &mockPubSub{}
	mgr := room.NewManager(store, pubsub, 50)

	result, err := mgr.Join(context.Background(), "general", "conn1", "192.168.1.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.RoomName != "general" {
		t.Fatalf("expected room 'general', got %q", result.RoomName)
	}
	if result.AssignedName == "" {
		t.Fatal("expected assigned name")
	}

	users, _ := store.GetUsers(context.Background(), "general")
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
}

func TestJoin_ExistingRoom(t *testing.T) {
	store := newMockStore()
	pubsub := &mockPubSub{}
	mgr := room.NewManager(store, pubsub, 50)

	mgr.Join(context.Background(), "general", "conn1", "192.168.1.1")
	result, err := mgr.Join(context.Background(), "general", "conn2", "192.168.1.2")
	if err != nil {
		t.Fatal(err)
	}

	users, _ := store.GetUsers(context.Background(), "general")
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if result.AssignedName == "" {
		t.Fatal("expected assigned name for second user")
	}
}

func TestJoin_RoomFull(t *testing.T) {
	store := newMockStore()
	pubsub := &mockPubSub{}
	mgr := room.NewManager(store, pubsub, 2)

	mgr.Join(context.Background(), "general", "conn1", "192.168.1.1")
	mgr.Join(context.Background(), "general", "conn2", "192.168.1.2")

	_, err := mgr.Join(context.Background(), "general", "conn3", "192.168.1.3")
	if err == nil {
		t.Fatal("expected error for full room")
	}
}

func TestLeave_LastUser_DeletesRoom(t *testing.T) {
	store := newMockStore()
	pubsub := &mockPubSub{}
	mgr := room.NewManager(store, pubsub, 50)

	mgr.Join(context.Background(), "general", "conn1", "192.168.1.1")
	err := mgr.Leave(context.Background(), "general", "conn1")
	if err != nil {
		t.Fatal(err)
	}

	exists, _ := store.RoomExists(context.Background(), "general")
	if exists {
		t.Fatal("expected room to be deleted after last user leaves")
	}
}

func TestJoin_NormalizesRoomName(t *testing.T) {
	store := newMockStore()
	pubsub := &mockPubSub{}
	mgr := room.NewManager(store, pubsub, 50)

	result, _ := mgr.Join(context.Background(), "General", "conn1", "192.168.1.1")
	if result.RoomName != "general" {
		t.Fatalf("expected normalized room name 'general', got %q", result.RoomName)
	}
}
