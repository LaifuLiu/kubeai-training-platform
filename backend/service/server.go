package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/LaifuLiu/kubeai-training-platform/backend/registory"
)

type Server struct {
	client        kubernetes.Interface
	dynamicClient dynamic.Interface
	db            *sql.DB
}

type CreateTrainingRunRequest struct {
	Name     string `json:"name"`
	Epochs   int    `json:"epochs"`
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
	Priority string `json:"priority"`
}

type CreateTrainingRunResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

type TrainingRunSummary struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Status    string    `json:"status"`
	Epochs    string    `json:"epochs,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	Active    int32     `json:"active"`
	Succeeded int32     `json:"succeeded"`
	Failed    int32     `json:"failed"`
}

type PodCounters struct {
	Active    int32 `json:"active"`
	Succeeded int32 `json:"succeeded"`
	Failed    int32 `json:"failed"`
}

var volcanoJobGVR = schema.GroupVersionResource{
	Group:    "batch.volcano.sh",
	Version:  "v1alpha1",
	Resource: "jobs",
}

const trainingQueue = "ai-training"
const namespace = "kubeai-training"

func newClients() (kubernetes.Interface, dynamic.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(
		"",
		clientcmd.RecommendedHomeFile,
	)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, dynamicClient, nil
}

func NewServer(db *sql.DB) *Server {
	clientset, dynamicClient, err := newClients()
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		client:        clientset,
		dynamicClient: dynamicClient,
		db:            db,
	}

}

func (s *Server) CreateTrainingRun(w http.ResponseWriter, r *http.Request) {
	var req CreateTrainingRunRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Epochs <= 0 {
		http.Error(w, "epochs must be greater than 0", http.StatusBadRequest)
		return
	}

	if req.CPU == "" {
		req.CPU = "500m"
	}

	if req.Memory == "" {
		req.Memory = "1Gi"
	}

	if req.Priority == "" {
		req.Priority = "normal"
	}

	var priorityClassName string

	switch req.Priority {
	case "normal":
		priorityClassName = "kubeai-normal"
	case "high":
		priorityClassName = "kubeai-high"
	default:
		http.Error(w, "priority must be normal or high", http.StatusBadRequest)
		return
	}

	job := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "batch.volcano.sh/v1alpha1",
			"kind":       "Job",
			"metadata": map[string]interface{}{
				"name":      req.Name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"kubeai.io/managed-by": "kubeai",
					"kubeai.io/component":  "training",
				},
			},
			"spec": map[string]interface{}{
				"schedulerName": "volcano",
				"queue":         trainingQueue,
				"minAvailable":  int64(1),
				"maxRetry":      int64(0),
				"tasks": []interface{}{
					map[string]interface{}{
						"name":     "trainer",
						"replicas": int64(1),
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"kubeai.io/managed-by": "kubeai",
									"kubeai.io/component":  "training",
								},
							},
							"spec": map[string]interface{}{
								"restartPolicy":     "Never",
								"priorityClassName": priorityClassName,
								"containers": []interface{}{
									map[string]interface{}{
										"name":            "trainer",
										"image":           "192.168.122.1:5000/kubeai-trainer:v0.1.0",
										"imagePullPolicy": "IfNotPresent",
										"env": []interface{}{
											map[string]interface{}{
												"name":  "EPOCHS",
												"value": strconv.Itoa(req.Epochs),
											},
											map[string]interface{}{
												"name":  "SEED",
												"value": "42",
											},
										},
										"resources": map[string]interface{}{
											"requests": map[string]interface{}{
												"cpu":    req.CPU,
												"memory": req.Memory,
											},
											"limits": map[string]interface{}{
												"cpu":    req.CPU,
												"memory": req.Memory,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	created, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace("kubeai-training").
		Create(ctx, job, metav1.CreateOptions{})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	run := registory.TrainingRun{
		Name:      created.GetName(),
		Namespace: namespace,
		Epochs:    req.Epochs,
		CPU:       req.CPU,
		Memory:    req.Memory,
		Priority:  req.Priority,
		Queue:     trainingQueue,
		Status:    "Pending",
	}

	if err := registory.CreateTrainingRunRecord(ctx, s.db, run); err != nil {
		// Best-effort cleanup so we don't leave an untracked Volcano Job.
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cleanupCancel()

		cleanupErr := s.dynamicClient.
			Resource(volcanoJobGVR).
			Namespace(namespace).
			Delete(cleanupCtx, created.GetName(), metav1.DeleteOptions{})

		if cleanupErr != nil {
			log.Printf(
				"database insert failed for training run %q; cleanup failed: %v",
				created.GetName(),
				cleanupErr,
			)
		}

		log.Printf(
			"database insert failed for training run %q: %v",
			created.GetName(),
			err,
		)

		http.Error(
			w,
			"training Job was created but its database record could not be saved",
			http.StatusInternalServerError,
		)
		return
	}

	response := CreateTrainingRunResponse{
		Name:      "" + created.GetName(),
		Namespace: namespace,
		Status:    "submitted",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

func (s *Server) ListTrainingRuns(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	runsRecords, err := registory.GetAllTrainingRunRecords(ctx, s.db)
	if err != nil {
		log.Printf("query training run records: %v", err)
		http.Error(w, "failed to query training run records", http.StatusInternalServerError)
		return
	}

	runs := make([]TrainingRunSummary, 0, len(runsRecords))
	for _, run := range runsRecords {
		summary := TrainingRunSummary{
			Name:      run.Name,
			Namespace: run.Namespace,
			Status:    run.Status,
			CreatedAt: run.CreatedAt,
		}

		// Volcano provides live status while the Job still exists.
		job, err := s.dynamicClient.
			Resource(volcanoJobGVR).
			Namespace(run.Namespace).
			Get(ctx, run.Name, metav1.GetOptions{})

		if err == nil {
			summary.Status = volcanoJobStatus(job)

			if run.Status != "Cancelled" {
				if err := registory.UpdateTrainingRunStatus(
					ctx,
					s.db,
					run.Name,
					summary.Status,
				); err != nil {
					log.Printf("sync status for training run %s: %v", run.Name, err)
					http.Error(w, "failed to synchronize training run status", http.StatusInternalServerError)
					return
				}
			}

			counters, err := s.getPodCounters(ctx, run.Name)
			if err != nil {
				http.Error(w, "failed to query training Pod status", http.StatusInternalServerError)
				log.Printf("get Pod counters for %s: %v", run.Name, err)
				return
			}

			summary.Active = counters.Active
			summary.Succeeded = counters.Succeeded
			summary.Failed = counters.Failed
		} else if apierrors.IsNotFound(err) {
			// No Volcano Job: retain the last status recorded in PostgreSQL.
			// We will improve lifecycle status synchronization in a later step.
		} else {
			http.Error(w, "failed to query Volcano Job", http.StatusInternalServerError)
			log.Printf("get Volcano Job %s: %v", run.Name, err)
			return
		}

		runs = append(runs, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runs); err != nil {
		log.Printf("encode training runs: %v", err)
	}
}

func (s *Server) GetTrainingRun(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "training run name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	job, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace("kubeai-training").
		Get(
			ctx,
			name,
			metav1.GetOptions{},
		)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "training run not found", http.StatusNotFound)
			return
		}

		log.Printf("get training run %q: %v", name, err)
		http.Error(w, "failed to get training run", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	summary := TrainingRunSummary{
		Name:      job.GetName(),
		Namespace: job.GetNamespace(),
		Status:    volcanoJobStatus(job),
		CreatedAt: job.GetCreationTimestamp().Time,
	}
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		log.Printf("encode training run: %v", err)
	}
}

func (s *Server) GetTrainingRunLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "training run name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	namespace := "kubeai-training"

	// Verify that the Job exists before retrieving its logs.
	_, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace("kubeai-training").
		Get(
			ctx,
			name,
			metav1.GetOptions{},
		)
	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "training run not found", http.StatusNotFound)
			return
		}

		log.Printf("get job %q: %v", name, err)
		http.Error(w, "failed to retrieve training run", http.StatusInternalServerError)
		return
	}

	// Find the Pod created for this Job.
	selector := fmt.Sprintf(
		"volcano.sh/job-name=%s",
		name,
	)

	pods, err := s.client.CoreV1().
		Pods(namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: selector,
		})
	if err != nil {
		log.Printf("list pods for job %q: %v", name, err)
		http.Error(w, "failed to find training pod", http.StatusInternalServerError)
		return
	}

	// Support the legacy Job Pod label as a fallback.
	if len(pods.Items) == 0 {
		pods, err = s.client.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: "job-name=" + name,
			})
		if err != nil {
			log.Printf("list legacy pods for job %q: %v", name, err)
			http.Error(w, "failed to find training pod", http.StatusInternalServerError)
			return
		}
	}

	if len(pods.Items) == 0 {
		http.Error(w, "training pod is not available yet", http.StatusNotFound)
		return
	}

	// Prefer the most recently created Pod.
	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.After(
			pods.Items[j].CreationTimestamp.Time,
		)
	})

	pod := pods.Items[0]

	req := s.client.CoreV1().
		Pods(namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: "trainer",
			TailLines: int64Ptr(500),
		})

	stream, err := req.Stream(ctx)
	if err != nil {
		log.Printf("open logs for pod %q: %v", pod.Name, err)
		http.Error(w, "failed to retrieve training logs", http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	var output strings.Builder
	if _, err := io.Copy(&output, stream); err != nil {
		log.Printf("read logs for pod %q: %v", pod.Name, err)
		http.Error(w, "failed to read training logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(output.String()))
}

func (s *Server) DeleteTrainingRun(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "training run name is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	namespace := "kubeai-training"

	job, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "training run not found", http.StatusNotFound)
			return
		}

		log.Printf("get training run %q: %v", name, err)
		http.Error(w, "failed to retrieve training run", http.StatusInternalServerError)
		return
	}

	// Only allow this API to delete Jobs managed by KubeAI.
	if job.GetLabels()["kubeai.io/managed-by"] != "kubeai" {
		http.Error(w, "training run is not managed by KubeAI", http.StatusForbidden)
		return
	}

	propagation := metav1.DeletePropagationForeground

	err = s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(namespace).
		Delete(ctx, name, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		})

	if err != nil {
		if apierrors.IsNotFound(err) {
			http.Error(w, "training run not found", http.StatusNotFound)
			return
		}

		log.Printf("delete training run %q: %v", name, err)
		http.Error(w, "failed to delete training run", http.StatusInternalServerError)
		return
	}

	if err := registory.UpdateTrainingRunStatus(ctx, s.db, name, "Cancelled"); err != nil {
		log.Printf("sync cancelled status for training run %q: %v", name, err)
		http.Error(w, "deletion was requested but database status update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"name":      name,
		"namespace": namespace,
		"status":    "deletion_requested",
	}); err != nil {
		log.Printf("encode delete response for %q: %v", name, err)
	}
}

func (s *Server) getPodCounters(ctx context.Context, jobName string) (PodCounters, error) {
	pods, err := s.client.CoreV1().
		Pods("kubeai-training").
		List(
			ctx,
			metav1.ListOptions{
				LabelSelector: fmt.Sprintf(
					"volcano.sh/job-name=%s",
					jobName,
				),
			},
		)
	if err != nil {
		return PodCounters{}, err
	}

	var counters PodCounters

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodPending, corev1.PodRunning:
			counters.Active++

		case corev1.PodSucceeded:
			counters.Succeeded++

		case corev1.PodFailed:
			counters.Failed++
		}
	}

	return counters, nil
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}

func jobStatus(job *batchv1.Job) string {
	for _, condition := range job.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			continue
		}

		switch condition.Type {
		case batchv1.JobComplete:
			return "Completed"
		case batchv1.JobFailed:
			return "Failed"
		}
	}

	if job.Status.Active > 0 {
		return "Running"
	}

	return "Pending"
}

func toSummary(job *batchv1.Job) TrainingRunSummary {
	summary := TrainingRunSummary{
		Name:      job.Name,
		Namespace: job.Namespace,
		Status:    jobStatus(job),
		CreatedAt: job.CreationTimestamp.Time,
		Active:    job.Status.Active,
		Succeeded: job.Status.Succeeded,
		Failed:    job.Status.Failed,
	}

	for _, container := range job.Spec.Template.Spec.Containers {
		if container.Name != "trainer" {
			continue
		}

		for _, env := range container.Env {
			if env.Name == "EPOCHS" {
				summary.Epochs = env.Value
			}
		}
	}

	return summary
}

func volcanoJobStatus(job *unstructured.Unstructured) string {
	phase, found, err := unstructured.NestedString(
		job.Object,
		"status",
		"state",
		"phase",
	)

	if err != nil || !found {
		return "Pending"
	}

	switch phase {
	case "Pending":
		return "Pending"
	case "Running":
		return "Running"
	case "Completed":
		return "Succeeded"
	case "Failed":
		return "Failed"
	default:
		return "Pending"
	}
}
