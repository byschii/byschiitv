package main

import (
	"context"
	"sync"
)

type PlaylistElement struct {
	Path string `json:"path"`
}

// Server holds the queue and worker control.
type Server struct {
	mu            sync.Mutex
	queue         []PlaylistElement
	workerCancel  context.CancelFunc
	workerRunning bool
	// current item control
	currentCancel context.CancelFunc
	currentItem   string
}
