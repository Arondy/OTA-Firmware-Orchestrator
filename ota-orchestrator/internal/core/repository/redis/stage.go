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

func (r *StageCacheRepo) SetMinSampleSize(ctx context.Context, id uuid.UUID, size int) error {
	key := fmt.Sprintf("%s:%s:min_sample_size", r.key, id)
	return r.client.Set(ctx, key, size, 0).Err()
}

func (r *StageCacheRepo) DeleteMinSampleSize(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:min_sample_size", r.key, id)
	return r.client.Del(ctx, key).Err()
}

func (r *StageCacheRepo) SetSuccessThreshold(ctx context.Context, id uuid.UUID, threshold float32) error {
	key := fmt.Sprintf("%s:%s:success_threshold", r.key, id)
	return r.client.Set(ctx, key, threshold, 0).Err()
}

func (r *StageCacheRepo) DeleteSuccessThreshold(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("%s:%s:success_threshold", r.key, id)
	return r.client.Del(ctx, key).Err()
}
