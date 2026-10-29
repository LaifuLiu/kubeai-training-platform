package model

const CreateTrainingRunsTableSQL = `
CREATE TABLE IF NOT EXISTS training_runs (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    namespace TEXT NOT NULL DEFAULT 'kubeai-training',
    epochs INTEGER NOT NULL,
    replicas INTEGER NOT NULL DEFAULT 1,
		min_available INTEGER NOT NULL DEFAULT 1,
    cpu TEXT NOT NULL,
		gpu INTEGER NOT NULL DEFAULT 0,
    memory TEXT NOT NULL,
    priority TEXT NOT NULL,
    queue TEXT NOT NULL DEFAULT 'ai-training',
    status TEXT NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		scheduled_at TIMESTAMPTZ,
		started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
`

const CreateTrainingRunRecordSQL = `
INSERT INTO training_runs (
			name, namespace, epochs, replicas, min_available, cpu, gpu, memory,
			priority, queue, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
`

const GetAllTrainingRunRecordsSQL = `
SELECT
    name,
    namespace,
    epochs,
    replicas,
    min_available,
    cpu,
    gpu,
    memory,
    priority,
    queue,
    status,
    created_at,
    updated_at,
    scheduled_at,
    started_at
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
const UpdateTrainingRunScheduledAtSQL = `
		UPDATE training_runs
		SET scheduled_at = COALESCE(scheduled_at, $2),
		    updated_at = NOW()
		WHERE name = $1
	`

const UpdateTrainingRunStartedAtSQL = ` 
		UPDATE training_runs
		SET started_at = COALESCE(started_at, $2),
		    updated_at = NOW()
		WHERE name = $1
	`
