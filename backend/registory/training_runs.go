package registory

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/LaifuLiu/kubeai-training-platform/backend/model"
)

type TrainingRunRepo struct {
	db *sql.DB
}

type TrainingRun struct {
	ID           int64      `json:"id"`                    // This field is not used in the API response, but it is used internally for database operations.
	Name         string     `json:"name"`                  // This field is used as the unique identifier for the training run and is used in the API response.
	Namespace    string     `json:"namespace"`             // This field is used to specify the Kubernetes namespace in which the training run is executed and is used in the API response.
	Epochs       int        `json:"epochs"`                // This field is used to specify the number of epochs for the training run and is used in the API response.
	Replicas     int        `json:"replicas"`              // This field is used to specify the number of replicas for the training run and is used in the API response.
	MinAvailable int        `json:"minAvailable"`          // This field is used to specify the minimum number of available replicas for the training run and is used in the API response.
	CPU          string     `json:"cpu"`                   // This field is used to specify the CPU resources for the training run and is used in the API response.
	GPUs         int        `json:"gpus"`                  // This field is used to specify the GPU resources for the training run and is used in the API response.
	Memory       string     `json:"memory"`                // This field is used to specify the memory resources for the training run and is used in the API response.
	Priority     string     `json:"priority"`              // This field is used to specify the priority for the training run and is used in the API response.
	Queue        string     `json:"queue"`                 // This field is used to specify the queue for the training run and is used in the API response.
	Status       string     `json:"status"`                // This field is used to specify the status for the training run and is used in the API response.
	CreatedAt    time.Time  `json:"createdAt"`             // This field is used to specify the creation time for the training run and is used in the API response.
	UpdatedAt    time.Time  `json:"updatedAt"`             // This field is used to specify the last update time for the training run record in the database.
	ScheduledAt  *time.Time `json:"scheduledAt,omitempty"` // This field is used to specify the scheduling time for the training run and is used in the API response.
	StartedAt    *time.Time `json:"startedAt,omitempty"`   // This field is used to specify the start time for the training run and is used in the API response.
	CompletedAt  *time.Time `json:"completedAt,omitempty"` // This field is used to specify the completion time for the training run and is used in the API response.
}

func NewTrainingRunRepo(db *sql.DB) *TrainingRunRepo {
	return &TrainingRunRepo{db: db}
}

func (r *TrainingRunRepo) InitializeTrainingRunsSchema(
	ctx context.Context,
) error {
	if _, err := r.db.ExecContext(ctx, model.CreateTrainingRunsTableSQL); err != nil {
		return fmt.Errorf("create training_runs table: %w", err)
	}
	return nil
}

func (r *TrainingRunRepo) CreateTrainingRunRecord(ctx context.Context,
	run TrainingRun) error {
	_, err := r.db.ExecContext(ctx, model.CreateTrainingRunRecordSQL,
		run.Name,
		run.Namespace,
		run.Epochs,
		run.Replicas,
		run.MinAvailable,
		run.CPU,
		run.GPUs,
		run.Memory,
		run.Priority,
		run.Queue,
		run.Status,
	)
	if err != nil {
		return fmt.Errorf("create training run record: %w", err)
	}

	return nil
}

func (r *TrainingRunRepo) GetAllTrainingRunRecords(ctx context.Context, page int, size int) ([]TrainingRun, error) {
	if page < 1 || size < 1 {
		return nil, fmt.Errorf("invalid pagination parameters: page and size must be positive")
	}

	offset := (page - 1) * size

	rows, err := r.db.QueryContext(ctx, model.GetAllTrainingRunRecordsSQL, size, offset)
	if err != nil {
		return nil, fmt.Errorf("query training run records: %w", err)
	}
	defer rows.Close()

	var runs []TrainingRun
	for rows.Next() {
		var run TrainingRun
		if err := rows.Scan(
			&run.Name,
			&run.Namespace,
			&run.Epochs,
			&run.Replicas,
			&run.MinAvailable,
			&run.CPU,
			&run.GPUs,
			&run.Memory,
			&run.Priority,
			&run.Queue,
			&run.Status,
			&run.CreatedAt,
			&run.UpdatedAt,
			&run.ScheduledAt,
			&run.StartedAt,
		); err != nil {
			return nil, fmt.Errorf("scan training run record: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training run records: %w", err)
	}

	return runs, nil
}

func (r *TrainingRunRepo) CountTrainingRunRecords(
	ctx context.Context,
) (int, error) {
	var total int

	err := r.db.QueryRowContext(
		ctx,
		model.CountTrainingRunRecordsSQL,
	).Scan(&total)

	if err != nil {
		return 0, fmt.Errorf("count training run records: %w", err)
	}

	return total, nil
}

func (r *TrainingRunRepo) UpdateTrainingRunStatus(ctx context.Context,
	name string,
	status string) error {
	result, err := r.db.ExecContext(ctx, model.UpdateTrainingRunStatusSQL, name, status)
	if err != nil {
		return fmt.Errorf("update training run status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check updated training run: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("training run %q not found in database", name)
	}

	log.Printf("training run %q status updated to %q in database", name, status)

	return nil
}

func (r *TrainingRunRepo) UpdateTrainingRunScheduledAt(
	ctx context.Context,
	name string,
	at time.Time,
) error {
	_, err := r.db.ExecContext(ctx, model.UpdateTrainingRunScheduledAtSQL, name, at)

	if err != nil {
		return fmt.Errorf("update training run scheduled_at: %w", err)
	}

	return nil
}

func (r *TrainingRunRepo) UpdateTrainingRunStartedAt(
	ctx context.Context,
	name string,
	at time.Time,
) error {
	_, err := r.db.ExecContext(ctx, model.UpdateTrainingRunStartedAtSQL, name, at)

	if err != nil {
		return fmt.Errorf("update training run started_at: %w", err)
	}

	return nil
}

func (r *TrainingRunRepo) GetTrainingRunsToReconcile(ctx context.Context) ([]TrainingRun, error) {
	rows, err := r.db.QueryContext(ctx, model.GetTrainingRunsToReconcileSQL)
	if err != nil {
		return nil, fmt.Errorf("query training runs to reconcile: %w", err)
	}
	defer rows.Close()

	var runs []TrainingRun
	for rows.Next() {
		var run TrainingRun
		if err := rows.Scan(
			&run.Name,
			&run.Namespace,
			&run.Status,
			&run.CreatedAt,
			&run.ScheduledAt,
			&run.StartedAt,
		); err != nil {
			return nil, fmt.Errorf("scan training run record: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate training run records: %w", err)
	}

	return runs, nil
}
