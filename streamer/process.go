package streamer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IProcess is an interface around the FFMPEG process
type IProcess interface {
	Spawn(path, URI string) *exec.Cmd
}

// ProcessLoggingOpts describes options for process logging
type ProcessLoggingOpts struct {
	Enabled    bool   // Option to set logging for transcoding processes
	Directory  string // Directory for the logs
	MaxSize    int    // Maximum size of kept logging files in megabytes
	MaxBackups int    // Maximum number of old log files to retain
	MaxAge     int    // Maximum number of days to retain an old log file.
	Compress   bool   // Indicates if the log rotation should compress the log files
}

// Process is the main type for creating new processes
type Process struct {
	keepFiles   bool
	audio       bool
	loggingOpts ProcessLoggingOpts
}

// Type check
var _ IProcess = (*Process)(nil)

// NewProcess creates a new process able to spawn transcoding FFMPEG processes
func NewProcess(
	keepFiles bool,
	audio bool,
	loggingOpts ProcessLoggingOpts,
) *Process {
	return &Process{keepFiles, audio, loggingOpts}
}

// getHLSFlags are for getting the flags based on the config context
func (p Process) getHLSFlags() string {
	if p.keepFiles {
		return "append_list"
	}
	return "delete_segments+append_list"
}

// Spawn creates a new FFMPEG cmd
func (p Process) Spawn(path, URI string) *exec.Cmd {
	os.MkdirAll(path, os.ModePerm)
	processCommands := []string{
		"-y",
		"-fflags",
		"nobuffer",
		"-rtsp_transport",
		"tcp",
		"-i",
		URI,
		"-vsync",		"2",
		//"-preset",
		//"ultrafast",
		//"-tune",
		//"zerolatency",
		//"-c:v","libx264",
		//"-b:v","64k",
		//"-r","24",
		//"-c:v", "libx264",
		//"-crf", "21",
		//"-preset", "veryfast",
		//"-copyts",
		//"-g", "25",
		"-vcodec","copy",
		"-sc_threshold", "0",
		//"-movflags","frag_keyframe+empty_moov",
	}
	if p.audio {
		processCommands = append(processCommands, "-an")
	}
	processCommands = append(processCommands,
		"-hls_flags",
		p.getHLSFlags(),
		"-f","hls",
		"-segment_list_flags","live",
		//"-max_delay",//"1",
		//"-segment_time","3",
		"-hls_time","1",
		//"-hls_wrap","0",
		"-hls_list_size","1", //设置为1延迟在3秒左右
		//"-hls_playlist_type", "event",
		//"-segment_list_size","102400",
		"-muxdelay","1",
		"-hls_segment_filename",
		fmt.Sprintf("%s/%%d.ts", path),
		fmt.Sprintf("%s/index.m3u8", path),
	)
	fmt.Println(strings.Join(processCommands," "))
	cmd := exec.Command("ffmpeg", processCommands...)
	return cmd
}
