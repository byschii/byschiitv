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

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func FfmpegLightCommand(videoPath string, rtmpURL string) []string {
	textFilter := fmt.Sprintf(
		"drawtext=text='%s':fontsize=24:fontcolor=white:"+
			"x=w-(mod(t\\,%d)*w*%.1f/%d):y=h-50:"+
			"enable='lt(mod(t\\,%d),%d)'",
		description,
		interval, scrollDistance, duration, // x position calculation
		interval, duration, // enable condition
	)

	sliceCommand := []string{
		"-re",
		"-i", videoPath,
		"-c:v", "h264_v4l2m2m",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-b:v", "1000k", // set bitrate
		"-c:a", "aac",
		"-b:a", "96k",
		"-vf", "scale=1280:720,fps=30,format=yuv420p",
		"-f", "flv",
		rtmpURL,
	}

	return sliceCommand
}

func FfmpegVeryLightCommand(videoPath string, rtmpURL string) []string {
	textFilter := fmt.Sprintf(
		"drawtext=text='%s':fontsize=24:fontcolor=white:"+
			"x=w-(mod(t\\,%d)*w*%.1f/%d):y=h-50:"+
			"enable='lt(mod(t\\,%d),%d)'",
		description,
		interval, scrollDistance, duration, // x position calculation
		interval, duration, // enable condition
	)

	sliceCommand := []string{
		"-re",           // read at native frame rate
		"-i", videoPath, // input file
		"-c:v", "h264_v4l2m2m",
		"-b:v", "800k", // set video bitrate
		"-c:a", "aac",
		"-b:a", "64k", // set audio bitrate
		"-vf", "scale=854:480,fps=24,format=yuv420p",
		"-f", "flv", // output format
		rtmpURL}

	return sliceCommand

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
			"drawtext=text='⏸ INTERMISSION':fontsize=42:fontcolor=#ff6b6b:"+
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
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	var cmd *exec.Cmd
	if video.IdleLength > 0 {
		cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegIdleStreamCommand(rtmpURL, video.IdleLength)...)
	} else {
		if video.HiQuality {
			cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegLightCommand(video.Path, rtmpURL)...)
		} else {
			cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegVeryLightCommand(video.Path, rtmpURL)...)
		}
	}

	// Optional: capture output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("streaming: %s -> %s", video.Path, rtmpURL)

	if err := cmd.Run(); err != nil {
		// Check if it was cancelled vs actual error
		if ctx.Err() == context.Canceled {
			log.Printf("streaming interrupted: %s", video.Path)
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg error: %w", err)
	}

	log.Printf("streaming completed: %s", video.Path)
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
