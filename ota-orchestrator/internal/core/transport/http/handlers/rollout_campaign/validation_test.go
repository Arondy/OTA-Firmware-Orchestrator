package rollout_campaign

import (
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/transport/http/handlers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func validCreateCampaignRequest() CreateRolloutCampaignRequest {
	return CreateRolloutCampaignRequest{
		FirmwareVersionID: uuid.New(),
		RolloutStages: []RolloutStageRequest{
			{
				OrderIndex:       0,
				TargetPercent:    50,
				MinSampleSize:    10,
				SuccessThreshold: 0.8,
			},
		},
	}
}

func TestCreateCampaignValidation_StageFieldBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*RolloutStageRequest)
		valid  bool
	}{
		{"baseline stage", func(*RolloutStageRequest) {}, true},
		{"target_percent min=1", func(s *RolloutStageRequest) { s.TargetPercent = 1 }, true},
		{"target_percent max=100", func(s *RolloutStageRequest) { s.TargetPercent = 100 }, true},
		{"target_percent zero", func(s *RolloutStageRequest) { s.TargetPercent = 0 }, false},
		{"target_percent above max", func(s *RolloutStageRequest) { s.TargetPercent = 101 }, false},
		{"min_sample_size min=1", func(s *RolloutStageRequest) { s.MinSampleSize = 1 }, true},
		{"min_sample_size zero", func(s *RolloutStageRequest) { s.MinSampleSize = 0 }, false},
		{"success_threshold max=1", func(s *RolloutStageRequest) { s.SuccessThreshold = 1 }, true},
		{"success_threshold just above zero", func(s *RolloutStageRequest) { s.SuccessThreshold = 0.0001 }, true},
		{"success_threshold zero", func(s *RolloutStageRequest) { s.SuccessThreshold = 0 }, false},
		{"success_threshold negative", func(s *RolloutStageRequest) { s.SuccessThreshold = -0.5 }, false},
		{"success_threshold above max", func(s *RolloutStageRequest) { s.SuccessThreshold = 1.01 }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := validCreateCampaignRequest()
			tt.mutate(&req.RolloutStages[0])

			err := handlers.Validate.Struct(req)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
