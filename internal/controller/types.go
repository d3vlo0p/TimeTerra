package controller

type Activable interface {
	IsActive() bool
}

type JobResult string

const (
	JobResultSuccess JobResult = "Success"
	JobResultFailure JobResult = "Failure"
	JobResultError   JobResult = "Error"
	JobResultSkipped JobResult = "Skipped"
)

func (jr JobResult) String() string {
	return string(jr)
}

type JobMetadata map[string]any
