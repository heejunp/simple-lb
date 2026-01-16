package main

import (
	"fmt"
	"log"
	"net/http"

	"simple-lb/config"
	"simple-lb/proxy"
)

func main() {
	// config.yaml 파일 로드
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("Config loaded successfully:")
	fmt.Printf("Port: %s\n", cfg.Port)
	fmt.Printf("Strategy: %s\n", cfg.Strategy)
	fmt.Printf("Backends: %v\n", cfg.Backends)

	// 로드 밸런서 인스턴스 생성
	lb := proxy.NewLoadBalancer(cfg.Backends)

	server := &http.Server{
		Addr: cfg.Port,
		Handler: lb,
	}

	log.Printf("Starting load balancer on %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}