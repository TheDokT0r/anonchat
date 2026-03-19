package redisclient_test

import (
	"context"
	"testing"

	"anonchat/backend/internal/redisclient"
	"github.com/redis/go-redis/v9"
)

func setupClient(t *testing.T) *redisclient.RedisClient {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("Redis not available:", err)
	}
	rdb.FlushDB(context.Background())
	return redisclient.New(rdb)
}

func TestAddUser_And_GetUsers(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()
	err := c.AddUser(ctx, "general", "conn1", "Blue Fox")
	if err != nil { t.Fatal(err) }
	users, err := c.GetUsers(ctx, "general")
	if err != nil { t.Fatal(err) }
	if len(users) != 1 || users["conn1"] != "Blue Fox" {
		t.Fatalf("unexpected users: %v", users)
	}
}

func TestRemoveUser_LastUser_DeletesRoom(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()
	c.AddUser(ctx, "general", "conn1", "Blue Fox")
	empty, err := c.RemoveUser(ctx, "general", "conn1")
	if err != nil { t.Fatal(err) }
	if !empty { t.Fatal("expected room to be empty") }
	users, err := c.GetUsers(ctx, "general")
	if err != nil { t.Fatal(err) }
	if len(users) != 0 { t.Fatalf("expected empty room, got: %v", users) }
}

func TestRoomExists(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()
	exists, err := c.RoomExists(ctx, "nonexistent")
	if err != nil { t.Fatal(err) }
	if exists { t.Fatal("expected room to not exist") }
	c.AddUser(ctx, "general", "conn1", "Blue Fox")
	exists, err = c.RoomExists(ctx, "general")
	if err != nil { t.Fatal(err) }
	if !exists { t.Fatal("expected room to exist") }
}

func TestUserCount(t *testing.T) {
	c := setupClient(t)
	ctx := context.Background()
	c.AddUser(ctx, "general", "conn1", "Blue Fox")
	c.AddUser(ctx, "general", "conn2", "Red Panda")
	count, err := c.UserCount(ctx, "general")
	if err != nil { t.Fatal(err) }
	if count != 2 { t.Fatalf("expected 2 users, got %d", count) }
}
