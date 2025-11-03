package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// use gin in release mode by default for cleaner logging
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	rtmpURL := os.Getenv("RTMP_URL")
	if rtmpURL == "" {
		rtmpURL = "rtmp://iptvsim-nginx:1935/live/stream"
	}
	log.Printf("Using RTMP URL: %s", rtmpURL)

	srv := NewServer(rtmpURL)

	// Enqueue: /enque/<string> (capture rest of path)
	r.GET(`/enque/*item`, func(c *gin.Context) {
		item := c.Param("item")
		item = strings.TrimPrefix(item, "/")
		if item == "" {
			c.String(http.StatusBadRequest, "missing item to enqueue")
			return
		}
		n := srv.Append(item)
		c.JSON(http.StatusOK, gin.H{"enqueued": item, "length": n})
	})

	// List
	r.GET("/list", func(c *gin.Context) {
		list := srv.List()
		c.JSON(http.StatusOK, gin.H{"queue": list})
	})

	// Start
	r.GET("/start", func(c *gin.Context) {
		ok := srv.StartPlayer()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "already running"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "started"})
	})

	// Stop
	r.GET("/stop", func(c *gin.Context) {
		ok := srv.StopPlayer()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "not running"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "stopping"})
	})

	// Next: cancel current item only
	r.GET("/next", func(c *gin.Context) {
		cur, ok := srv.Current()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "not playing"})
			return
		}
		ok = srv.Next()
		if !ok {
			c.JSON(http.StatusOK, gin.H{"status": "not playing"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "skipped", "item": cur})
	})

	// Load playlist from JSON
	r.POST("/load", func(c *gin.Context) {
		var items []map[string]interface{}
		if err := c.BindJSON(&items); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		srv.LoadPlaylist(items)
		c.JSON(http.StatusOK, gin.H{"status": "loaded", "count": len(items)})
	})

	// root
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "iptvsim server. endpoints: /enque/<string> /next /list /start /stop /load (POST)")
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// List files in /media folder
	entries, err := os.ReadDir("/media")
	if err != nil {
		log.Printf("failed to read /media: %v", err)
	} else {
		for _, entry := range entries {
			log.Printf("/media: %s (dir: %v)", entry.Name(), entry.IsDir())
		}
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
	srv.StopPlayer()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("gin server: Shutdown: %v", err)
	}
	log.Println("gin server: exited")
}
