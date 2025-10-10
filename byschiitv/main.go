package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func streamToRTMP(ctx context.Context, videoPath string, rtmpURL string) error {
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegLightCommand(videoPath, rtmpURL)...)

	// Optional: capture output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("streaming: %s -> %s", videoPath, rtmpURL)

	if err := cmd.Run(); err != nil {
		// Check if it was cancelled vs actual error
		if ctx.Err() == context.Canceled {
			log.Printf("streaming interrupted: %s", videoPath)
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg error: %w", err)
	}

	log.Printf("streaming completed: %s", videoPath)
	return nil
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

func NewServer() *Server { return &Server{} }

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
				item, ok := s.Dequeue()
				if !ok {
					time.Sleep(600 * time.Millisecond)
					continue
				}
				itemCtx, itemCancel := context.WithCancel(ctx)
				s.mu.Lock()
				s.currentCancel = itemCancel
				s.currentItem = item
				s.mu.Unlock()

				// simBackGroundTask(itemCtx, item)
				// Stream the video file
				rtmpURL := "rtmp://iptvsim-nginx:1935/live/stream"
				err := streamToRTMP(itemCtx, item, rtmpURL)
				if err != nil && err != context.Canceled {
					log.Printf("streaming error: %v", err)
				}
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
	if s.currentCancel != nil {
		s.currentCancel()
	}
	s.workerCancel()
	s.mu.Unlock()
	return true
}

func main() {
	// use gin in release mode by default for cleaner logging
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	srv := NewServer()

	// Enqueue: /enque/<string> (capture rest of path)
	r.GET(`/enque/*item`, func(c *gin.Context) {
		item := c.Param("item")
		item = strings.TrimPrefix(item, "/")
		if item == "" {
			c.String(http.StatusBadRequest, "missing item to enqueue")
			return
		}
		n := srv.Enqueue(item)
		c.JSON(http.StatusOK, gin.H{"enqueued": item, "length": n})
	})

	// Dequeue
	r.GET("/deque", func(c *gin.Context) {
		item, ok := srv.Dequeue()
		if !ok {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusOK, gin.H{"dequeued": item})
	})

	// List
	r.GET("/list", func(c *gin.Context) {
		list := srv.List()
		c.JSON(http.StatusOK, gin.H{"queue": list})
	})

	// Start
	r.GET("/start", func(c *gin.Context) {
		ok := srv.StartWorker()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "already running"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "started"})
	})

	// Stop
	r.GET("/stop", func(c *gin.Context) {
		ok := srv.StopWorker()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "not running"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "stopping"})
	})

	// Next: cancel current item only
	r.GET("/next", func(c *gin.Context) {
		srv.mu.Lock()
		if srv.currentCancel == nil {
			srv.mu.Unlock()
			c.JSON(http.StatusOK, gin.H{"status": "no current item"})
			return
		}
		srv.currentCancel()
		srv.currentCancel = nil
		cur := srv.currentItem
		srv.currentItem = ""
		srv.mu.Unlock()

		c.JSON(http.StatusOK, gin.H{"status": "skipped", "item": cur})
	})

	// root
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "iptvsim server. endpoints: /enque/<string> /deque /list /start /stop")
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("gin server: starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("gin server: ListenAndServe: %v", err)
		}
	}()

	<-stop
	log.Println("gin server: shutting down")
	srv.StopWorker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("gin server: Shutdown: %v", err)
	}
	log.Println("gin server: exited")
}
