package update

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newBucketService() *UpdateService {
	return &UpdateService{}
}

func TestCalculateBucket_Deterministic(t *testing.T) {
	deviceID := uuid.New()
	campaignID := uuid.New()

	first := newBucketService().calculateBucket(deviceID, campaignID)
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, newBucketService().calculateBucket(deviceID, campaignID))
	}
}

func TestCalculateBucket_Range(t *testing.T) {
	for i := 0; i < 10000; i++ {
		deviceID := uuid.New()
		campaignID := uuid.New()

		bucket := newBucketService().calculateBucket(deviceID, campaignID)

		assert.GreaterOrEqual(t, int(bucket), 1)
		assert.LessOrEqual(t, int(bucket), 100)
	}
}

func TestCalculateBucket_Distribution(t *testing.T) {
	const total = 1000000
	counts := make([]int, 101)

	for i := 0; i < total; i++ {
		deviceID := uuid.New()
		campaignID := uuid.New()

		bucket := int(newBucketService().calculateBucket(deviceID, campaignID))
		counts[bucket]++
	}

	expected := float64(total) / 100.0
	for bucket := 1; bucket <= 100; bucket++ {
		diff := math.Abs(float64(counts[bucket])-expected) / expected
		assert.Less(t, diff, 0.05, "bucket %d deviates by %.2f%%", bucket, diff*100)
	}
}

func TestCalculateBucket_DifferentCampaigns(t *testing.T) {
	deviceID := uuid.New()
	firstCampaign := uuid.New()
	baseBucket := int(newBucketService().calculateBucket(deviceID, firstCampaign))

	const total = 1000
	differ := 0
	for i := 0; i < total; i++ {
		campaignID := uuid.New()
		if int(newBucketService().calculateBucket(deviceID, campaignID)) != baseBucket {
			differ++
		}
	}

	assert.Greater(t, differ, total/2, "expected most devices to be re-bucketed across campaigns")
}
