package queue

import (
	"context"
	"fmt"
	"time"

	r "github.com/redis/go-redis/v9"
)

type RedisQ struct{ rdb *r.Client }

func New(rdb *r.Client) *RedisQ { return &RedisQ{rdb} }

func (q *RedisQ) Enqueue(ctx context.Context, tenant string, jobID string, runAt time.Time) error {
	if time.Until(runAt) > 0 {
		return q.rdb.ZAdd(ctx, "delay:"+tenant, r.Z{Score: float64(runAt.Unix()), Member: jobID}).Err()
	}
	return q.rdb.LPush(ctx, "queue:"+tenant, jobID).Err()
}

func (q *RedisQ) Dequeue(ctx context.Context, tenant string, block time.Duration) (string, error) {
	res, err := q.rdb.BRPop(ctx, block, "queue:"+tenant).Result()
	if err != nil {
		return "", err
	}
	if len(res) == 2 {
		return res[1], nil
	}
	return "", nil
}

func (q *RedisQ) MoveDue(ctx context.Context, tenant string, now int64, batch int64) error {
	// fetch due IDs
	ids, err := q.rdb.ZRangeByScore(ctx, "delay:"+tenant, &r.ZRangeBy{Min: "-inf", Max: fmt.Sprintf("%d", now), Offset: 0, Count: batch}).Result()
	if err != nil || len(ids) == 0 {
		return err
	}
	pipe := q.rdb.TxPipeline()
	for _, id := range ids {
		pipe.LPush(ctx, "queue:"+tenant, id)
		pipe.ZRem(ctx, "delay:"+tenant, id)
	}
	_, err = pipe.Exec(ctx)
	return err
}
