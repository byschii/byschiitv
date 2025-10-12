package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"
)

type PlaylistElement interface {
	Type() string
	Desc() string
}

type VideoElement struct {
	Path      string `json:"path"`
	HiQuality bool   `json:"hi_quality"`
}

func (v VideoElement) Type() string {
	return "video"
}
func (v VideoElement) Desc() string {
	return v.Path
}

type IdleElement struct {
	IdleSeconds int    `json:"idle_seconds"`
	Description string `json:"description,omitempty"`
}

func (i IdleElement) Type() string {
	return "idle"
}
func (i IdleElement) Desc() string {
	if i.Description != "" {
		return i.Description
	}
	return fmt.Sprintf("Idle for %d seconds", i.IdleSeconds)
}

// Server holds the queue and worker control.
type Server struct {
	mu               sync.Mutex
	playlist         []PlaylistElement
	currentlyPlaying int
	loop             bool
	// worker control: if called, stops after current item
	playerCancel  context.CancelFunc
	playerRunning bool
	// current item control
	currentCancel context.CancelFunc
}

type PlayerStatus struct {
	Running           bool
	Playing           bool
	CurrentIdx        int
	Loop              bool
	Length            int
	ProgrammedSeconds int
	ProgrammedHours   float32
}

func NewServer() *Server {
	return &Server{
		loop: true,
	}
}

func (s *Server) Append(item string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	pl := VideoElement{Path: item, HiQuality: false}
	s.playlist = append(s.playlist, pl)
	return len(s.playlist)
}

func (s *Server) Status() PlayerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	duration := 0
	for i, _ := range s.playlist {
		dur, err := s.GetDuration(i)
		if err == nil {
			duration += int(dur.Seconds())
		}
	}

	return PlayerStatus{
		Running:           s.playerRunning,
		Playing:           s.playerRunning && s.currentCancel != nil,
		CurrentIdx:        s.currentlyPlaying,
		Loop:              s.loop,
		Length:            len(s.playlist),
		ProgrammedSeconds: duration,
		ProgrammedHours:   float32(duration) / 3600.0,
	}
}

func (s *Server) Remove(index int) (PlaylistElement, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.playlist) {
		return nil, false
	}
	item := s.playlist[index]
	s.playlist = slices.Delete(s.playlist, index, index+1)
	return item, true
}

func (s *Server) List() []PlaylistElement {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PlaylistElement, len(s.playlist))
	copy(out, s.playlist)
	return out
}

func (s *Server) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlist = nil
}

func (s *Server) Current() (PlaylistElement, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentlyPlaying < 0 || s.currentlyPlaying >= len(s.playlist) {
		return nil, false
	}
	return s.playlist[s.currentlyPlaying], true
}

func (s *Server) Insert(index int, element PlaylistElement) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index > len(s.playlist) {
		return false
	}
	s.playlist = slices.Insert(s.playlist, index, element)
	return true
}

func (s *Server) Length() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.playlist)
}

// se player running state = true
// significa che il player e' in esecuzione (puo' essere in pausa)
// appena un video va in lista, viene riprodotto
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playerRunning
}

func (s *Server) IsPlaying() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.playerRunning && s.currentCancel != nil
}

func (s *Server) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.playerRunning || s.currentlyPlaying+1 >= len(s.playlist) {
		return false
	}

	if s.loop {
		s.currentlyPlaying = (s.currentlyPlaying + 1) % len(s.playlist)
	} else {
		s.currentlyPlaying++
	}
	if s.currentCancel != nil {
		s.currentCancel()
	}
	return true
}

func (s *Server) Previous() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.playerRunning || s.currentlyPlaying-1 < 0 {
		return false
	}

	if s.loop {
		s.currentlyPlaying = (s.currentlyPlaying - 1 + len(s.playlist)) % len(s.playlist)
	} else {
		s.currentlyPlaying--
	}
	if s.currentCancel != nil {
		s.currentCancel()
	}
	return true
}

func (s *Server) SetLoop(loop bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loop = loop
}

func (s *Server) IsLoop() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loop
}

func (s *Server) StartPlayer() bool {
	s.mu.Lock()
	if s.playerRunning {
		s.mu.Unlock()
		return false
	}
	playerLoopCtx, cancel := context.WithCancel(context.Background())
	s.playerCancel = cancel
	s.playerRunning = true
	s.currentlyPlaying = 0
	s.mu.Unlock()

	go s.playerLoop(playerLoopCtx)

	return true
}

// GetDuration returns the duration of the video at the given playlist index.
// Returns error if index is invalid or ffprobe fails.
func (s *Server) GetDuration(index int) (time.Duration, error) {
	s.mu.Lock()
	if index < 0 || index >= len(s.playlist) {
		s.mu.Unlock()
		return 0, fmt.Errorf("index %d out of bounds (playlist length: %d)", index, len(s.playlist))
	}
	switch item := s.playlist[index].(type) {
	case IdleElement:
		s.mu.Unlock()
		return time.Duration(item.IdleSeconds) * time.Second, nil
	case VideoElement:
		path := item.Path

		s.mu.Unlock()
		dur, err := GetVideoDuration(context.Background(), path)
		if err != nil {
			return 0, fmt.Errorf("ffprobe error for %s: %w", path, err)
		}
		return dur, nil

	default:
		s.mu.Unlock()
		return 0, fmt.Errorf("unknown playlist item type at index %d", index)
	}

}

func (s *Server) playerLoop(playerLoopCtx context.Context) {
	log.Println("worker: started")
	defer func() {
		s.mu.Lock()
		s.playerRunning = false
		s.playerCancel = nil
		s.mu.Unlock()
		log.Println("worker: stopped")
	}()

	for {
		select {
		case <-playerLoopCtx.Done():
			return
		default:
			item, ok := s.Current()
			if !ok {
				s.mu.Lock()
				if !s.playerRunning {
					s.mu.Unlock()
					return
				}
				s.mu.Unlock()
				time.Sleep(250 * time.Millisecond) // Wait before checking again
				continue
			}

			itemCtx, itemCancel := context.WithCancel(playerLoopCtx)
			s.mu.Lock()
			s.currentCancel = itemCancel
			s.mu.Unlock()

			// simBackGroundTask(itemCtx, item)
			// Stream the video file
			rtmpURL := "rtmp://iptvsim-nginx:1935/live/stream"
			err := StreamToRTMP(itemCtx, item, rtmpURL)
			if err != nil && err != context.Canceled {
				log.Printf("streaming error: %v", err)
			}
			s.Next()

			s.mu.Lock()
			s.currentCancel = nil
			s.mu.Unlock()
		}
	}
}

func (s *Server) StopPlayer() bool {
	s.mu.Lock()
	if !s.playerRunning || s.playerCancel == nil {
		s.mu.Unlock()
		return false
	}
	if s.currentCancel != nil {
		s.currentCancel()
	}
	s.playerCancel()
	s.mu.Unlock()
	return true
}
