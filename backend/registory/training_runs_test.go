package registory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestInitializeTrainingRunsSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS training_runs").
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := InitializeTrainingRunsSchema(context.Background(), db); err != nil {
		t.Fatalf("InitializeTrainingRunsSchema: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestInitializeTrainingRunsSchemaError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS training_runs").
		WillReturnError(errors.New("permission denied"))

	err = InitializeTrainingRunsSchema(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "create training_runs table") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateTrainingRunRecord(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	run := TrainingRun{
		Name:      "demo",
		Namespace: "kubeai-training",
		Epochs:    3,
		Replicas:  1,
		CPU:       "500m",
		Memory:    "1Gi",
		Priority:  "normal",
		Queue:     "ai-training",
		Status:    "Pending",
	}

	mock.ExpectExec("INSERT INTO training_runs").
		WithArgs(run.Name, run.Namespace, run.Epochs, run.Replicas, run.CPU, run.Memory, run.Priority, run.Queue, run.Status).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := CreateTrainingRunRecord(context.Background(), db, run); err != nil {
		t.Fatalf("CreateTrainingRunRecord: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateTrainingRunRecordError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO training_runs").
		WillReturnError(errors.New("duplicate key"))

	err = CreateTrainingRunRecord(context.Background(), db, TrainingRun{Name: "demo"})
	if err == nil || !strings.Contains(err.Error(), "create training run record") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetAllTrainingRunRecords(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	created := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("demo", "kubeai-training", 5, 2, "1", "2Gi", "high", "ai-training", "Running", created)

	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)

	got, err := GetAllTrainingRunRecords(context.Background(), db)
	if err != nil {
		t.Fatalf("GetAllTrainingRunRecords: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "demo" || got[0].Replicas != 2 || !got[0].CreatedAt.Equal(created) {
		t.Fatalf("record = %+v", got[0])
	}
}

func TestGetAllTrainingRunRecordsQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT name, namespace").WillReturnError(errors.New("connection reset"))

	_, err = GetAllTrainingRunRecords(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "query training run records") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetAllTrainingRunRecordsScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("demo", "kubeai-training", "not-an-int", 1, "500m", "1Gi", "normal", "ai-training", "Pending", time.Now())

	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)

	_, err = GetAllTrainingRunRecords(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "scan training run record") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateTrainingRunStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE training_runs").
		WithArgs("demo", "Succeeded").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := UpdateTrainingRunStatus(context.Background(), db, "demo", "Succeeded"); err != nil {
		t.Fatalf("UpdateTrainingRunStatus: %v", err)
	}
}

func TestUpdateTrainingRunStatusNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE training_runs").
		WithArgs("missing", "Failed").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = UpdateTrainingRunStatus(context.Background(), db, "missing", "Failed")
	if err == nil || !strings.Contains(err.Error(), `training run "missing" not found`) {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateTrainingRunStatusExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE training_runs").
		WillReturnError(errors.New("deadlock"))

	err = UpdateTrainingRunStatus(context.Background(), db, "demo", "Failed")
	if err == nil || !strings.Contains(err.Error(), "update training run status") {
		t.Fatalf("error = %v", err)
	}
}
