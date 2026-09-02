package redis

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type StageCacheRepo struct {
	client *redis.Client
	key    string
}

func NewStageCacheRepo(rdb *redis.Client) *StageCacheRepo {
	return &StageCacheRepo{
		client: rdb,
		key:    "stage",
	}
}

func (r *StageCacheRepo) GetMinSampleSize(ctx context.Context, id uuid.UUID) (int, error) {
	key := fmt.Sprintf("%s:%s:min_sample_size", r.key, id)
	return r.client.Get(ctx, key).Int()
}

func (r *StageCacheRepo) GetSuccessThreshold(ctx context.Context, id uuid.UUID) (float32, error) {
	key := fmt.Sprintf("%s:%s:success_threshold", r.key, id)
	return r.client.Get(ctx, key).Float32()
}
