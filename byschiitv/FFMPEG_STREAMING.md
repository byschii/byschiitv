# FFmpeg RTMP Streaming Implementation

## Overview

This document explains how to adapt `simBackGroundTask` to use FFmpeg for streaming video files to a local nginx-rtmp server, with proper handling of cancellation signals (SIGTERM).

## Implementation

### Replace simBackGroundTask with streamToRTMP

```go
import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func streamToRTMP(ctx context.Context, videoPath string, rtmpURL string) error {
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-re",                    // read input at native frame rate
		"-i", videoPath,          // input file
		"-c", "copy",             // copy codec (no re-encoding)
		"-f", "flv",              // output format
		rtmpURL,                  // e.g., "rtmp://localhost/live/mystream"
	)
	
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
```

### Update Worker to Call streamToRTMP

Replace the call to `simBackGroundTask(itemCtx, item)` in the worker goroutine:

```go
// process the item with a cancellable context so /next can interrupt it
itemCtx, itemCancel := context.WithCancel(ctx)
s.mu.Lock()
s.currentCancel = itemCancel
s.currentItem = item
s.mu.Unlock()

// Stream the video file
rtmpURL := "rtmp://localhost/live/stream"
err := streamToRTMP(itemCtx, item, rtmpURL)
if err != nil && err != context.Canceled {
	log.Printf("streaming error: %v", err)
}

// clear current item (hold lock while clearing)
s.mu.Lock()
s.currentCancel = nil
s.currentItem = ""
s.mu.Unlock()
```

### Update Imports

Add `"os/exec"` to your imports:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"  // Add this
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)
```

## How SIGTERM Handling Works

1. **Cancellation Trigger**: When you call `/next` or `/stop`, it executes `s.currentCancel()`

2. **Context Cancellation**: This cancels `itemCtx`

3. **Automatic SIGTERM**: `exec.CommandContext` automatically sends **SIGTERM** to the FFmpeg process

4. **Graceful Shutdown**: FFmpeg receives SIGTERM, closes files cleanly, and exits

5. **Error Return**: `cmd.Run()` returns with the context cancellation error

## Key Features

- **Graceful Cancellation**: Uses `exec.CommandContext` which handles SIGTERM automatically
- **Error Handling**: Distinguishes between cancellation and actual FFmpeg errors
- **Logging**: Provides visibility into streaming lifecycle
- **No Zombie Processes**: Context cancellation ensures clean process termination

## Usage Example

1. Enqueue video files:
   ```bash
   curl http://localhost:8080/enque//path/to/video1.mp4
   curl http://localhost:8080/enque//path/to/video2.mp4
   ```

2. Start the worker:
   ```bash
   curl http://localhost:8080/start
   ```

3. Skip to next video:
   ```bash
   curl http://localhost:8080/next
   ```

4. Stop streaming:
   ```bash
   curl http://localhost:8080/stop
   ```

## Additional Considerations

### File Validation

Consider validating video files before streaming:

```go
if _, err := os.Stat(videoPath); os.IsNotExist(err) {
	return fmt.Errorf("video file not found: %s", videoPath)
}
```

### Stream Timeout

Add a maximum duration for streams:

```go
// In worker, before calling streamToRTMP:
itemCtx, itemCancel := context.WithTimeout(ctx, 2*time.Hour)
defer itemCancel()
```

### Dynamic Stream Keys

Make the RTMP URL configurable per video:

```go
// Server struct:
type Server struct {
	// ... existing fields ...
	rtmpBaseURL string  // e.g., "rtmp://localhost/live"
}

// In worker:
streamKey := fmt.Sprintf("stream_%d", time.Now().Unix())
rtmpURL := fmt.Sprintf("%s/%s", s.rtmpBaseURL, streamKey)
```

### FFmpeg Options

Common FFmpeg options for RTMP streaming:

```go
cmd := exec.CommandContext(ctx, "ffmpeg",
	"-re",                          // read at native frame rate
	"-i", videoPath,                // input file
	"-c:v", "libx264",             // video codec
	"-preset", "veryfast",          // encoding speed
	"-tune", "zerolatency",         // low latency
	"-c:a", "aac",                  // audio codec
	"-b:a", "128k",                 // audio bitrate
	"-f", "flv",                    // output format
	rtmpURL,
)
```

## Testing with nginx-rtmp

### Setup nginx-rtmp (Docker)

```bash
docker run -d -p 1935:1935 -p 8080:8080 \
  --name nginx-rtmp \
  tiangolo/nginx-rtmp
```

### Watch the Stream

Using ffplay:
```bash
ffplay rtmp://localhost/live/stream
```

Using VLC:
```bash
vlc rtmp://localhost/live/stream
```

## Troubleshooting

- **FFmpeg not found**: Ensure FFmpeg is installed and in PATH
- **Connection refused**: Verify nginx-rtmp is running and accessible
- **Permission denied**: Check video file permissions
- **Stream stuttering**: Adjust `-preset` and `-tune` parameters
