package registory

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/LaifuLiu/kubeai-training-platform/backend/model"
)

func InitializeTrainingRunsSchema(
	ctx context.Context,
	db *sql.DB,
) error {
	if _, err := db.ExecContext(ctx, model.CreateTrainingRunsTableSQL); err != nil {
		return fmt.Errorf("create training_runs table: %w", err)
	}

	return nil
}
