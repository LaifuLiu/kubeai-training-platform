package registory

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/LaifuLiu/kubeai-training-platform/backend/model"
)

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

func InitializeTrainingRunsSchema(
	ctx context.Context,
	db *sql.DB,
) error {
	if _, err := db.ExecContext(ctx, model.CreateTrainingRunsTableSQL); err != nil {
		return fmt.Errorf("create training_runs table: %w", err)
	}

	return nil
}

func CreateTrainingRunRecord(ctx context.Context,
	db *sql.DB,
	run TrainingRun) error {
	_, err := db.ExecContext(ctx, model.CreateTrainingRunRecordSQL,
		run.Name,
		run.Namespace,
		run.Epochs,
		run.CPU,
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

func GetAllTrainingRunRecords(ctx context.Context,
	db *sql.DB) ([]TrainingRun, error) {
	rows, err := db.QueryContext(ctx, model.GetAllTrainingRunRecordsSQL)
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
			&run.CPU,
			&run.Memory,
			&run.Priority,
			&run.Queue,
			&run.Status,
			&run.CreatedAt,
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

func UpdateTrainingRunStatus(ctx context.Context,
	db *sql.DB,
	name string,
	status string) error {
	result, err := db.ExecContext(ctx, model.UpdateTrainingRunStatusSQL, name, status)
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
