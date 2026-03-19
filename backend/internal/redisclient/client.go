package redisclient

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

type Store interface {
	AddUser(ctx context.Context, room, connID, displayName string) error
	RemoveUser(ctx context.Context, room, connID string) (empty bool, err error)
	GetUsers(ctx context.Context, room string) (map[string]string, error)
	UserCount(ctx context.Context, room string) (int64, error)
	RoomExists(ctx context.Context, room string) (bool, error)
	Ping(ctx context.Context) error
}

type PubSub interface {
	Publish(ctx context.Context, room string, data []byte) error
	Subscribe(ctx context.Context, room string) (<-chan []byte, func(), error)
}

type RedisClient struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *RedisClient {
	return &RedisClient{rdb: rdb}
}

func roomKey(room string) string {
	return fmt.Sprintf("room:%s:users", room)
}

func channelKey(room string) string {
	return fmt.Sprintf("room:%s:channel", room)
}

func (c *RedisClient) AddUser(ctx context.Context, room, connID, displayName string) error {
	return c.rdb.HSet(ctx, roomKey(room), connID, displayName).Err()
}

func (c *RedisClient) RemoveUser(ctx context.Context, room, connID string) (bool, error) {
	pipe := c.rdb.Pipeline()
	pipe.HDel(ctx, roomKey(room), connID)
	lenCmd := pipe.HLen(ctx, roomKey(room))
	_, err := pipe.Exec(ctx)
	if err != nil { return false, err }
	remaining := lenCmd.Val()
	if remaining == 0 {
		c.rdb.Del(ctx, roomKey(room))
		return true, nil
	}
	return false, nil
}

func (c *RedisClient) GetUsers(ctx context.Context, room string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, roomKey(room)).Result()
}

func (c *RedisClient) UserCount(ctx context.Context, room string) (int64, error) {
	return c.rdb.HLen(ctx, roomKey(room)).Result()
}

func (c *RedisClient) RoomExists(ctx context.Context, room string) (bool, error) {
	n, err := c.rdb.Exists(ctx, roomKey(room)).Result()
	return n > 0, err
}

func (c *RedisClient) Publish(ctx context.Context, room string, data []byte) error {
	return c.rdb.Publish(ctx, channelKey(room), data).Err()
}

func (c *RedisClient) Subscribe(ctx context.Context, room string) (<-chan []byte, func(), error) {
	sub := c.rdb.Subscribe(ctx, channelKey(room))
	_, err := sub.Receive(ctx)
	if err != nil { return nil, nil, err }
	ch := make(chan []byte, 64)
	go func() {
		defer close(ch)
		for msg := range sub.Channel() {
			ch <- []byte(msg.Payload)
		}
	}()
	cancel := func() { sub.Close() }
	return ch, cancel, nil
}

func (c *RedisClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
