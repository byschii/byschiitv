# Seek/Skip Functionality Design

## Overview
Add ability to seek within currently playing video via API endpoints. Client UI will not have seek controls - this is backend-only functionality for programmatic control.

## Architecture Approach
**Restart ffmpeg with `-ss` flag**
- When seeking, kill current ffmpeg process and restart from new position
- Acceptable 4-6 second rebuffering time for occasional seeks
- Fits "TV" philosophy - not smooth on-demand seeking

## Design Decisions
- **Skip past end:** Jump immediately to next video in playlist
- **Multiple rapid skips:** Latest skip wins (simplest implementation)
- **Skip during idle:** Return error (only works on video elements)
- **Position tracking:** Track `currentStartedAt` timestamp, calculate elapsed time
- **StartTime format:** `float64` seconds (e.g., 125.5)
- **API response:** `{from_position, to_position, video}`
- **HLS segments:** 3 seconds for faster rebuffering
- **ffmpeg `-ss`:** Before `-i` flag for faster seeking

## Changes Required

### 1. VideoElement Enhancement
Add seek position to VideoElement struct:
```go
type VideoElement struct {
    Path          string  `json:"path"`
    StartTime     float64 `json:"start_time,omitempty"`     // in seconds
    QualityIndex  int     `json:"quality_index,omitempty"`
    AspectRatio43 bool    `json:"aspect_ratio_4_3,omitempty"`
    TextBanner    bool    `json:"text_banner,omitempty"`
}
```

### 2. New API Endpoints

#### `/skip` - Skip forward/backward in current video
**Parameters:**
- `?delta=<seconds>` - positive for forward, negative for backward
  - Example: `/skip?delta=5` (skip 5s forward)
  - Example: `/skip?delta=-10` (skip 10s back)

**Behavior:**
1. Check if currently playing a video (not idle), return error if not
2. Get current video and calculate current playback position
3. Calculate target position: `current_position + delta`
4. Validate bounds:
   - If `target < 0`: Clamp to 0 (restart video)
   - If `target > duration`: Jump immediately to next video in playlist
5. Create new VideoElement with same path but `StartTime = target_position`
6. Insert as next item in playlist
7. Call `Next()` to skip to it

**Response:**
```json
{
  "from_position": 120.5,
  "to_position": 125.5,
  "video": "/media/video1.mp4"
}
```

**Edge cases:**
- Skip past end: Jump immediately to next video
- Skip before start: Clamp to 0 (restart video)
- Skip while not playing: Return error
- Skip during idle element: Return error
- Rapid multiple skips: Latest skip wins (simple - just check if playing)

#### `/jump` - Jump to percentage of current video
**Parameters:**
- `?percent=<0-100>` - jump to percentage of video
  - Example: `/jump?percent=50` (jump to halfway)

**Behavior:**
1. Check if currently playing a video (not idle), return error if not
2. Get current video duration via `GetDuration()` (already implemented)
3. Calculate target position: `duration * (percent / 100)`
4. Create new VideoElement with `StartTime = target_position`
5. Insert as next item and call `Next()`

**Response:**
```json
{
  "from_position": 120.5,
  "to_position": 180.0,
  "video": "/media/video1.mp4"
}
```

### 3. Playback Position Tracking
Track current playback position for skip/jump calculations.

**Add to Server struct:**
```go
type Server struct {
    // ... existing fields
    currentStartedAt time.Time  // when current video started playing
}
```

**Set `currentStartedAt` when video starts:**
- In `playerLoop()`, before calling `StreamToRTMP()`
- Reset on every video/item transition

**Calculate current position:**
```go
func (s *Server) getCurrentPosition() (float64, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !s.playerRunning || s.currentCancel == nil {
        return 0, fmt.Errorf("not playing")
    }

    elapsed := time.Since(s.currentStartedAt).Seconds()

    // Add StartTime offset if video was seeked
    currentVideo, ok := s.playlist[s.currentlyPlaying].(VideoElement)
    if !ok {
        return 0, fmt.Errorf("current item is not a video")
    }
    return currentVideo.StartTime + elapsed, nil
}
```

### 4. ffmpeg Integration
Modify `StreamToRTMP()` to accept `startTime` parameter and use `-ss` flag:

```go
func StreamToRTMP(ctx context.Context, item PlaylistElement, rtmpURL string, startTime float64) error {
    // ... existing code

    if startTime > 0 {
        args = append(args, "-ss", fmt.Sprintf("%.2f", startTime))
    }

    args = append(args, "-i", videoPath)
    // ... rest of ffmpeg args
}
```

**Note:** Use `-ss` **before** `-i` for faster seeking (less accurate but fine for streaming)

### 5. HLS Configuration
Update nginx.conf for shorter segments:
```nginx
hls_fragment 3s;          # 3-second segments
hls_playlist_length 15s;  # Keep last 5 segments
```

**Expected seek latency:** 4-6 seconds total

### 6. `/load` Enhancement
Allow specifying `start_time` when loading playlist:

**Example JSON:**
```json
[
  {
    "type": "video",
    "path": "/media/video1.mp4",
    "start_time": 120.5,
    "quality_index": 1
  },
  {
    "type": "idle",
    "idle_seconds": 10
  }
]
```

## Implementation Order

1. Add `StartTime` field to VideoElement
2. Update `LoadPlaylist()` to parse `start_time`
3. Modify `StreamToRTMP()` to use `-ss` flag
4. Add position tracking (`currentStartedAt`)
5. Implement `/skip` endpoint
6. Implement `/jump` endpoint
7. Update nginx.conf for 3s segments
8. Test edge cases

## Testing Scenarios

1. **Basic skip forward:** Skip 5s forward mid-video
2. **Basic skip backward:** Skip 10s backward mid-video
3. **Skip past end:** Skip 1000s forward in 5-minute video
4. **Skip before start:** Skip -1000s backward
5. **Skip while paused:** Player stopped, call `/skip`
6. **Skip during idle:** IdleElement playing, call `/skip`
7. **Load with start_time:** Load playlist with videos starting at specific times
8. **Jump to percentage:** Jump to 50% of video
9. **Rapid skips:** Call `/skip` multiple times quickly

## Notes

- **No UI changes** - API-only controls
- **Expected rebuffering:** 4-6 seconds per seek (acceptable for occasional use)
- **Performance:** Raspberry Pi 4B handles 3s HLS segments without issues
- **Philosophy:** Maintains "TV broadcast" feel - not smooth Netflix-style seeking
