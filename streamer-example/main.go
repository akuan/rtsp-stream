package main


import (
	"fmt"
	"github.com/riltech/streamer"
	"time"
)

func main() {
	stream, id := streamer.NewStream(
		"rtsp://admin:abcd1234@192.168.0.119:554/Streaming/Channels/1", // URI of raw RTSP stream
		"./videos", // Directory where to store video chunks and indexes. Should exist already
		true, // Indicates if stream should be keeping files after it is stopped or clean the directory
		true, // Indicates if Audio should be enabled or not
		streamer.ProcessLoggingOpts{
			Enabled:    true, // Indicates if process logging is enabled
			Compress:   true, // Indicates if logs should be compressed
			Directory:  "/tmp/logs/stream", // Directory to store logs
			MaxAge:     0, // Max age for a log. 0 is infinite
			MaxBackups: 2, // Maximum backups to keep
			MaxSize:    500, // Maximum size of a log in megabytes
		},
		25*time.Second, // Time to wait before declaring a stream start failed
	)
    fmt.Print(id)
	// Returns a waitGroup where the stream checking the underlying process for a successful start
	stream.Start().Wait()
}