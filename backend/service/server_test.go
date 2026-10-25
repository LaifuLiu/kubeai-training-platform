package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func newDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			volcanoJobGVR: "JobList",
		},
		objects...,
	)
}

func volcanoJob(name, phase string, labels map[string]string) *unstructured.Unstructured {
	labelMap := map[string]interface{}{}
	for k, v := range labels {
		labelMap[k] = v
	}

	obj := map[string]interface{}{
		"apiVersion": "batch.volcano.sh/v1alpha1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": metav1.Now().Format(time.RFC3339),
		},
	}
	if len(labelMap) > 0 {
		obj["metadata"].(map[string]interface{})["labels"] = labelMap
	}
	if phase != "" {
		obj["status"] = map[string]interface{}{
			"state": map[string]interface{}{
				"phase": phase,
			},
		}
	}

	return &unstructured.Unstructured{Object: obj}
}

func trainingPod(name, jobName string, phase corev1.PodPhase, created time.Time) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.NewTime(created),
			Labels: map[string]string{
				"volcano.sh/job-name": jobName,
			},
		},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func TestVolcanoJobStatus(t *testing.T) {
	cases := []struct {
		name string
		job  *unstructured.Unstructured
		want string
	}{
		{
			name: "missing status",
			job:  &unstructured.Unstructured{Object: map[string]interface{}{}},
			want: "Pending",
		},
		{
			name: "pending",
			job:  volcanoJob("a", "Pending", nil),
			want: "Pending",
		},
		{
			name: "running",
			job:  volcanoJob("a", "Running", nil),
			want: "Running",
		},
		{
			name: "completed maps to succeeded",
			job:  volcanoJob("a", "Completed", nil),
			want: "Succeeded",
		},
		{
			name: "failed",
			job:  volcanoJob("a", "Failed", nil),
			want: "Failed",
		},
		{
			name: "unknown phase",
			job:  volcanoJob("a", "Aborting", nil),
			want: "Pending",
		},
		{
			name: "non-string phase",
			job: &unstructured.Unstructured{Object: map[string]interface{}{
				"status": map[string]interface{}{
					"state": map[string]interface{}{
						"phase": 1,
					},
				},
			}},
			want: "Pending",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := volcanoJobStatus(tc.job); got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInt64Ptr(t *testing.T) {
	got := int64Ptr(500)
	if got == nil || *got != 500 {
		t.Fatalf("int64Ptr(500) = %v", got)
	}
}

func TestCreateTrainingRunValidation(t *testing.T) {
	s := &Server{}

	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"invalid json", "{", http.StatusBadRequest, "invalid request body"},
		{"missing name", `{"epochs":1}`, http.StatusBadRequest, "name is required"},
		{"epochs zero", `{"name":"r","epochs":0}`, http.StatusBadRequest, "epochs must be greater than 0"},
		{"replicas too high", `{"name":"r","epochs":1,"replicas":9}`, http.StatusBadRequest, "replicas must be between 1 and 8"},
		{"replicas negative", `{"name":"r","epochs":1,"replicas":-1}`, http.StatusBadRequest, "replicas must be between 1 and 8"},
		{"bad priority", `{"name":"r","epochs":1,"priority":"low"}`, http.StatusBadRequest, "priority must be normal or high"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/training-runs", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			s.CreateTrainingRun(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %q, want substring %q", rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestCreateTrainingRunSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	dyn := newDynamicClient()
	s := &Server{dynamicClient: dyn, db: db}

	mock.ExpectExec("INSERT INTO training_runs").
		WithArgs("mnist", namespace, 4, 2, "1", "2Gi", "high", TrainingQueue, "Pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"name":"mnist","epochs":4,"cpu":"1","memory":"2Gi","priority":"high","replicas":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/training-runs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.CreateTrainingRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp CreateTrainingRunResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "mnist" || resp.Namespace != namespace || resp.Status != "submitted" {
		t.Fatalf("response = %+v", resp)
	}

	job, err := dyn.Resource(volcanoJobGVR).Namespace(namespace).Get(context.Background(), "mnist", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get created job: %v", err)
	}

	tasks, found, err := unstructured.NestedSlice(job.Object, "spec", "tasks")
	if err != nil || !found || len(tasks) != 1 {
		t.Fatalf("tasks missing: found=%v err=%v", found, err)
	}
	task := tasks[0].(map[string]interface{})
	if task["replicas"].(int64) != 2 {
		t.Fatalf("replicas = %v", task["replicas"])
	}
	template := task["template"].(map[string]interface{})
	spec := template["spec"].(map[string]interface{})
	if spec["priorityClassName"] != "kubeai-high" {
		t.Fatalf("priorityClassName = %v", spec["priorityClassName"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateTrainingRunAppliesDefaults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	dyn := newDynamicClient()
	s := &Server{dynamicClient: dyn, db: db}

	mock.ExpectExec("INSERT INTO training_runs").
		WithArgs("defaults", namespace, 1, 1, "500m", "1Gi", "normal", TrainingQueue, "Pending").
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/training-runs", strings.NewReader(`{"name":"defaults","epochs":1}`))
	rec := httptest.NewRecorder()
	s.CreateTrainingRun(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTrainingRunKubernetesError(t *testing.T) {
	dyn := newDynamicClient()
	dyn.PrependReactor("create", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("quota exceeded")
	})
	s := &Server{dynamicClient: dyn}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/training-runs", strings.NewReader(`{"name":"r","epochs":1}`))
	rec := httptest.NewRecorder()
	s.CreateTrainingRun(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "quota exceeded") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestCreateTrainingRunDatabaseFailureCleansUpJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	dyn := newDynamicClient()
	s := &Server{dynamicClient: dyn, db: db}

	mock.ExpectExec("INSERT INTO training_runs").WillReturnError(errors.New("unique violation"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/training-runs", strings.NewReader(`{"name":"cleanup","epochs":1}`))
	rec := httptest.NewRecorder()
	s.CreateTrainingRun(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "database record could not be saved") {
		t.Fatalf("body = %q", rec.Body.String())
	}

	_, err = dyn.Resource(volcanoJobGVR).Namespace(namespace).Get(context.Background(), "cleanup", metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected job cleanup, get err = %v", err)
	}
}

func TestListTrainingRunsQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT name, namespace").WillReturnError(errors.New("db down"))
	s := &Server{db: db}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListTrainingRunsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	})
	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)

	s := &Server{db: db, dynamicClient: newDynamicClient(), client: fake.NewSimpleClientset()}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var runs []TrainingRunSummary
	if err := json.NewDecoder(rec.Body).Decode(&runs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("runs = %+v", runs)
	}
}

func TestListTrainingRunsSyncsLiveJobAndSkipsCancelled(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("live", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Pending", created).
		AddRow("gone", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Succeeded", created).
		AddRow("stopped", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Cancelled", created)

	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)
	mock.ExpectExec("UPDATE training_runs").
		WithArgs("live", "Running").
		WillReturnResult(sqlmock.NewResult(0, 1))

	dyn := newDynamicClient(
		volcanoJob("live", "Running", nil),
		volcanoJob("stopped", "Running", nil),
	)
	cs := fake.NewSimpleClientset(
		trainingPod("live-0", "live", corev1.PodRunning, created),
		trainingPod("stopped-0", "stopped", corev1.PodSucceeded, created),
	)
	s := &Server{db: db, dynamicClient: dyn, client: cs}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var runs []TrainingRunSummary
	if err := json.NewDecoder(rec.Body).Decode(&runs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("len = %d", len(runs))
	}
	if runs[0].Status != "Running" || runs[0].Active != 1 {
		t.Fatalf("live run = %+v", runs[0])
	}
	if runs[1].Status != "Succeeded" {
		t.Fatalf("missing job should keep db status, got %+v", runs[1])
	}
	if runs[2].Status != "Running" {
		t.Fatalf("cancelled run still reports live volcano status, got %+v", runs[2])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListTrainingRunsVolcanoError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("live", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Pending", time.Now())
	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)

	dyn := newDynamicClient()
	dyn.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("apiserver unavailable")
	})
	s := &Server{db: db, dynamicClient: dyn}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListTrainingRunsStatusSyncError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("live", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Pending", time.Now())
	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)
	mock.ExpectExec("UPDATE training_runs").WillReturnError(errors.New("write failed"))

	s := &Server{
		db:            db,
		dynamicClient: newDynamicClient(volcanoJob("live", "Running", nil)),
		client:        fake.NewSimpleClientset(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestListTrainingRunsPodCounterError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"name", "namespace", "epochs", "replicas", "cpu", "memory", "priority", "queue", "status", "created_at",
	}).AddRow("live", namespace, 3, 1, "500m", "1Gi", "normal", TrainingQueue, "Pending", time.Now())
	mock.ExpectQuery("SELECT name, namespace").WillReturnRows(rows)
	mock.ExpectExec("UPDATE training_runs").
		WithArgs("live", "Running").
		WillReturnResult(sqlmock.NewResult(0, 1))

	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("list pods failed")
	})
	s := &Server{
		db:            db,
		dynamicClient: newDynamicClient(volcanoJob("live", "Running", nil)),
		client:        cs,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs", nil)
	rec := httptest.NewRecorder()
	s.ListTrainingRuns(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRun(t *testing.T) {
	s := &Server{dynamicClient: newDynamicClient(volcanoJob("demo", "Completed", nil))}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	s.GetTrainingRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var summary TrainingRunSummary
	if err := json.NewDecoder(rec.Body).Decode(&summary); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if summary.Name != "demo" || summary.Status != "Succeeded" {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestGetTrainingRunMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs/", nil)
	rec := httptest.NewRecorder()
	(&Server{}).GetTrainingRun(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs/missing", nil)
	req.SetPathValue("name", "missing")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: newDynamicClient()}).GetTrainingRun(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunAPIError(t *testing.T) {
	dyn := newDynamicClient()
	dyn.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("timeout")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: dyn}).GetTrainingRun(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunLogsMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	rec := httptest.NewRecorder()
	(&Server{}).GetTrainingRunLogs(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunLogsJobNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "missing")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: newDynamicClient()}).GetTrainingRunLogs(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunLogsJobError(t *testing.T) {
	dyn := newDynamicClient()
	dyn.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("boom")
	})
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: dyn}).GetTrainingRunLogs(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunLogsPodListError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("cannot list")
	})
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{
		dynamicClient: newDynamicClient(volcanoJob("demo", "Running", nil)),
		client:        cs,
	}).GetTrainingRunLogs(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetTrainingRunLogsPodNotReady(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{
		dynamicClient: newDynamicClient(volcanoJob("demo", "Pending", nil)),
		client:        fake.NewSimpleClientset(),
	}).GetTrainingRunLogs(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestGetTrainingRunLogsSuccessPrefersNewestPod(t *testing.T) {
	now := time.Now()
	older := trainingPod("old", "demo", corev1.PodSucceeded, now.Add(-time.Hour))
	newer := trainingPod("new", "demo", corev1.PodRunning, now)
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{
		dynamicClient: newDynamicClient(volcanoJob("demo", "Running", nil)),
		client:        fake.NewSimpleClientset(older, newer),
	}).GetTrainingRunLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected log body")
	}
}

func TestGetTrainingRunLogsLegacySelector(t *testing.T) {
	legacy := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "legacy-pod",
			Namespace: namespace,
			Labels:    map[string]string{"job-name": "demo"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{
		dynamicClient: newDynamicClient(volcanoJob("demo", "Running", nil)),
		client:        fake.NewSimpleClientset(legacy),
	}).GetTrainingRunLogs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteTrainingRunMissingName(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/", nil)
	rec := httptest.NewRecorder()
	(&Server{}).DeleteTrainingRun(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteTrainingRunNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/missing", nil)
	req.SetPathValue("name", "missing")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: newDynamicClient()}).DeleteTrainingRun(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteTrainingRunForbiddenWhenUnmanaged(t *testing.T) {
	job := volcanoJob("foreign", "Running", map[string]string{"kubeai.io/managed-by": "other"})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/foreign", nil)
	req.SetPathValue("name", "foreign")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: newDynamicClient(job)}).DeleteTrainingRun(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteTrainingRunSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE training_runs").
		WithArgs("demo", "Cancelled").
		WillReturnResult(sqlmock.NewResult(0, 1))

	job := volcanoJob("demo", "Running", map[string]string{"kubeai.io/managed-by": "kubeai"})
	s := &Server{dynamicClient: newDynamicClient(job), db: db}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	s.DeleteTrainingRun(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["status"] != "deletion_requested" || payload["name"] != "demo" {
		t.Fatalf("payload = %+v", payload)
	}

	_, err = s.dynamicClient.Resource(volcanoJobGVR).Namespace(namespace).Get(context.Background(), "demo", metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected job deleted, err = %v", err)
	}
}

func TestDeleteTrainingRunDatabaseUpdateFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("UPDATE training_runs").
		WithArgs("demo", "Cancelled").
		WillReturnError(errors.New("db write failed"))

	job := volcanoJob("demo", "Running", map[string]string{"kubeai.io/managed-by": "kubeai"})
	s := &Server{dynamicClient: newDynamicClient(job), db: db}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	s.DeleteTrainingRun(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteTrainingRunGetError(t *testing.T) {
	dyn := newDynamicClient()
	dyn.PrependReactor("get", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("get failed")
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: dyn}).DeleteTrainingRun(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestDeleteTrainingRunDeleteError(t *testing.T) {
	dyn := newDynamicClient(volcanoJob("demo", "Running", map[string]string{"kubeai.io/managed-by": "kubeai"}))
	dyn.PrependReactor("delete", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("delete failed")
	})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/training-runs/demo", nil)
	req.SetPathValue("name", "demo")
	rec := httptest.NewRecorder()
	(&Server{dynamicClient: dyn}).DeleteTrainingRun(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestGetPodCounters(t *testing.T) {
	now := time.Now()
	cs := fake.NewSimpleClientset(
		trainingPod("a", "job", corev1.PodPending, now),
		trainingPod("b", "job", corev1.PodRunning, now),
		trainingPod("c", "job", corev1.PodSucceeded, now),
		trainingPod("d", "job", corev1.PodFailed, now),
		trainingPod("e", "job", corev1.PodUnknown, now),
		trainingPod("other", "other-job", corev1.PodRunning, now),
	)
	s := &Server{client: cs}

	got, err := s.getPodCounters(context.Background(), "job")
	if err != nil {
		t.Fatalf("getPodCounters: %v", err)
	}
	if got.Active != 2 || got.Succeeded != 1 || got.Failed != 1 {
		t.Fatalf("counters = %+v", got)
	}
}

func TestGetPodCountersListError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("no")
	})
	_, err := (&Server{client: cs}).getPodCounters(context.Background(), "job")
	if err == nil {
		t.Fatal("expected error")
	}
}
