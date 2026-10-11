package model

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

const CreateTrainingRunRecordSQL = `
INSERT INTO training_runs (
			name, namespace, epochs, cpu, memory,
			priority, queue, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

const GetAllTrainingRunRecordsSQL = `
SELECT name, namespace, epochs, cpu, memory, priority, queue, status, created_at
FROM training_runs
ORDER BY created_at DESC
`

const UpdateTrainingRunStatusSQL = `
		UPDATE training_runs
		SET status = $2,
		    updated_at = NOW(),
		    completed_at = CASE
		        WHEN $2 IN ('Succeeded', 'Failed', 'Cancelled')
		            THEN COALESCE(completed_at, NOW())
		        ELSE NULL
		    END
		WHERE name = $1
	`
