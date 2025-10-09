package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// simBackGroundTask prints the name letter by letter with a delay to simulate work.
// It listens on ctx and returns early when cancelled.
func simBackGroundTask(ctx context.Context, name string) {
	for _, char := range name {
		select {
		case <-ctx.Done():
			// interrupted: print newline to keep output tidy and return
			fmt.Println()
			return
		default:
			fmt.Printf("%c", char)
			time.Sleep(700 * time.Millisecond)
		}
	}
	fmt.Println()
}

// Server holds the queue and worker control.
type Server struct {
	mu            sync.Mutex
	queue         []string
	workerCancel  context.CancelFunc
	workerRunning bool
	// current item control
	currentCancel context.CancelFunc
	currentItem   string
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Enqueue(item string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, item)
	return len(s.queue)
}

func (s *Server) Dequeue() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return "", false
	}
	item := s.queue[0]
	s.queue = s.queue[1:]
	return item, true
}

func (s *Server) List() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// return a copy
	out := make([]string, len(s.queue))
	copy(out, s.queue)
	return out
}

func (s *Server) StartWorker() bool {
	s.mu.Lock()
	if s.workerRunning {
		s.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.workerCancel = cancel
	s.workerRunning = true
	s.mu.Unlock()

	go func() {
		log.Println("worker: started")
		defer func() {
			s.mu.Lock()
			s.workerRunning = false
			s.workerCancel = nil
			s.mu.Unlock()
			log.Println("worker: stopped")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// try to get an item
				item, ok := s.Dequeue()
				if !ok {
					// nothing to do; sleep briefly
					time.Sleep(600 * time.Millisecond)
					continue
				}
				// process the item with a cancellable context so /next can interrupt it
				itemCtx, itemCancel := context.WithCancel(ctx)
				s.mu.Lock()
				s.currentCancel = itemCancel
				s.currentItem = item
				s.mu.Unlock()

				simBackGroundTask(itemCtx, item)

				// clear current item (hold lock while clearing)
				s.mu.Lock()
				s.currentCancel = nil
				s.currentItem = ""
				s.mu.Unlock()
			}
		}
	}()

	return true
}

func (s *Server) StopWorker() bool {
	s.mu.Lock()
	if !s.workerRunning || s.workerCancel == nil {
		s.mu.Unlock()
		return false
	}
	// cancel current item if any
	if s.currentCancel != nil {
		s.currentCancel()
	}
	s.workerCancel()
	s.mu.Unlock()
	return true
}

func main() {
	srv := NewServer()

	mux := http.NewServeMux()

	// Enqueue: /enque/<string>
	mux.HandleFunc("/enque/", func(w http.ResponseWriter, r *http.Request) {
		// Everything after the prefix is the item
		item := strings.TrimPrefix(r.URL.Path, "/enque/")
		if item == "" {
			http.Error(w, "missing item to enqueue", http.StatusBadRequest)
			return
		}
		n := srv.Enqueue(item)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"enqueued": item, "length": n})
	})

	// Dequeue: /deque
	mux.HandleFunc("/deque", func(w http.ResponseWriter, r *http.Request) {
		item, ok := srv.Dequeue()
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"dequeued": item})
	})

	// List: /list
	mux.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		list := srv.List()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"queue": list})
	})

	// Start worker: /start
	mux.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		ok := srv.StartWorker()
		if !ok {
			json.NewEncoder(w).Encode(map[string]string{"status": "already running"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "started"})
	})

	// Stop worker: /stop
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		ok := srv.StopWorker()
		if !ok {
			json.NewEncoder(w).Encode(map[string]string{"status": "not running"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "stopping"})
	})

	// Next: /next - kill current item processing and move to next
	mux.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		srv.mu.Lock()
		if srv.currentCancel == nil {
			srv.mu.Unlock()
			json.NewEncoder(w).Encode(map[string]string{"status": "no current item"})
			return
		}
		// cancel current item
		srv.currentCancel()
		// clear here; worker will also clear after stopping the item
		srv.currentCancel = nil
		cur := srv.currentItem
		srv.currentItem = ""
		srv.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]string{"status": "skipped", "item": cur})
	})

	// health or root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "iptvsim server. endpoints: /enque/<string> /deque /list /start /stop")
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("server: starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("server: shutting down")
	// try to stop worker if running
	srv.StopWorker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server: Shutdown: %v", err)
	}
	log.Println("server: exited")
}
