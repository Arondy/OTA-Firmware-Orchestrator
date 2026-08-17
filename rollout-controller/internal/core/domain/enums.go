package domain

type UpdateAttemptsResult string

const (
	UpdateAttemptsResultSuccess UpdateAttemptsResult = "success"
	UpdateAttemptsResultFailure UpdateAttemptsResult = "failure"
	UpdateAttemptsResultTimeout UpdateAttemptsResult = "timeout"
)
