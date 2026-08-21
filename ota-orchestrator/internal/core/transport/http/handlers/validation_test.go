package handlers

import (
	"testing"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rolloutStagesWrapper struct {
	Stages []struct {
		OrderIndex int
	} `validate:"rollout_stages"`
}

type semverWrapper struct {
	V string `validate:"semver"`
}

type updateAttemptResultWrapper struct {
	R domain.UpdateAttemptsResult `validate:"update_attempt_result"`
}

func TestValidateRolloutStages_Empty_ReturnsTrue(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	err := Validate.Struct(w)
	assert.NoError(t, err)
}

func TestValidateRolloutStages_SingleZero_ReturnsTrue(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages, struct{ OrderIndex int }{OrderIndex: 0})
	err := Validate.Struct(w)
	assert.NoError(t, err)
}

func TestValidateRolloutStages_Sequential_ReturnsTrue(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages,
		struct{ OrderIndex int }{OrderIndex: 0},
		struct{ OrderIndex int }{OrderIndex: 1},
		struct{ OrderIndex int }{OrderIndex: 2},
	)
	err := Validate.Struct(w)
	assert.NoError(t, err)
}

func TestValidateRolloutStages_Gap_ReturnsFalse(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages,
		struct{ OrderIndex int }{OrderIndex: 0},
		struct{ OrderIndex int }{OrderIndex: 2},
	)
	err := Validate.Struct(w)
	assert.Error(t, err)
}

func TestValidateRolloutStages_Duplicate_ReturnsFalse(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages,
		struct{ OrderIndex int }{OrderIndex: 0},
		struct{ OrderIndex int }{OrderIndex: 1},
		struct{ OrderIndex int }{OrderIndex: 1},
	)
	err := Validate.Struct(w)
	assert.Error(t, err)
}

func TestValidateRolloutStages_StartsNotFromZero_ReturnsFalse(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages,
		struct{ OrderIndex int }{OrderIndex: 1},
		struct{ OrderIndex int }{OrderIndex: 2},
	)
	err := Validate.Struct(w)
	assert.Error(t, err)
}

func TestValidateRolloutStages_Unordered_ReturnsTrue(t *testing.T) {
	t.Parallel()
	w := rolloutStagesWrapper{}
	w.Stages = append(w.Stages,
		struct{ OrderIndex int }{OrderIndex: 2},
		struct{ OrderIndex int }{OrderIndex: 0},
		struct{ OrderIndex int }{OrderIndex: 1},
	)
	err := Validate.Struct(w)
	assert.NoError(t, err)
}

func TestValidateSemver_Valid(t *testing.T) {
	t.Parallel()
	valid := []string{"1.2.3", "0.0.1", "10.20.30", "1.0.0-alpha", "1.0.0+build.1"}
	for _, v := range valid {
		w := semverWrapper{V: v}
		require.NoErrorf(t, Validate.Struct(w), "expected %q to be valid", v)
	}
}

func TestValidateSemver_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []string{"1.2", "v1.2.3", "1.2.3.4", "", "abc", "1.2.3-"}
	for _, v := range invalid {
		w := semverWrapper{V: v}
		require.Errorf(t, Validate.Struct(w), "expected %q to be invalid", v)
	}
}

func TestValidateUpdateAttemptResult_Valid(t *testing.T) {
	t.Parallel()
	valid := []domain.UpdateAttemptsResult{
		domain.UpdateAttemptsResultSuccess,
		domain.UpdateAttemptsResultFailure,
		domain.UpdateAttemptsResultTimeout,
	}
	for _, r := range valid {
		w := updateAttemptResultWrapper{R: r}
		require.NoErrorf(t, Validate.Struct(w), "expected %q to be valid", r)
	}
}

func TestValidateUpdateAttemptResult_Invalid(t *testing.T) {
	t.Parallel()
	invalid := []domain.UpdateAttemptsResult{"", "Success", "unknown", "timeout "}
	for _, r := range invalid {
		w := updateAttemptResultWrapper{R: r}
		require.Errorf(t, Validate.Struct(w), "expected %q to be invalid", r)
	}
}
