package domain

import "errors"

var ErrUpdateResultNotProduced = errors.New("failed to produce update result")
var ErrCurrentStageNotFound = errors.New("current stage for this campaign wasn't found")
