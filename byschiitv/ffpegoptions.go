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
	{Width: 1280, Height: 720, FPS: 30, VBitrate: "2000k", ABitrate: "128k"}, // HD
	{Width: 854, Height: 480, FPS: 23, VBitrate: "1000k", ABitrate: "96k"},   // SD
	{Width: 640, Height: 360, FPS: 15, VBitrate: "600k", ABitrate: "64k"},    // LD
}

var Qualities43 = []Q{
	{Width: 960, Height: 720, FPS: 30, VBitrate: "2000k", ABitrate: "128k"}, // HD
	{Width: 640, Height: 480, FPS: 23, VBitrate: "1000k", ABitrate: "96k"},  // SD
	{Width: 480, Height: 360, FPS: 15, VBitrate: "600k", ABitrate: "64k"},   // LD
}

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func FfmpegLightCommand(videoPath string, rtmpURL string, ciccione bool, textBanner bool) []string {

	var q Q
	if ciccione {
		q = Qualities43[0] // 4:3 aspect ratio, high quality
	} else {
		q = Qualities169[0] // 16:9 aspect ratio, high quality
	}

	var vFilter string
	if textBanner {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p,%s", q.Width, q.Height, q.FPS, getTextFilter(videoPath))
	} else {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p", q.Width, q.Height, q.FPS)
	}

	sliceCommand := []string{
		"-re",
		"-i", videoPath,
		"-c:v", "h264_v4l2m2m",
		"-preset", "veryfast",
		"-tune", "zerolatency",
		"-b:v", q.VBitrate,
		"-c:a", "aac",
		"-b:a", q.ABitrate,
		"-vf", vFilter,
		"-f", "flv",
		rtmpURL,
	}

	return sliceCommand
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

func FfmpegVeryLightCommand(videoPath string, rtmpURL string, ciccione bool, textBanner bool) []string {

	var q Q
	if ciccione {
		q = Qualities43[2] // 4:3 aspect ratio, medium quality
	} else {
		q = Qualities169[2] // 16:9 aspect ratio, medium quality
	}

	var vFilter string
	if textBanner {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p,%s", q.Width, q.Height, q.FPS, getTextFilter(videoPath))
	} else {
		vFilter = fmt.Sprintf("scale=%d:%d,fps=%d,format=yuv420p", q.Width, q.Height, q.FPS)
	}

	sliceCommand := []string{
		"-re",           // read at native frame rate
		"-i", videoPath, // input file
		"-c:v", "h264_v4l2m2m",
		"-b:v", q.VBitrate,
		"-c:a", "aac",
		"-b:a", q.ABitrate,
		"-vf", vFilter,
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
		// Normal video streaming
		// Decide command based on quality setting
		if video.HighQuality {
			cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegLightCommand(video.Path, rtmpURL, video.AspectRatio43, video.TextBanner)...)
		} else {
			cmd = exec.CommandContext(ctx, "ffmpeg", FfmpegVeryLightCommand(video.Path, rtmpURL, video.AspectRatio43, video.TextBanner)...)
		}
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
