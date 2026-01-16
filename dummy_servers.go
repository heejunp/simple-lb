package main

import (
	"fmt"
	"net/http"
	"sync"
)

func runServer(port string, name string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] 요청 받음!\n", name)
		fmt.Fprintf(w, "안녕하세요! 저는 백엔드 서버 [%s] 입니다.", name)
	})

	fmt.Printf("백엔드 서버 [%s] 시작 (Port %s)\n", name, port)
	http.ListenAndServe(port, mux)
}

func main() {
	var wg sync.WaitGroup
	wg.Add(3)

	// 9001, 9002, 9003 포트에 서버 3개 동시에 띄우기
	go func() { runServer(":9001", "Server 1"); wg.Done() }()
	go func() { runServer(":9002", "Server 2"); wg.Done() }()
	go func() { runServer(":9003", "Server 3"); wg.Done() }()

	wg.Wait()
}