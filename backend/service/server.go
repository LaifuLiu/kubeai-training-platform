package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/LaifuLiu/kubeai-training-platform/backend/registory"
)

type Server struct {
	client        kubernetes.Interface       // for k8s resources like Pods, Services, etc.
	dynamicClient dynamic.Interface          // for volcano Job resources
	repo          *registory.TrainingRunRepo // for database operations related to training runs
}

type CreateTrainingRunRequest struct {
	Name         string `json:"name"`
	Epochs       int    `json:"epochs"`
	CPU          string `json:"cpu"`
	GPUs         int    `json:"gpus"`
	Memory       string `json:"memory"`
	Priority     string `json:"priority"`
	Replicas     int    `json:"replicas"`
	MinAvailable int    `json:"minAvailable"`
	Queue        string `json:"queue"`
}

type CreateTrainingRunResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

type PodResourceRequests struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	GPUs   string `json:"gpus,omitempty"`
}

type TrainingPodSummary struct {
	Name           string               `json:"name"`
	Phase          string               `json:"phase"`
	NodeName       string               `json:"nodeName,omitempty"`
	PendingReason  string               `json:"pendingReason,omitempty"`
	PendingMessage string               `json:"pendingMessage,omitempty"`
	Requests       PodResourceRequests  `json:"requests"`
	Usage          *PodResourceRequests `json:"usage,omitempty"`
}

type VolcanoJobPhase string

const (
	VolcanoJobPending   VolcanoJobPhase = "Pending"
	VolcanoJobRunning   VolcanoJobPhase = "Running"
	VolcanoJobCompleted VolcanoJobPhase = "Completed"
	VolcanoJobFailed    VolcanoJobPhase = "Failed"
	VolcanoJobUnknown   VolcanoJobPhase = "Unknown"
)

type TrainingRunPhase string

const (
	TrainingRunPending   TrainingRunPhase = "Pending"
	TrainingRunRunning   TrainingRunPhase = "Running"
	TrainingRunSucceeded TrainingRunPhase = "Succeeded"
	TrainingRunFailed    TrainingRunPhase = "Failed"
	TrainingRunCancelled TrainingRunPhase = "Cancelled"
)

type TrainingRunSummary struct {
	Name                  string               `json:"name"`
	Namespace             string               `json:"namespace"`
	Status                string               `json:"status"`
	Epochs                string               `json:"epochs,omitempty"`
	Replicas              int                  `json:"replicas,omitempty"`
	MinAvailable          int                  `json:"minAvailable,omitempty"`
	CreatedAt             time.Time            `json:"createdAt"`
	Active                int32                `json:"active"`
	Succeeded             int32                `json:"succeeded"`
	Failed                int32                `json:"failed"`
	CPU                   string               `json:"cpu"`
	Memory                string               `json:"memory"`
	GPUs                  int                  `json:"gpus"`
	ScheduledAt           *time.Time           `json:"scheduledAt,omitempty"`
	StartedAt             *time.Time           `json:"startedAt,omitempty"`
	QueueWaitSeconds      *float64             `json:"queueWaitSeconds,omitempty"`
	StartupLatencySeconds *float64             `json:"startupLatencySeconds,omitempty"`
	Pods                  []TrainingPodSummary `json:"pods,omitempty"`
}

type podMetricsList struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Containers []struct {
			Usage corev1.ResourceList `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

type PodCounters struct {
	Active    int32 `json:"active"`
	Succeeded int32 `json:"succeeded"`
	Failed    int32 `json:"failed"`
}

type NodeResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	GPUs   string `json:"gpus"`
}

type NodeSummary struct {
	Name          string        `json:"name"`
	Ready         bool          `json:"ready"`
	Unschedulable bool          `json:"unschedulable"`
	Capacity      NodeResources `json:"capacity"`
	Allocatable   NodeResources `json:"allocatable"`
}

type Pagination struct {
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type TrainingRunsResponse struct {
	Data       []TrainingRunSummary `json:"data"`
	Pagination Pagination           `json:"pagination"`
}

var volcanoJobGVR = schema.GroupVersionResource{
	Group:    "batch.volcano.sh",
	Version:  "v1alpha1",
	Resource: "jobs",
}

const (
	// queue names must match the Volcano queue names configured in the cluster.
	TrainingQueue string = "ai-training"
	ResearchQueue string = "ai-research"
	PriorityQueue string = "ai-priority"

	// priority class names must match the PriorityClass names configured in the cluster.
	NormalPriority string = "kubeai-normal"
	HighPriority   string = "kubeai-high"

	// The namespace where training Jobs are created.
	namespace string = "kubeai-training"

	// These constants define the default intervals and timeouts for the training run reconciler.
	defaultReconcileInterval = 30 * time.Second
	defaultPendingTimeout    = 30 * time.Minute
)

func newClients() (kubernetes.Interface, dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags(
			"",
			clientcmd.RecommendedHomeFile,
		)
		if err != nil {
			return nil, nil, err
		}
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

func NewServer(repo *registory.TrainingRunRepo) *Server {
	clientset, dynamicClient, err := newClients()
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		client:        clientset,
		dynamicClient: dynamicClient,
		repo:          repo,
	}

}

func (s *Server) CreateTrainingRun(w http.ResponseWriter, r *http.Request) {
	var req CreateTrainingRunRequest

	// Decode the JSON request body into the CreateTrainingRunRequest struct.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request fields.
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if req.Epochs <= 0 {
		http.Error(w, "epochs must be greater than 0", http.StatusBadRequest)
		return
	}

	if err := validateResourceQuantity("cpu", req.CPU); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.CPU == "" {
		req.CPU = "500m"
	}

	if err := validateResourceQuantity("memory", req.Memory); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Memory == "" {
		req.Memory = "1Gi"
	}

	if req.GPUs < 0 || req.GPUs > 8 {
		http.Error(
			w,
			"gpus must be between 0 and 8 per worker",
			http.StatusBadRequest,
		)
		return
	}

	if req.Priority == "" {
		req.Priority = NormalPriority
	}

	if req.Replicas < 1 || req.Replicas > 8 {
		http.Error(w, "replicas must be between 1 and 8", http.StatusBadRequest)
		return
	}

	if req.Replicas == 0 {
		req.Replicas = 1
	}

	if req.MinAvailable < 1 || req.MinAvailable > req.Replicas {
		// Use the same error-response pattern as your existing validation.
		http.Error(
			w,
			"minAvailable must be between 1 and replicas",
			http.StatusBadRequest,
		)
		return
	}

	if req.MinAvailable == 0 {
		req.MinAvailable = req.Replicas
	}

	if req.Queue == "" {
		req.Queue = TrainingQueue
	}

	req.Queue = strings.TrimSpace(req.Queue)

	if err := validateTrainingRunResources(
		req.Queue,
		req.CPU,
		req.Memory,
		req.Replicas,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Queue {
	case TrainingQueue, ResearchQueue, PriorityQueue:
		// Supported queue.
	default:
		http.Error(
			w,
			"queue must be ai-training, ai-research, or ai-priority",
			http.StatusBadRequest,
		)
		return
	}

	var priorityClassName string

	switch req.Priority {
	case "normal":
		priorityClassName = NormalPriority
	case "high":
		priorityClassName = HighPriority
	default:
		http.Error(w, "priority must be normal or high", http.StatusBadRequest)
		return
	}

	requests := map[string]interface{}{
		"cpu":    req.CPU,
		"memory": req.Memory,
	}

	limits := map[string]interface{}{
		"cpu":    req.CPU,
		"memory": req.Memory,
	}

	if req.GPUs > 0 {
		gpuCount := strconv.Itoa(req.GPUs)
		requests["nvidia.com/gpu"] = gpuCount
		limits["nvidia.com/gpu"] = gpuCount
	}

	masterServiceName := req.Name + "-master"
	masterAddr := fmt.Sprintf(
		"%s.%s.svc.cluster.local",
		masterServiceName,
		namespace,
	)

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
				"schedulerName":     "volcano",
				"queue":             req.Queue,
				"minAvailable":      req.MinAvailable,
				"maxRetry":          int64(0),
				"priorityClassName": priorityClassName,
				"tasks": []interface{}{
					map[string]interface{}{
						"name":     "trainer",
						"replicas": int64(req.Replicas),
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"kubeai.io/managed-by":   "kubeai",
									"kubeai.io/component":    "training",
									"kubeai.io/training-run": req.Name,
								},
							},
							"spec": map[string]interface{}{
								"subdomain":         masterServiceName,
								"restartPolicy":     "Never",
								"priorityClassName": priorityClassName,
								"containers": []interface{}{
									map[string]interface{}{
										"name":            "trainer",
										"image":           "192.168.122.1:5000/kubeai-trainer:v0.4.0",
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
											map[string]interface{}{
												"name": "POD_NAME",
												"valueFrom": map[string]interface{}{
													"fieldRef": map[string]interface{}{
														"fieldPath": "metadata.name",
													},
												},
											},
											map[string]interface{}{
												"name":  "WORLD_SIZE",
												"value": strconv.Itoa(req.Replicas),
											},
											map[string]interface{}{
												"name":  "MASTER_ADDR",
												"value": masterAddr,
											},
											map[string]interface{}{
												"name":  "MASTER_PORT",
												"value": "29500",
											},
										},
										"resources": map[string]interface{}{
											"requests": requests,
											"limits":   limits,
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

	serviceCreated := false
	serviceName := req.Name + "-master"

	if req.Replicas > 1 {
		if err := s.createTrainingMasterService(ctx, req.Name, serviceName); err != nil {
			log.Printf(
				"failed to create Master Service for training run %q: %v",
				req.Name,
				err,
			)
			http.Error(
				w,
				"failed to create training networking Service",
				http.StatusInternalServerError,
			)
			return // Clean up the first Service here.
		}

		serviceCreated = true
	}

	created, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(namespace).
		Create(ctx, job, metav1.CreateOptions{})

	if err != nil {
		if serviceCreated {
			cleanupCtx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			cleanupErr := s.client.CoreV1().
				Services(namespace).
				Delete(
					cleanupCtx,
					serviceName,
					metav1.DeleteOptions{},
				)

			if cleanupErr != nil {
				log.Printf(
					"failed to clean up headless Service for training run %q: %v",
					req.Name,
					cleanupErr,
				)
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	run := registory.TrainingRun{
		Name:         created.GetName(),
		Namespace:    namespace,
		Epochs:       req.Epochs,
		Replicas:     req.Replicas,
		MinAvailable: req.MinAvailable,
		CPU:          req.CPU,
		Memory:       req.Memory,
		GPUs:         req.GPUs,
		Priority:     req.Priority,
		Queue:        req.Queue,
		Status:       "Pending",
	}

	if err := s.repo.CreateTrainingRunRecord(ctx, run); err != nil {
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
		} else {
			if serviceCreated {
				cleanupCtx, cancel := context.WithTimeout(
					context.Background(),
					5*time.Second,
				)
				defer cancel()

				cleanupErr := s.client.CoreV1().
					Services(namespace).
					Delete(
						cleanupCtx,
						serviceName,
						metav1.DeleteOptions{},
					)

				if cleanupErr != nil {
					log.Printf(
						"failed to clean up headless Service for training run %q: %v",
						req.Name,
						cleanupErr,
					)
				}
			}
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

	page, size, err := parsePagination(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	runsRecords, err := s.repo.GetAllTrainingRunRecords(ctx, page, size)
	if err != nil {
		log.Printf("query training run records: %v", err)
		http.Error(w, "failed to query training run records", http.StatusInternalServerError)
		return
	}

	total, err := s.repo.CountTrainingRunRecords(ctx)
	if err != nil {
		log.Printf("count training run records: %v", err)
		http.Error(w, "failed to count training run records", http.StatusInternalServerError)
		return
	}

	runs := make([]TrainingRunSummary, 0, len(runsRecords))
	for _, run := range runsRecords {
		summary := TrainingRunSummary{
			Name:         run.Name,
			Namespace:    run.Namespace,
			Status:       run.Status,
			Replicas:     run.Replicas,
			MinAvailable: run.MinAvailable,
			CreatedAt:    run.CreatedAt,
			Epochs:       strconv.Itoa(run.Epochs),
			CPU:          run.CPU,
			Memory:       run.Memory,
			GPUs:         run.GPUs,
		}

		// Volcano provides live status while the Job still exists.
		_, err := s.dynamicClient.
			Resource(volcanoJobGVR).
			Namespace(run.Namespace).
			Get(ctx, run.Name, metav1.GetOptions{})

		if err == nil {
			// calculate the number of active, succeeded, and failed Pods for this training run vcjob.
			counters, err := s.getPodCounters(ctx, run.Name)
			if err != nil {
				http.Error(w, "failed to query training Pod status", http.StatusInternalServerError)
				log.Printf("get Pod counters for %s: %v", run.Name, err)
				return
			}

			summary.Active = counters.Active
			summary.Succeeded = counters.Succeeded
			summary.Failed = counters.Failed

			podSummaries, err := s.getTrainingPodSummaries(ctx, run.Name)
			if err != nil {
				log.Printf(
					"get Pod summaries for training run %s: %v",
					run.Name,
					err,
				)
				http.Error(
					w,
					"failed to query training Pod details",
					http.StatusInternalServerError,
				)
				return
			}

			summary.Pods = podSummaries
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

	totalPages := (total + size - 1) / size

	response := TrainingRunsResponse{
		Data: runs,
		Pagination: Pagination{
			Page:       page,
			Size:       size,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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
		Namespace(namespace).
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
		Status:    string(volcanoJobPhaseToTrainingRunPhase(job)),
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

	// Verify that the Job exists before retrieving its logs.
	_, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(namespace).
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

	if err := s.repo.UpdateTrainingRunStatus(ctx, name, string(TrainingRunCancelled)); err != nil {
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

func (s *Server) ListNodes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	nodes, err := s.client.CoreV1().Nodes().List(
		ctx,
		metav1.ListOptions{},
	)
	if err != nil {
		log.Printf("list Kubernetes nodes: %v", err)
		http.Error(
			w,
			"failed to query Kubernetes nodes",
			http.StatusInternalServerError,
		)
		return
	}

	getResources := func(
		resources corev1.ResourceList,
	) NodeResources {
		return NodeResources{
			CPU:    resources.Cpu().String(),
			Memory: resources.Memory().String(),
			GPUs:   resources.Name(corev1.ResourceName("nvidia.com/gpu"), resource.DecimalSI).String(),
		}
	}

	result := make([]NodeSummary, 0, len(nodes.Items))

	for _, node := range nodes.Items {
		ready := false

		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				ready = condition.Status == corev1.ConditionTrue
				break
			}
		}

		result = append(result, NodeSummary{
			Name:          node.Name,
			Ready:         ready,
			Unschedulable: node.Spec.Unschedulable,
			Capacity:      getResources(node.Status.Capacity),
			Allocatable:   getResources(node.Status.Allocatable),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("encode node summaries: %v", err)
	}
}

func (s *Server) RunReconciler(
	ctx context.Context,
	intervals ...time.Duration,
) {
	// Default interval
	interval := defaultReconcileInterval

	// Override default if an interval is provided
	if len(intervals) > 0 && intervals[0] > 0 {
		interval = intervals[0]
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf(
		"Starting job reconciler, interval: %s",
		interval,
	)

	// Run reconciliation immediately on startup.
	s.runReconciliation(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping job reconciler")
			return

		case <-ticker.C:
			s.runReconciliation(ctx)
		}
	}
}

func (s *Server) runReconciliation(ctx context.Context) {
	// Give each reconciliation cycle a maximum execution time.
	reconcileCtx, cancel := context.WithTimeout(
		ctx,
		2*time.Minute,
	)
	defer cancel()

	if err := s.reconcileJobStatuses(reconcileCtx); err != nil {
		log.Printf("Job reconciliation completed with errors: %v", err)
	}
}

// RunPendingTimeoutReconciler periodically checks for training runs
// that have remained Pending longer than the configured timeout.
func (s *Server) RunPendingTimeoutReconciler(
	ctx context.Context,
	interval time.Duration,
	pendingTimeout time.Duration,
) {
	if interval <= 0 {
		interval = defaultReconcileInterval
	}
	if pendingTimeout <= 0 {
		pendingTimeout = defaultPendingTimeout
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately, then continue periodically.
	for {
		if err := s.reconcilePendingTimeouts(ctx, pendingTimeout, 100, 0); err != nil {
			log.Printf("pending-timeout reconciliation failed: %v", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) reconcileJobStatus(
	ctx context.Context,
	run *registory.TrainingRun, // Use the actual type returned by your repository.
) error {
	// 1. Retrieve the Volcano Job.
	job, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(run.Namespace).
		Get(ctx, run.Name, metav1.GetOptions{})

	if apierrors.IsNotFound(err) {
		// The Volcano Job no longer exists.
		// Retain the last status in PostgreSQL for now.
		return nil
	}

	if err != nil {
		return fmt.Errorf(
			"get Volcano Job %s: %w",
			run.Name,
			err,
		)
	}

	// 2. Determine the actual Job status.
	newStatus := string(volcanoJobPhaseToTrainingRunPhase(job))

	// 3. Synchronize the status with PostgreSQL.
	if run.Status != newStatus {
		if err := s.repo.UpdateTrainingRunStatus(
			ctx,
			run.Name,
			newStatus,
		); err != nil {
			return fmt.Errorf(
				"update status for training run %s: %w",
				run.Name,
				err,
			)
		}
	}

	// 4. Synchronize lifecycle timestamps.
	if err := s.syncTrainingRunTimestamps(ctx, run); err != nil {
		return fmt.Errorf(
			"sync timestamps for training run %s: %w",
			run.Name,
			err,
		)
	}

	return nil
}

func (s *Server) reconcileJobStatuses(ctx context.Context) error {
	// 1. Retrieve all training runs that need reconciliation.
	runs, err := s.repo.GetTrainingRunsToReconcile(ctx)
	if err != nil {
		return fmt.Errorf("get training runs to reconcile: %w", err)
	}

	// 2. Reconcile each training run.
	var reconcileErrors []error

	for i := range runs {
		run := &runs[i]

		if err := s.reconcileJobStatus(ctx, run); err != nil {
			log.Printf(
				"failed to reconcile training run %s/%s: %v",
				run.Namespace,
				run.Name,
				err,
			)

			reconcileErrors = append(reconcileErrors, err)
			continue
		}
	}

	// 3. Return aggregated errors, if any.
	return errors.Join(reconcileErrors...)
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

func (s *Server) getTrainingPodSummaries(
	ctx context.Context,
	jobName string,
) ([]TrainingPodSummary, error) {
	pods, err := s.client.CoreV1().
		Pods(namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf(
				"volcano.sh/job-name=%s",
				jobName,
			),
		})
	if err != nil {
		return nil, err
	}

	// Metrics are optional: a metrics API problem should not prevent
	// users from seeing Pod scheduling status and resource requests.
	usageByPod := make(map[string]PodResourceRequests)

	metricsPath := fmt.Sprintf(
		"/apis/metrics.k8s.io/v1beta1/namespaces/%s/pods",
		namespace,
	)

	raw, metricsErr := s.client.Discovery().
		RESTClient().
		Get().
		AbsPath(metricsPath).
		Param("labelSelector", fmt.Sprintf(
			"volcano.sh/job-name=%s",
			jobName,
		)).
		Do(ctx).
		Raw()

	if metricsErr == nil {
		var metrics podMetricsList
		if err := json.Unmarshal(raw, &metrics); err != nil {
			log.Printf("decode Pod metrics for %s: %v", jobName, err)
		} else {
			for _, item := range metrics.Items {
				var cpu, memory, gpu resource.Quantity

				for _, container := range item.Containers {
					if q, ok := container.Usage[corev1.ResourceCPU]; ok {
						cpu.Add(q)
					}
					if q, ok := container.Usage[corev1.ResourceMemory]; ok {
						memory.Add(q)
					}
					if q, ok := container.Usage[corev1.ResourceName("nvidia.com/gpu")]; ok {
						gpu.Add(q)
					}
				}

				usageByPod[item.Metadata.Name] = PodResourceRequests{
					CPU:    cpu.String(),
					Memory: memory.String(),
					GPUs:   gpu.String(),
				}
			}
		}
	} else {
		log.Printf("retrieve Pod metrics for %s: %v", jobName, metricsErr)
	}

	result := make([]TrainingPodSummary, 0, len(pods.Items))

	for _, pod := range pods.Items {
		summary := TrainingPodSummary{
			Name:     pod.Name,
			Phase:    string(pod.Status.Phase),
			NodeName: pod.Spec.NodeName,
			Requests: PodResourceRequests{},
		}

		// Report the sum of the resource requests across all containers.
		var cpuRequest, memoryRequest, gpuRequest resource.Quantity

		for _, container := range pod.Spec.Containers {
			if q, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				cpuRequest.Add(q)
			}
			if q, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				memoryRequest.Add(q)
			}
			if q, ok := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]; ok {
				gpuRequest.Add(q)
			}

		}

		// Image pulls, container creation, and similar startup failures
		// may be more informative than the generic Pod phase.
		for _, containerStatus := range pod.Status.ContainerStatuses {
			waiting := containerStatus.State.Waiting
			if waiting != nil && summary.PendingReason == "" {
				summary.PendingReason = waiting.Reason
				summary.PendingMessage = waiting.Message
			}
		}

		summary.Requests = PodResourceRequests{
			CPU:    cpuRequest.String(),
			Memory: memoryRequest.String(),
			GPUs:   gpuRequest.String(),
		}

		// A scheduling condition gives a more specific explanation
		// when Kubernetes cannot place the Pod.
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled &&
				condition.Status == corev1.ConditionFalse {
				summary.PendingReason = condition.Reason
				summary.PendingMessage = condition.Message
				break
			}
		}

		if usage, ok := usageByPod[pod.Name]; ok {
			summary.Usage = &usage
		}

		result = append(result, summary)
	}

	return result, nil
}

func (s *Server) syncTrainingRunTimestamps(
	ctx context.Context,
	run *registory.TrainingRun,
) error {
	pods, err := s.client.CoreV1().
		Pods(run.Namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf(
				"volcano.sh/job-name=%s",
				run.Name,
			),
		})
	if err != nil {
		return fmt.Errorf("list Pods for training run %q: %w", run.Name, err)
	}

	for _, pod := range pods.Items {
		var scheduledAt *time.Time

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodScheduled &&
				condition.Status == corev1.ConditionTrue {
				t := condition.LastTransitionTime.Time
				if !t.IsZero() {
					scheduledAt = &t
				}
				break
			}
		}

		if scheduledAt != nil && run.ScheduledAt == nil {
			if err := s.repo.UpdateTrainingRunScheduledAt(
				ctx, run.Name, *scheduledAt,
			); err != nil {
				return err
			}
			run.ScheduledAt = scheduledAt
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name != "trainer" ||
				containerStatus.State.Running == nil {
				continue
			}

			startedAt := containerStatus.State.Running.StartedAt.Time
			if startedAt.IsZero() || run.StartedAt != nil {
				continue
			}

			if err := s.repo.UpdateTrainingRunStartedAt(
				ctx, run.Name, startedAt,
			); err != nil {
				return err
			}
			run.StartedAt = &startedAt
		}
	}

	return nil
}

func (s *Server) reconcilePendingTimeouts(
	ctx context.Context,
	pendingTimeout time.Duration,
	size int,
	offset int,
) error {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	runs, err := s.repo.GetAllTrainingRunRecords(checkCtx, size, offset)
	if err != nil {
		return fmt.Errorf("list training runs: %w", err)
	}

	now := time.Now()

	for _, run := range runs {
		// Only expire runs that the database still considers Pending.
		if run.Status != "Pending" {
			continue
		}

		if now.Sub(run.CreatedAt) < pendingTimeout {
			continue
		}

		jobClient := s.dynamicClient.
			Resource(volcanoJobGVR).
			Namespace(run.Namespace)

		job, err := jobClient.Get(
			checkCtx,
			run.Name,
			metav1.GetOptions{},
		)

		if err == nil {
			// Do not time out a job that Volcano has already started.
			if volcanoJobPhaseToTrainingRunPhase(job) != TrainingRunPending {
				continue
			}

			// Only delete Jobs managed by this platform.
			labels := job.GetLabels()
			if labels["kubeai.io/managed-by"] != "kubeai" {
				log.Printf(
					"skip timeout cleanup for unmanaged Job %q",
					run.Name,
				)
				continue
			}

			deleteErr := jobClient.Delete(
				checkCtx,
				run.Name,
				metav1.DeleteOptions{},
			)
			if deleteErr != nil &&
				!apierrors.IsNotFound(deleteErr) {
				log.Printf(
					"delete timed-out Job %q: %v",
					run.Name,
					deleteErr,
				)
				continue
			}
		} else if !apierrors.IsNotFound(err) {
			log.Printf(
				"get Job %q during timeout check: %v",
				run.Name,
				err,
			)
			continue
		}

		// If the Job was already absent, or its deletion was accepted,
		// record the timeout. A database error will be retried next cycle.
		if err := s.repo.UpdateTrainingRunStatus(
			checkCtx,
			run.Name,
			"TimedOut",
		); err != nil {
			log.Printf(
				"update timed-out status for %q: %v",
				run.Name,
				err,
			)
			continue
		}

		log.Printf(
			"training run %q timed out after %s",
			run.Name,
			pendingTimeout,
		)
	}

	return nil
}

func (s *Server) createTrainingMasterService(
	ctx context.Context,
	runName string,
	serviceName string,
) error {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                "None",
			PublishNotReadyAddresses: true,
			Selector: map[string]string{
				"kubeai.io/training-run": runName,
				"volcano.sh/task-index":  "0",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "distributed",
					Port:       29500,
					TargetPort: intstr.FromInt32(29500),
				},
			},
		},
	}

	_, err := s.client.CoreV1().Services(namespace).Create(
		ctx, service, metav1.CreateOptions{},
	)
	return err
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
}

func volcanoJobPhaseToTrainingRunPhase(job *unstructured.Unstructured) TrainingRunPhase {
	phase, found, err := unstructured.NestedString(
		job.Object,
		"status",
		"state",
		"phase",
	)

	if err != nil || !found {
		return TrainingRunPending
	}

	switch phase {
	case string(VolcanoJobPending):
		return TrainingRunPending
	case string(VolcanoJobRunning):
		return TrainingRunRunning
	case string(VolcanoJobCompleted):
		return TrainingRunSucceeded
	case string(VolcanoJobFailed):
		return TrainingRunFailed
	default:
		return TrainingRunPending
	}
}

func validateResourceQuantity(
	name string,
	value string,
) error {
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return fmt.Errorf("invalid %s quantity %q: %w", name, value, err)
	}

	if quantity.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", name)
	}

	return nil
}

func validateTrainingRunResources(
	queue string,
	cpu string,
	memory string,
	replicas int,
) error {
	cpuQuantity, err := resource.ParseQuantity(cpu)
	if err != nil {
		return fmt.Errorf("invalid CPU quantity: %w", err)
	}

	memoryQuantity, err := resource.ParseQuantity(memory)
	if err != nil {
		return fmt.Errorf("invalid memory quantity: %w", err)
	}

	var maxCPU, maxMemory string

	switch queue {
	case TrainingQueue:
		maxCPU, maxMemory = "4", "8Gi"
	case ResearchQueue:
		maxCPU, maxMemory = "2", "4Gi"
	case PriorityQueue:
		maxCPU, maxMemory = "4", "8Gi"
	default:
		return fmt.Errorf("unsupported queue %q", queue)
	}

	maxJobCPU := resource.MustParse(maxCPU)
	maxJobMemory := resource.MustParse(maxMemory)

	totalCPUMilli := cpuQuantity.MilliValue() * int64(replicas)
	totalMemoryBytes := memoryQuantity.Value() * int64(replicas)

	if totalCPUMilli > maxJobCPU.MilliValue() {
		return fmt.Errorf(
			"CPU request exceeds the maximum of %s across all replicas for queue %s",
			maxCPU, queue,
		)
	}

	if totalMemoryBytes > maxJobMemory.Value() {
		return fmt.Errorf(
			"memory request exceeds the maximum of %s across all replicas for queue %s",
			maxMemory, queue,
		)
	}

	return nil
}

func parsePagination(r *http.Request) (int, int, error) {
	page := 1
	size := 10

	log.Printf("value page=%s, size=%s", r.URL.Query().Get("page"), r.URL.Query().Get("size"))

	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
		page = parsed
	}

	if value := r.URL.Query().Get("size"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, fmt.Errorf("size must be between 1 and 100")
		}
		size = parsed
	}

	return page, size, nil
}
