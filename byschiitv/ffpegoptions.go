package main

// streamToRTMP starts an FFmpeg command to stream a video file to nginx-rtmp.
// It listens on ctx and stops the stream when cancelled.
func FfmpegLightCommand(videoPath string, rtmpURL string) []string {
	// Example: ffmpeg -re -i input.mp4 -c copy -f flv rtmp://localhost/live/stream
	sliceCommand := []string{
		"-re",           // read at native frame rate
		"-i", videoPath, // input file
		"-c:v", "h264_v4l2m2m", // video codec
		"-c:a", "aac",
		"-vf", "scale=1280:720,fps=30", // scale video to 720p
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
