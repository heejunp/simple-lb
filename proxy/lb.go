package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Backend 구조체: URL + Health Check
type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex // 동시성 제어를 위한 뮤텍스
	ReverseProxy *httputil.ReverseProxy
}

// SetAlive: 서버 상태를 변경 (Write Lock)
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

// IsAlive: 서버 상태 확인 (Read Lock)
func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

// LoadBalancer 구조체
type LoadBalancer struct {
	backendMux sync.RWMutex
	backends []*Backend
	current  uint64
}

// NewLoadBalancer 생성자: URL 문자열 리스트를 받아서 초기화
func NewLoadBalancer(urls []string) *LoadBalancer {
	lb := &LoadBalancer{
		current: 0,
	}

	// 초기 백엔드 설정
	lb.UpdateBackends(urls)

	// 별도의 고루틴으로 헬스 체크 시작
	go lb.HealthCheck()

	return lb
}

// UpdateBackends: 실행 중에 백엔드 리스트를 통째로 교채 (Hot Reload)
func (lb *LoadBalancer) UpdateBackends(urls []string) {
	var newBackends []*Backend
	
	for _, u := range urls {
		parseUrl, err := url.Parse(u)
		if err != nil {
			log.Printf("[Error] Fail to parse URL %s: %v", u, err)
			continue
		}

		newBackends = append(newBackends, &Backend{
			URL: parseUrl,
			Alive: true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(parseUrl),
		})
	}

	// 교체하는 순간에 Write Lock을 걸어 안전하게 교체
	lb.backendMux.Lock()
	lb.backends = newBackends
	lb.backendMux.Unlock()

	log.Printf("[Info] Updated backends: %v\n", urls)
}

// HealthCheck: 주기적으로 서버에 연결을 시도해서 상태 업데이트
func (lb *LoadBalancer) HealthCheck() {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	for range t.C {
		// 현재 시점의 백엔드 리스트를 읽기 위해 Read Lock 사용
		lb.backendMux.RLock()
		currentBackends := lb.backends
		lb.backendMux.RUnlock()
		
		log.Println("--- Health Check Start ---")

		for _, b := range currentBackends {
			status := "up"
			alive := isBackendAlive(b.URL)
			b.SetAlive(alive)

			if !alive {
				status = "down"
			}

			log.Printf("%s [%s]\n", b.URL, status)
		}
		log.Println("--- Health Check End ---")
	}
}

// isBackendAlive: 실제로 TCP 연결을 시도해봄 (Ping)
func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		return false
	}
	
	conn.Close()
	return true
}

// NextBackend: '살아있는' 서버만 골라서 반환 (무한 루프 방지)
func (lb *LoadBalancer) NextBackend() *Backend {
	lb.backendMux.RLock()
	defer lb.backendMux.RUnlock()

	l := len(lb.backends)
	if l == 0 {
		return nil
	}
	
	// 1. Atomic 연산을 이용해 current 값을 1 증가시킴 (Lock 없이 안전하게 증가)
	next := atomic.AddUint64(&lb.current, 1)

	log.Printf("Debug: Current Count %d, Modulo %d", next, int(next)%l)

	// 2. 시작 지점부터 한 바퀴 돌면서 살아있는 서버를 찾음
	for i := 0; i < l; i++ {
		idx := (int(next) + i) % l

		if lb.backends[idx].IsAlive() {
			if i != 0 {
				atomic.StoreUint64(&lb.current, uint64(idx))
			}

			return lb.backends[idx]
		}
	}

	return nil
}

// ServeHTTP: http.Handler 인터페이스 구현
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := lb.NextBackend()
	if target == nil {
		http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
		return
	}

	log.Printf("[LB] Redirecting request to: %s\n", target.URL.Host)
	target.ReverseProxy.ServeHTTP(w, r)
}