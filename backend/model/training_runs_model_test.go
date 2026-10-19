package model

import (
	"strings"
	"testing"
)

func TestSQLConstantsIncludeExpectedStatements(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"create table", CreateTrainingRunsTableSQL, "CREATE TABLE IF NOT EXISTS training_runs"},
		{"insert", CreateTrainingRunRecordSQL, "INSERT INTO training_runs"},
		{"list", GetAllTrainingRunRecordsSQL, "ORDER BY created_at DESC"},
		{"update", UpdateTrainingRunStatusSQL, "UPDATE training_runs"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(tc.sql, tc.want) {
				t.Fatalf("SQL does not contain %q", tc.want)
			}
		})
	}
}
