package router

import (
	"log"
	"net/http"

	"github.com/LaifuLiu/kubeai-training-platform/backend/middleware"
	"github.com/LaifuLiu/kubeai-training-platform/backend/service"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter() *Router {
	server := service.NewServer()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/training-runs", server.CreateTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs", server.ListTrainingRuns)
	mux.HandleFunc("GET /api/v1/training-runs/{name}", server.GetTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs/{name}/logs", server.GetTrainingRunLogs)
	mux.HandleFunc("DELETE /api/v1/training-runs/{name}", server.DeleteTrainingRun)

	return &Router{mux: mux}
}

func (r *Router) Run(port string) {
	log.Println("KubeAI API listening on :" + port)

	if err := http.ListenAndServe(":"+port, middleware.Cors(r.mux)); err != nil {
		log.Fatal(err)
	}
}
