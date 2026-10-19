package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOpenDatabaseRequiresPassword(t *testing.T) {
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "kubeai")
	t.Setenv("DB_NAME", "kubeai")

	_, err := OpenDatabase(context.Background())
	if err == nil {
		t.Fatal("expected error when DB_PASSWORD is empty")
	}
	if !strings.Contains(err.Error(), "DB_PASSWORD must be set") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenDatabasePingFailure(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "1")
	t.Setenv("DB_USER", "kubeai")
	t.Setenv("DB_NAME", "kubeai")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := OpenDatabase(ctx)
	if err == nil {
		t.Fatal("expected ping failure against closed port")
	}
	if !strings.Contains(err.Error(), "ping database") {
		t.Fatalf("error = %v", err)
	}
}
