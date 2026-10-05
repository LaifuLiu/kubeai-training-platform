package main

import "github.com/LaifuLiu/kubeai-training-platform/backend/router"

func main() {
	r := router.NewRouter()
	r.Run("8080")
}
