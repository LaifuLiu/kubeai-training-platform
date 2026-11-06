package router

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/LaifuLiu/kubeai-training-platform/backend/middleware"
	"github.com/LaifuLiu/kubeai-training-platform/backend/registory"
	"github.com/LaifuLiu/kubeai-training-platform/backend/service"
)

type Router struct {
	mux    *http.ServeMux
	server *service.Server
}

func NewRouter(repo *registory.TrainingRunRepo) *Router {
	promHandler := service.Handler()
	server := service.NewServer(repo)

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promHandler)
	mux.HandleFunc("POST /api/v1/training-runs", server.CreateTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs", server.ListTrainingRuns)
	mux.HandleFunc("GET /api/v1/training-runs/{name}", server.GetTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs/{name}/logs", server.GetTrainingRunLogs)
	mux.HandleFunc("DELETE /api/v1/training-runs/{name}", server.DeleteTrainingRun)
	mux.HandleFunc("GET /api/v1/nodes", server.ListNodes)

	return &Router{mux: mux, server: server}
}

func (r *Router) Run(port string) {
	log.Println("KubeAI API listening on :" + port)

	handler := middleware.Metrics(middleware.Cors(r.mux))

	// Create an application-level context.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go r.server.RunReconciler(ctx)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
