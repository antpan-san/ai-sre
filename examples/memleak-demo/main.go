// memleak-demo: 故意包含多种内存泄露风险的演示程序，用于 SRE/诊断演练。
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// leakStore 全局 map 只增不减，模拟缓存无界增长。
var leakStore = struct {
	sync.Mutex
	data map[string][]byte
}{
	data: make(map[string][]byte),
}

// orphanChans 泄漏的 goroutine 会阻塞在这些 channel 上。
var orphanChans []chan struct{}

func main() {
	port := envOr("PORT", "8080")
	chunkMB := envIntOr("LEAK_CHUNK_MB", 8)
	intervalSec := envIntOr("LEAK_INTERVAL_SEC", 3)

	go backgroundLeak(chunkMB, intervalSec)
	go goroutineLeak()

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/leak", func(w http.ResponseWriter, r *http.Request) {
		n := envIntOr("n", chunkMB)
		if v := r.URL.Query().Get("mb"); v != "" {
			if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
				n = parsed
			}
		}
		allocateOnce(n)
		fmt.Fprintf(w, "allocated ~%d MiB entry, store size=%d\n", n, storeLen())
	})

	addr := ":" + port
	log.Printf("memleak-demo listening on %s (chunk=%dMiB interval=%ds)", addr, chunkMB, intervalSec)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func backgroundLeak(chunkMB, intervalSec int) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()
	i := 0
	for range ticker.C {
		i++
		allocateOnce(chunkMB)
		if i%5 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("background leak tick=%d store=%d alloc=%.1fMiB sys=%.1fMiB goroutines=%d",
				i, storeLen(), float64(m.Alloc)/1024/1024, float64(m.Sys)/1024/1024, runtime.NumGoroutine())
		}
	}
}

func goroutineLeak() {
	for {
		ch := make(chan struct{})
		orphanChans = append(orphanChans, ch)
		go func(c chan struct{}) {
			<-c // 永不 close，goroutine 与 channel 常驻
		}(ch)
		time.Sleep(2 * time.Second)
	}
}

func allocateOnce(mb int) {
	if mb <= 0 {
		mb = 1
	}
	buf := make([]byte, mb*1024*1024)
	for i := range buf {
		buf[i] = byte(i % 251)
	}
	key := fmt.Sprintf("leak-%d-%d", time.Now().UnixNano(), len(leakStore.data))
	leakStore.Lock()
	leakStore.data[key] = buf
	leakStore.Unlock()
}

func storeLen() int {
	leakStore.Lock()
	defer leakStore.Unlock()
	return len(leakStore.data)
}

func statsHandler(w http.ResponseWriter, _ *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "goroutines=%d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "leak_store_entries=%d\n", storeLen())
	fmt.Fprintf(w, "orphan_channels=%d\n", len(orphanChans))
	fmt.Fprintf(w, "heap_alloc_mib=%.2f\n", float64(m.Alloc)/1024/1024)
	fmt.Fprintf(w, "heap_sys_mib=%.2f\n", float64(m.Sys)/1024/1024)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
