package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Q struct {
	Width    int
	Height   int
	FPS      int
	VBitrate string
	ABitrate string
}

var Qualities169 = []Q{
	// 0 Ultra (SW fallback for 1080p60)
	{Width: 1920, Height: 1080, FPS: 60, VBitrate: "10000k", ABitrate: "128k"}, // ULTRA_1080p60 (SW libx264 recommended)

	// 1 High (safe for Pi HW)
	{Width: 1920, Height: 1080, FPS: 30, VBitrate: "8000k", ABitrate: "128k"}, // HIGH_1080p30 (HW h264_v4l2m2m)

	// 2 Sports (fast motion with fewer pixels)
	{Width: 1280, Height: 720, FPS: 60, VBitrate: "6000k", ABitrate: "128k"}, // SPORTS_720p60 (HW ok)

	// 3 Standard HD
	{Width: 1280, Height: 720, FPS: 30, VBitrate: "3500k", ABitrate: "128k"}, // STANDARD_720p30

	// 4 Economy SD
	{Width: 854, Height: 480, FPS: 30, VBitrate: "1200k", ABitrate: "96k"}, // ECONOMY_480p30

	// 5 Mobile / low bandwidth
	{Width: 640, Height: 360, FPS: 30, VBitrate: "700k", ABitrate: "64k"}, // MOBILE_360p30
}

var Qualities43 = []Q{
	{Width: 960, Height: 720, FPS: 30, VBitrate: "2000k", ABitrate: "128k"}, // HD
	{Width: 640, Height: 480, FPS: 23, VBitrate: "1000k", ABitrate: "96k"},  // SD
	{Width: 480, Height: 360, FPS: 15, VBitrate: "600k", ABitrate: "64k"},   // LD
}

// FfmpegCommand builds an ffmpeg arg list for RTMP streaming.
// - Uses HW encoder (h264_v4l2m2m) for typical cases.
// - Automatically switches to software (libx264) for 1080p60, which Pi HW can't do.
// - Adds realtime-friendly flags: GOP≈2s, VBV, zerolatency, etc.
func FfmpegCommand(videoPath string, rtmpURL string, ciccione bool, quality int, textBanner bool) []string {
	// Pick quality safely
	var q Q
	if ciccione {
		if quality < 0 {
			quality = 0
		}
		if quality >= len(Qualities43) {
			quality = len(Qualities43) - 1
		}
		q = Qualities43[quality]
	} else {
		if quality < 0 {
			quality = 0
		}
		if quality >= len(Qualities169) {
			quality = len(Qualities169) - 1
		}
		q = Qualities169[quality]
	}

	// Build video filter chain
	var vFilter string
	if textBanner {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p,%s", q.Width, q.Height, q.FPS, getTextFilter(videoPath))
	} else {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p", q.Width, q.Height, q.FPS)
	}

	// Decide encoder
	usingRaspberryPi := true
	want1080p60 := (q.Width >= 1920 && q.FPS > 30)

	var encoder string
	var extra []string

	if want1080p60 || !usingRaspberryPi {
		// Fall back to software for 1080p60
		encoder = "libx264"
		// Real-time, low-latency RTMP-friendly settings
		level := "4.2" // for 1080p60
		gop := q.FPS * 2
		bufk := 2 * atoiK(q.VBitrate) // 2x VBV buffer
		extra = []string{
			"-preset", "veryfast", // try "ultrafast" if CPU is tight
			"-tune", "zerolatency",
			"-profile:v", "high",
			"-level:v", level,
			"-g", strconv.Itoa(gop),
			"-keyint_min", strconv.Itoa(gop),
			"-sc_threshold", "0",
			"-maxrate", q.VBitrate,
			"-bufsize", fmt.Sprintf("%dk", bufk),
			"-threads", "0",
		}
	} else {
		// Use Pi HW encoder
		encoder = "h264_v4l2m2m"
		// Keep a stable GOP; VBV helps RTMP stability on some setups
		gop := q.FPS * 2
		bufk := 2 * atoiK(q.VBitrate)
		extra = []string{
			"-g", strconv.Itoa(gop),
			"-maxrate", q.VBitrate,
			"-bufsize", fmt.Sprintf("%dk", bufk),
		}
	}

	fmt.Printf("FFmpeg command for %s (encoder=%v, quality=%d, textBanner=%v)\n", videoPath, encoder, quality, textBanner)

	// Assemble args
	args := []string{
		"-re",
		"-i", videoPath,
		"-vf", vFilter,
		"-pix_fmt", "yuv420p",
		"-c:v", encoder,
	}
	args = append(args, extra...)
	args = append(args,
		"-b:v", q.VBitrate,
		"-c:a", "aac",
		"-b:a", q.ABitrate,
		"-ar", "48000",
		"-ac", "2",
		"-f", "flv",
		rtmpURL,
	)

	return args
}

// atoiK converts "8000k" -> 8000 (kbit). Returns 0 on error.
func atoiK(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, "k")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func getTextFilter(description string) string {
	interval := 25        // seconds for one full scroll cycle, from appearance to disappearance
	duration := 10        // seconds the text is fully visible, from left edge to right edge
	scrollDistance := 1.8 // how far to scroll (1.0 = full width, 2.0 = twice width, etc)

	// remove first chars from description
	description = description[10:] // remove "/media/n. "
	// padd up to 100 chars
	strPadding := 150
	if len(description) < strPadding {
		description = description + strings.Repeat(" ", strPadding-len(description))
	}

	return fmt.Sprintf(
		"drawtext=text='%s':fontsize=24:fontcolor=white:"+
			"x=w-(mod(t\\,%d)*w*%.1f/%d):y=h-50:"+
			"enable='lt(mod(t\\,%d),%d)'",
		description,
		interval, scrollDistance, duration, // x position calculation
		interval, duration, // enable condition
	)
}

func FfmpegIdleStreamCommand(rtmpURL string, durationSeconds int, nextMovie string, description string, startTimeUnix int64) []string {
	currentTime := time.Now().Unix()
	secondsUntilStart := startTimeUnix - currentTime

	// Intelligently handle long descriptions:
	// - Short descriptions: show static centered text
	// - Long descriptions: scroll horizontally (ticker style)
	descLen := len(description)
	var descFilter string

	if descLen <= 80 {
		// Short description - static centered display
		descFilter = fmt.Sprintf(
			"drawtext=text='%s':fontsize=22:fontcolor=#cccccc:"+
				"x=(w-text_w)/2:y=h/2+60:"+
				"box=1:boxcolor=black@0.4:boxborderw=5",
			escapeFFmpegText(description),
		)
	} else {
		// Long description - scrolling ticker
		// Scrolls right to left continuously
		descFilter = fmt.Sprintf(
			"drawtext=text='%s':fontsize=22:fontcolor=#cccccc:"+
				"x=w-mod(t*80\\,w+tw):y=h/2+60:"+
				"box=1:boxcolor=black@0.4:boxborderw=5",
			escapeFFmpegText(description),
		)
	}

	videoFilter := fmt.Sprintf(
		"color=size=1280x720:rate=15:color=#0f0f1e,"+
			// Top: Stream status with pulsing effect
			"drawtext=text=' [||] INTERMISSION':fontsize=42:fontcolor=#ff6b6b:"+
			"x=(w-text_w)/2:y=80:"+
			"box=1:boxcolor=black@0.6:boxborderw=10:"+
			"alpha='0.85+0.15*sin(t)',"+

			// Middle section: Next movie title
			"drawtext=text='COMING UP NEXT':fontsize=28:fontcolor=#00d4ff:"+
			"x=(w-text_w)/2:y=h/2-120,"+

			"drawtext=text='%s':fontsize=46:fontcolor=white:"+
			"x=(w-text_w)/2:y=h/2-70:"+
			"box=1:boxcolor=black@0.5:boxborderw=8,"+

			// Description (smart display)
			"%s,"+

			// Bottom: Countdown timer
			"drawtext=text='Starting in\\: %%{eif\\:%.0f-t\\:d} seconds':fontsize=36:fontcolor=#4ecdc4:"+
			"x=(w-text_w)/2:y=h-120:"+
			"box=1:boxcolor=black@0.5:boxborderw=6",

		escapeFFmpegText(nextMovie),
		descFilter,
		float64(secondsUntilStart),
	)

	return []string{
		"-f", "lavfi",
		"-t", strconv.Itoa(durationSeconds),
		"-i", videoFilter,
		"-f", "lavfi",
		"-t", strconv.Itoa(durationSeconds),
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-c:v", "h264_v4l2m2m",
		"-b:v", "500k",
		"-c:a", "aac",
		"-b:a", "64k",
		"-f", "flv",
		rtmpURL,
	}
}

// Helper function to escape special characters for FFmpeg drawtext
func escapeFFmpegText(text string) string {
	// FFmpeg drawtext requires escaping special characters
	replacer := strings.NewReplacer(
		":", "\\:",
		"'", "\\'",
		"[", "\\[",
		"]", "\\]",
		",", "\\,",
		"\\", "\\\\",
	)
	return replacer.Replace(text)
}

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func StreamToRTMP(ctx context.Context, video PlaylistElement, rtmpURL string) error {
	log.Print("streaming: ", video.Desc())

	var cmd *exec.Cmd
	switch video := video.(type) {
	case IdleElement:
		cmd = exec.CommandContext(
			ctx,
			"ffmpeg",
			FfmpegIdleStreamCommand(
				rtmpURL,
				video.IdleSeconds,
				"desc", // video.NextMovie,
				video.Description,
				0, // video.StartTimeUnix
			)...,
		)
	case VideoElement:
		cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegCommand(video.Path, rtmpURL, video.AspectRatio43, video.QualityIndex, video.TextBanner)...)
	default:
		return fmt.Errorf("unknown video element type")
	}

	// Optional: capture output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if it was cancelled vs actual error
		if ctx.Err() == context.Canceled {
			log.Printf("streaming interrupted: %s", video.Desc())
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg error: %w", err)
	}

	log.Printf("streaming completed: %s", video.Desc())
	return nil
}

// ffprobe output structure
type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// GetVideoDuration uses ffprobe to get the duration of a video file.
func GetVideoDuration(ctx context.Context, videoPath string) (time.Duration, error) {
	// ffprobe -v error -show_format -of json input.mp4
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_format",
		"-of", "json",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed for %s: %w", videoPath, err)
	}

	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	durationSeconds, err := strconv.ParseFloat(probe.Format.Duration, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}

	return time.Duration(durationSeconds * float64(time.Second)), nil
}
