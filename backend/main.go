package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

func (s *Server) createTrainingRun(w http.ResponseWriter, r *http.Request) {
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

func main() {
	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	server := &Server{
		client: client,
	}

	http.HandleFunc("/api/v1/training-runs", server.createTrainingRun)

	log.Println("KubeAI API listening on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
