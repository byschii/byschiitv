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
	Path          string  `json:"path"`
	StartTime     float64 `json:"start_time,omitempty"`
	QualityIndex  int     `json:"quality_index,omitempty"`
	AspectRatio43 bool    `json:"aspect_ratio_4_3,omitempty"`
	TextBanner    bool    `json:"text_banner,omitempty"`
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
	rtmpURL       string
	// seek tracking (reset on video change)
	currentStartTime float64   // -ss offset for current video
	currentStartedAt time.Time // when current video started playing
	currentDuration  float64   // cached duration of current video in seconds
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

func NewServer(rtmpURL string) *Server {
	if rtmpURL == "" {
		rtmpURL = "rtmp://iptvsim-nginx:1935/live/stream"
	}
	return &Server{
		loop:    true,
		rtmpURL: rtmpURL,
	}
}

func (s *Server) Append(item string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	pl := VideoElement{Path: item, QualityIndex: 1}
	s.playlist = append(s.playlist, pl)
	return len(s.playlist)
}

func (s *Server) Status() PlayerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	duration := 0
	for i := range s.playlist {
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
	if !s.playerRunning {
		return false
	}

	if s.currentlyPlaying+1 >= len(s.playlist) {
		if !s.loop {
			return false
		}
		s.currentlyPlaying = 0
	} else {
		s.currentlyPlaying++
	}

	s.resetSeekState()
	if s.currentCancel != nil {
		s.currentCancel()
	}
	return true
}

func (s *Server) Previous() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.playerRunning {
		return false
	}

	if s.currentlyPlaying-1 < 0 {
		if !s.loop {
			return false
		}
		s.currentlyPlaying = len(s.playlist) - 1
	} else {
		s.currentlyPlaying--
	}

	s.resetSeekState()
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

// resetSeekState resets seek position and duration tracking.
// Must be called with lock held.
func (s *Server) resetSeekState() {
	s.currentStartTime = 0
	s.currentDuration = 0
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
			rtmpURL := s.rtmpURL
			startTime := s.currentStartTime
			s.currentStartedAt = time.Now()
			// cache duration for current video
			if v, ok := item.(VideoElement); ok {
				if dur, err := GetVideoDuration(context.Background(), v.Path); err == nil {
					s.currentDuration = dur.Seconds()
				}
			} else if idle, ok := item.(IdleElement); ok {
				s.currentDuration = float64(idle.IdleSeconds)
			}
			s.mu.Unlock()

			// Stream the video file
			err := StreamToRTMP(itemCtx, item, rtmpURL, startTime)
			if err != nil && err != context.Canceled {
				log.Printf("streaming error: %v", err)
			}

			s.mu.Lock()
			s.resetSeekState()
			s.mu.Unlock()

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

func (s *Server) LoadPlaylist(items []map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playlist = nil

	for _, item := range items {
		itemType, ok := item["type"].(string)
		if !ok {
			continue
		}

		switch itemType {
		case "video":
			path, _ := item["path"].(string)
			qualityIndex := 0
			if qi, ok := item["quality_index"].(float64); ok {
				qualityIndex = int(qi)
			}
			startTime := 0.0
			if st, ok := item["start_time"].(float64); ok {
				startTime = st
			}
			aspectRatio43, _ := item["aspect_ratio_4_3"].(bool)
			textBanner, _ := item["text_banner"].(bool)
			s.playlist = append(s.playlist, VideoElement{
				Path:          path,
				StartTime:     startTime,
				QualityIndex:  qualityIndex,
				AspectRatio43: aspectRatio43,
				TextBanner:    textBanner,
			})
		case "idle":
			idleSeconds := int(item["idle_seconds"].(float64))
			description, _ := item["description"].(string)
			s.playlist = append(s.playlist, IdleElement{
				IdleSeconds: idleSeconds,
				Description: description,
			})
		}
	}
	return nil
}

// SkipResponse is the response for skip/jump endpoints
type SkipResponse struct {
	FromPosition float64 `json:"from_position"`
	ToPosition   float64 `json:"to_position"`
	Video        string  `json:"video"`
}

// getCurrentPosition returns current playback position in seconds
func (s *Server) getCurrentPosition() (float64, error) {
	// must be called with lock held
	if !s.playerRunning || s.currentCancel == nil {
		return 0, fmt.Errorf("not playing")
	}

	elapsed := time.Since(s.currentStartedAt).Seconds()
	return s.currentStartTime + elapsed, nil
}

// Skip skips delta seconds in current video. Positive = forward, negative = backward.
// Returns error if not playing a video. Skips to next video if past end.
func (s *Server) Skip(delta float64) (*SkipResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.playerRunning || s.currentCancel == nil {
		return nil, fmt.Errorf("not playing")
	}

	currentVideo, ok := s.playlist[s.currentlyPlaying].(VideoElement)
	if !ok {
		return nil, fmt.Errorf("current item is not a video")
	}

	fromPos, err := s.getCurrentPosition()
	if err != nil {
		return nil, err
	}

	toPos := fromPos + delta

	// Clamp to 0 if before start
	if toPos < 0 {
		toPos = 0
	}

	// If past duration, skip to next video
	if s.currentDuration > 0 && toPos >= s.currentDuration {
		s.mu.Unlock()
		s.Next()
		s.mu.Lock()
		return &SkipResponse{
			FromPosition: fromPos,
			ToPosition:   s.currentDuration,
			Video:        currentVideo.Path,
		}, nil
	}

	// Restart current video at new position
	s.currentStartTime = toPos
	if s.currentCancel != nil {
		s.currentCancel()
	}

	return &SkipResponse{
		FromPosition: fromPos,
		ToPosition:   toPos,
		Video:        currentVideo.Path,
	}, nil
}

// Jump jumps to a percentage (0-100) of current video.
func (s *Server) Jump(percent float64) (*SkipResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.playerRunning || s.currentCancel == nil {
		return nil, fmt.Errorf("not playing")
	}

	currentVideo, ok := s.playlist[s.currentlyPlaying].(VideoElement)
	if !ok {
		return nil, fmt.Errorf("current item is not a video")
	}

	if s.currentDuration <= 0 {
		return nil, fmt.Errorf("duration not available")
	}

	fromPos, err := s.getCurrentPosition()
	if err != nil {
		return nil, err
	}

	// Clamp percent
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	toPos := s.currentDuration * (percent / 100.0)

	// Restart current video at new position
	s.currentStartTime = toPos
	if s.currentCancel != nil {
		s.currentCancel()
	}

	return &SkipResponse{
		FromPosition: fromPos,
		ToPosition:   toPos,
		Video:        currentVideo.Path,
	}, nil
}
