package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"simple-lb/config"
	"simple-lb/proxy"
)

// PID 파일 저장 위치
const pidFileName = "mylb.pid"

func main() {
	// 1. 커맨드라인 옵션 정의
	// mylb -config /etc/mylb/config.yaml -port :8080
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	reloadCmd := flag.Bool("reload", false, "Reload the running load balancer configuration")
	stopCmd := flag.Bool("stop", false, "Stop the running load balancer")
	flag.Parse()

	// 2. 명령 모드 처리 (-reload, -stop)
	if *reloadCmd {
		sendSignalToProcess(syscall.SIGHUP, "Reloading configuration")
		return
	}

	if *stopCmd {
		sendSignalToProcess(syscall.SIGTERM, "Stopping load balancer")
		return
	}

	// 3. 서버 모드 시작
	// config 로드
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	fmt.Println("Config loaded successfully:")
	fmt.Printf("Port: %s\n", cfg.Port)
	fmt.Printf("Strategy: %s\n", cfg.Strategy)
	fmt.Printf("Backends: %v\n", cfg.Backends)

	// PID 파일 생성
	if err := wirtePIDFile(); err != nil {
		log.Fatalf("PID 파일 생성 실패: %v", err)
	}

	defer os.Remove(pidFileName)

	// 로드 밸런서 인스턴스 생성
	lb := proxy.NewLoadBalancer(cfg.Backends)

	// 시그널 처리
	setupSignalHandler(lb, *configPath)

	server := &http.Server{
		Addr: cfg.Port,
		Handler: lb,
	}

	fmt.Printf("로드 밸런서 시작 (PID: %d) | Port: %s\n", os.Getpid(), cfg.Port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// PID 파일에 현재 프로세스 ID 쓰기
func wirtePIDFile() error {
	pid := os.Getpid()
	data := []byte(strconv.Itoa(pid))

	// 0644: 나만쓰고 남들은 읽기 가능
	return os.WriteFile(pidFileName, data, 0644)
}

// PID 파일을 읽어서 해당 프로세스에게 시그널 보내기
func sendSignalToProcess(sig syscall.Signal, message string) {
	// 1. PID 읽기
	data, err := os.ReadFile(pidFileName)
	if err != nil {
		fmt.Println("실행 중인 로드 밸런서를 찾을 수 없습니다.")
		os.Exit(1)
	}

	// 2. 텍스트를 숫자로 변환
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		fmt.Printf("잘못된 PID 파일 형식: %v\n", err)
		os.Exit(1)
	}

	// 3. 프로세스 찾기
	process, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("프로세스(PID %d)를 찾을 수 없습니다.\n", pid)
		os.Exit(1)
	}

	// 4. 시그널 보내기
	err = process.Signal(sig)
	if err != nil {
		fmt.Printf("프로세스(PID %d)에게 시그널을 보내는 데 실패했습니다: %v\n", pid, err)
		os.Exit(1)
	}

	fmt.Printf("%s: 프로세스(PID %d)에게 %s 시그널을 보냈습니다.\n", message, pid, sig)
}

func setupSignalHandler(lb *proxy.LoadBalancer, configPath string) {
	// 시그널 감지용 채널 생성
	// SIGHUP: 설정 리로드, SIGINT/SIGTERM: 종료
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	// 별도 고루틴에서 시그널 처리
	go func() {
		for {
			sig := <-sigs
			switch sig {
			case syscall.SIGHUP:
				log.Println("SIGUP 수신: 설정 파일을 다시 읽습니다.")

				newCfg, err := config.Load(configPath)
				if err != nil {
					log.Printf("설정 리로드 실패! 이전 설정 유지: %v", err)
				} else {
					lb.UpdateBackends(newCfg.Backends)
					log.Println("핫 리로드 완료!")
				}
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("시스템 종료 신호 수신. 종료합니다.")
				os.Remove(pidFileName)
				os.Exit(0)
			}
		}
	}()
}