package main

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func FfmpegLightCommand(videoPath string, rtmpURL string) []string {
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	sliceCommand := []string{
		"-re",           // read at native frame rate
		"-i", videoPath, // input file
		"-c:v", "h264_v4l2m2m", // video codec
		"-b:v", "1000k",
		"-maxrate", "1200k",
		"-bufsize", "2000k", // video bitrate
		"-vf", "scale=1280:720,fps=30,format=yuv420p", // scale video to 720p
		"-g", "50",
		"-keyint_min", "50", // GOP size
		"-num_output_buffers", "32",
		"-num_capture_buffers", "16",
		"-c:a", "aac",
		"-b:a", "64k",
		"-ar", "44100",
		"-ac", "2", // audio codec
		"-f", "flv", // output format
		rtmpURL}

	return sliceCommand
}

func FfmpegVeryLightCommand(videoPath string, rtmpURL string) []string {
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	sliceCommand := []string{
		"-re",           // read at native frame rate
		"-i", videoPath, // input file
		"-vf", "scale=854:480", // scale video to 480p
		"-c:v", "h264_v4l2m2m", // video codec
		"-preset", "veryfast", // encoding speed
		"-tune", "zerolatency", // low latency
		"-c:a", "aac", // audio codec
		"-b:a", "64k", // audio bitrate
		"-b:v", "1000k", // video bitrate
		"-f", "flv", // output format
		rtmpURL}

	return sliceCommand
}
