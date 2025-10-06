package domain

import "time"

type Status string

const (
	Queued       Status = "queued"
	Leased       Status = "leased"
	Succeeded    Status = "succeeded"
	FailedTemp   Status = "failed_temp"
	FailedPerm   Status = "failed_perm"
	DeadLettered Status = "dead_lettered"
)

type Job struct {
	ID                   string
	TenantID             string
	Type                 string
	Payload              []byte
	Priority             int
	RunAt                time.Time
	DedupeKey            *string
	DedupeTTL            *int
	Attempt              int
	MaxAttempts          int
	BackoffPolicy        string
	VisibilityTimeoutSec int
	Status               Status
	LeasedBy             *string
	LeaseExpiresAt       *time.Time
	Error                *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
