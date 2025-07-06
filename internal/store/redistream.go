package store

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type StreamsStore struct {
	Client *redis.Client
}

type Streams interface {
	PublishJob(stream string, payload map[string]interface{}) error
	ReadJob(stream, group, consumer string, block time.Duration) ([]redis.XMessage, error)
	AckJob(stream, group, id string) error
}

func NewStreamsStore(addr string) *StreamsStore {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &StreamsStore{Client: rdb}
}

func (s *StreamsStore) PublishJob(stream string, payload map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.Client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: payload,
	}).Result()
	return err
}

func (s *StreamsStore) ReadJob(stream, group, consumer string, block time.Duration) ([]redis.XMessage, error) {
	ctx := context.Background()
	res, err := s.Client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Block:    block,
		Count:    1,
	}).Result()
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0].Messages, nil
}

func (s *StreamsStore) AckJob(stream, group, id string) error {
	ctx := context.Background()
	return s.Client.XAck(ctx, stream, group, id).Err()
}

func (s *StreamsStore) Ping() error {
	ctx := context.Background()
	_, err := s.Client.Ping(ctx).Result()
	return err
}
