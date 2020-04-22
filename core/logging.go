package core

import (
	"os"
	"rtsp-stream/core/config"
	flog "github.com/akuan/logrus"
)

// SetupLogger sets the logger for the proper settings based on the environment
func SetupLogger(spec *config.Specification) {
	flog.SetOutput(os.Stdout)
	if spec.Debug {
		flog.SetLevel(flog.DebugLevel)
		return
	}
	flog.SetLevel(flog.InfoLevel)
}
