package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Server struct {
	client kubernetes.Interface
}

type CreateTrainingRunRequest struct {
	Name   string `json:"name"`
	Epochs int    `json:"epochs"`
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

func newClient() (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(
		"",
		clientcmd.RecommendedHomeFile,
	)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func NewServer() *Server {
	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		client: client,
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

	const namespace = "kubeai-training"

	backoffLimit := int32(0)
	ttl := int32(3600)
	deadline := int64(600)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
			Labels: map[string]string{
				"kubeai.io/managed-by": "kubeai",
				"kubeai.io/component":  "training",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			ActiveDeadlineSeconds:   &deadline,
			TTLSecondsAfterFinished: &ttl,

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubeai.io/managed-by": "kubeai",
						"kubeai.io/component":  "training",
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,

					Containers: []corev1.Container{
						{
							Name:  "trainer",
							Image: "192.168.122.1:5000/kubeai-trainer:v0.1.0",

							Env: []corev1.EnvVar{
								{
									Name:  "EPOCHS",
									Value: fmt.Sprintf("%d", req.Epochs),
								},
								{
									Name:  "SEED",
									Value: "42",
								},
							},

							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("3Gi"),
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

	created, err := s.client.
		BatchV1().
		Jobs(namespace).
		Create(ctx, job, metav1.CreateOptions{})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := CreateTrainingRunResponse{
		Name:      created.Name,
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

	jobs, err := s.client.BatchV1().
		Jobs("kubeai-training").
		List(ctx, metav1.ListOptions{
			LabelSelector: "kubeai.io/managed-by=kubeai",
		})
	if err != nil {
		http.Error(w, "failed to list training runs", http.StatusInternalServerError)
		log.Printf("list training runs: %v", err)
		return
	}

	runs := make([]TrainingRunSummary, 0, len(jobs.Items))
	for i := range jobs.Items {
		runs = append(runs, toSummary(&jobs.Items[i]))
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

	job, err := s.client.BatchV1().
		Jobs("kubeai-training").
		Get(ctx, name, metav1.GetOptions{})
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
	if err := json.NewEncoder(w).Encode(toSummary(job)); err != nil {
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
	_, err := s.client.BatchV1().
		Jobs(namespace).
		Get(ctx, name, metav1.GetOptions{})
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
	pods, err := s.client.CoreV1().
		Pods(namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: "batch.kubernetes.io/job-name=" + name,
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

	job, err := s.client.BatchV1().
		Jobs(namespace).
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
	if job.Labels["kubeai.io/managed-by"] != "kubeai" {
		http.Error(w, "training run is not managed by KubeAI", http.StatusForbidden)
		return
	}

	propagation := metav1.DeletePropagationForeground

	err = s.client.BatchV1().
		Jobs(namespace).
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
