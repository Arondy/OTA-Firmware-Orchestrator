package domain

type UpdateAttemptsResult string

const (
	UpdateAttemptsResultSuccess UpdateAttemptsResult = "success"
	UpdateAttemptsResultFailure UpdateAttemptsResult = "failure"
	UpdateAttemptsResultTimeout UpdateAttemptsResult = "timeout"
)

type DecisionType string

const (
	DecisionTypeAdvance  DecisionType = "advance_stage"
	DecisionTypeRollback DecisionType = "rollback"
)
