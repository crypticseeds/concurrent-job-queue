package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/crypticseeds/concurrent-job-queue/internal/task"
	"github.com/redis/go-redis/v9"
)

// RedisQueue implements the Queue interface using Redis Streams.
type RedisQueue struct {
	client        *redis.Client
	streamName    string
	consumerGroup string
	consumerName  string
}

// NewRedisQueue initializes a new Redis-based queue.
func NewRedisQueue(client *redis.Client, streamName, consumerGroup, consumerName string) (*RedisQueue, error) {
	rq := &RedisQueue{
		client:        client,
		streamName:    streamName,
		consumerGroup: consumerGroup,
		consumerName:  consumerName,
	}

	// Create consumer group if it doesn't exist
	ctx := context.Background()
	err := client.XGroupCreateMkStream(ctx, streamName, consumerGroup, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return rq, nil
}

// Enqueue adds a task to the Redis stream.
func (rq *RedisQueue) Enqueue(t *task.Task) error {
	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	err = rq.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: rq.streamName,
		ID:     "*", // Auto-generate ID
		Values: map[string]interface{}{
			"task_id": t.ID,
			"data":    payload,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to add task to stream: %w", err)
	}

	return nil
}

// Dequeue removes a task from the Redis stream using a consumer group.
func (rq *RedisQueue) Dequeue(ctx context.Context) (*task.Task, error) {
	// 1. Try to read pending messages for this consumer (re-processing)
	t, err := rq.readPending(ctx)
	if err != nil {
		return nil, err
	}
	if t != nil {
		return t, nil
	}

	// 2. Periodic AutoClaim to recover from dead consumers
	// In a real high-performance system, this might be a separate background process
	// But for simplicity, we'll do it occasionally or before blocking.
	if time.Now().Unix()%10 == 0 {
		t, err := rq.autoClaim(ctx)
		if err == nil && t != nil {
			return t, nil
		}
	}

	for {
		streams, err := rq.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    rq.consumerGroup,
			Consumer: rq.consumerName,
			Streams:  []string{rq.streamName, ">"}, // ">" means new messages
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, fmt.Errorf("failed to read from group: %w", err)
		}

		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			return rq.parseMessage(streams[0].Messages[0])
		}
	}
}

func (rq *RedisQueue) readPending(ctx context.Context) (*task.Task, error) {
	streams, err := rq.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    rq.consumerGroup,
		Consumer: rq.consumerName,
		Streams:  []string{rq.streamName, "0"}, // "0" means pending messages for this consumer
		Count:    1,
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to read pending from group: %w", err)
	}

	if len(streams) > 0 && len(streams[0].Messages) > 0 {
		return rq.parseMessage(streams[0].Messages[0])
	}
	return nil, nil
}

func (rq *RedisQueue) autoClaim(ctx context.Context) (*task.Task, error) {
	// Claim tasks that have been pending for more than 30 seconds
	msgs, _, err := rq.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   rq.streamName,
		Group:    rq.consumerGroup,
		Consumer: rq.consumerName,
		MinIdle:  30 * time.Second,
		Count:    1,
		Start:    "0-0",
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to autoclaim: %w", err)
	}

	if len(msgs) > 0 {
		return rq.parseMessage(msgs[0])
	}
	return nil, nil
}

func (rq *RedisQueue) parseMessage(msg redis.XMessage) (*task.Task, error) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message format: missing data")
	}

	var t task.Task
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata["redis_msg_id"] = msg.ID

	return &t, nil
}

// Ack acknowledges that a task has been processed.
func (rq *RedisQueue) Ack(t *task.Task) error {
	if t == nil {
		return nil
	}
	msgID, ok := t.Metadata["redis_msg_id"]
	if !ok {
		return fmt.Errorf("task %s missing redis_msg_id in metadata", t.ID)
	}

	ctx := context.Background()
	return rq.client.XAck(ctx, rq.streamName, rq.consumerGroup, msgID).Err()
}

// Depth returns the current number of pending tasks.
func (rq *RedisQueue) Depth() (int64, error) {
	ctx := context.Background()
	pending, err := rq.client.XPending(ctx, rq.streamName, rq.consumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// Fail handles task failure and schedules a retry.
func (rq *RedisQueue) Fail(t *task.Task, retryAfter time.Duration) error {
	// Simplified: just let it stay in PEL, it will be retried if not Acked.
	// A more robust implementation would NACK/Claim or re-enqueue.
	return nil
}
