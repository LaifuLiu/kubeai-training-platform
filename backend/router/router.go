package router

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/LaifuLiu/kubeai-training-platform/backend/middleware"
	"github.com/LaifuLiu/kubeai-training-platform/backend/service"
)

type Router struct {
	mux *http.ServeMux
}

func NewRouter(db *sql.DB) *Router {
	server := service.NewServer(db)
	promHandler := service.Handler()

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promHandler)
	mux.HandleFunc("POST /api/v1/training-runs", server.CreateTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs", server.ListTrainingRuns)
	mux.HandleFunc("GET /api/v1/training-runs/{name}", server.GetTrainingRun)
	mux.HandleFunc("GET /api/v1/training-runs/{name}/logs", server.GetTrainingRunLogs)
	mux.HandleFunc("DELETE /api/v1/training-runs/{name}", server.DeleteTrainingRun)

	return &Router{mux: mux}
}

func (r *Router) Run(port string) {
	log.Println("KubeAI API listening on :" + port)

	handler := middleware.Metrics(middleware.Cors(r.mux))

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
