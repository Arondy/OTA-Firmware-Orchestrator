package domain

import "errors"

var ErrUpdateResultNotProduced = errors.New("failed to produce update result")
var ErrCurrentStageNotFound = errors.New("current stage for this campaign wasn't found")
var ErrNotEnoughSamples = errors.New("current stage hasn't reached minimum sample size for evaluation")
