package model

import "time"

type TrainingRun struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Namespace   string     `json:"namespace"`
	Epochs      int        `json:"epochs"`
	CPU         string     `json:"cpu"`
	Memory      string     `json:"memory"`
	Priority    string     `json:"priority"`
	Queue       string     `json:"queue"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

const CreateTrainingRunsTableSQL = `
CREATE TABLE IF NOT EXISTS training_runs (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    namespace TEXT NOT NULL DEFAULT 'kubeai-training',
    epochs INTEGER NOT NULL,
    cpu TEXT NOT NULL,
    memory TEXT NOT NULL,
    priority TEXT NOT NULL,
    queue TEXT NOT NULL DEFAULT 'ai-training',
    status TEXT NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
`
