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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	GPUs     int    `json:"gpus"`
	Memory   string `json:"memory"`
	Priority string `json:"priority"`
	Replicas int    `json:"replicas"`
	Queue    string `json:"queue"`
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

type TrainingRunSummary struct {
	Name                  string               `json:"name"`
	Namespace             string               `json:"namespace"`
	Status                string               `json:"status"`
	Epochs                string               `json:"epochs,omitempty"`
	Replicas              int                  `json:"replicas,omitempty"`
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

	if err := validateResourceQuantity("cpu", req.CPU); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateResourceQuantity("memory", req.Memory); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.CPU == "" {
		req.CPU = "500m"
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
		req.Priority = "normal"
	}

	if req.Replicas == 0 {
		req.Replicas = 1
	}

	if req.Replicas < 1 || req.Replicas > 8 {
		http.Error(w, "replicas must be between 1 and 8", http.StatusBadRequest)
		return
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
				"minAvailable":      int64(req.Replicas),
				"maxRetry":          int64(0),
				"priorityClassName": priorityClassName,
				"tasks": []interface{}{
					map[string]interface{}{
						"name":     "trainer",
						"replicas": int64(req.Replicas),
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

	created, err := s.dynamicClient.
		Resource(volcanoJobGVR).
		Namespace(namespace).
		Create(ctx, job, metav1.CreateOptions{})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	run := registory.TrainingRun{
		Name:      created.GetName(),
		Namespace: namespace,
		Epochs:    req.Epochs,
		Replicas:  req.Replicas,
		CPU:       req.CPU,
		Memory:    req.Memory,
		GPUs:      req.GPUs,
		Priority:  req.Priority,
		Queue:     req.Queue,
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
			Replicas:  run.Replicas,
			CreatedAt: run.CreatedAt,
			Epochs:    strconv.Itoa(run.Epochs),
			CPU:       run.CPU,
			Memory:    run.Memory,
			GPUs:      run.GPUs,
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

			// sync the lifecycle timestamps for the training run in PostgreSQL.
			if err := s.syncTrainingRunTimestamps(ctx, &run); err != nil {
				log.Printf("sync lifecycle timestamps for %s: %v", run.Name, err)
				http.Error(
					w,
					"failed to synchronize training run timestamps",
					http.StatusInternalServerError,
				)
				return
			}

			summary.ScheduledAt = run.ScheduledAt
			summary.StartedAt = run.StartedAt

			if run.ScheduledAt != nil {
				seconds := run.ScheduledAt.Sub(run.CreatedAt).Seconds()
				summary.QueueWaitSeconds = &seconds
			}

			if run.StartedAt != nil {
				seconds := run.StartedAt.Sub(run.CreatedAt).Seconds()
				summary.StartupLatencySeconds = &seconds
			}
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
			if err := registory.UpdateTrainingRunScheduledAt(
				ctx, s.db, run.Name, *scheduledAt,
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

			if err := registory.UpdateTrainingRunStartedAt(
				ctx, s.db, run.Name, startedAt,
			); err != nil {
				return err
			}
			run.StartedAt = &startedAt
		}
	}

	return nil
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return new(v)
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
