package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Roverr/rtsp-stream/core/config"
	"github.com/Roverr/rtsp-stream/core/streaming"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// ErrUnexpected describes an unexpected error
var ErrUnexpected = errors.New("Unexpected error")

// ErrDirectoryNotCreated is sent when the system cannot create the directory for the URI
var ErrDirectoryNotCreated = errors.New("Could not create directory for URI")

// ErrTimeout describes an error related to timing out
var ErrTimeout = errors.New("Timeout error")

// ErrDTO describes a DTO that has a message as an error
type ErrDTO struct {
	Error string `json:"error"`
}

// Controller holds all handler functions for the API
type Controller struct {
	spec       *config.Specification
	streams    map[string]*streaming.Stream
	fileServer http.Handler
	manager    IManager
	processor  streaming.IProcessor
	timeout    time.Duration
}

// NewController creates a new instance of Controller
func NewController(spec *config.Specification, fileServer http.Handler) *Controller {
	return &Controller{spec, map[string]*streaming.Stream{}, fileServer, Manager{}, streaming.NewProcessor(spec.StoreDir), time.Second * 15}
}

// SendError sends an error to the client
func (c *Controller) SendError(w http.ResponseWriter, err error, status int) {
	w.Header().Add("Content-Type", "application/json")
	b, _ := json.Marshal(ErrDTO{Error: err.Error()})
	w.WriteHeader(status)
	w.Write(b)
}

// ListStreamHandler is the HTTP handler of the /list call
func (c *Controller) ListStreamHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	dto := []*summariseDto{}
	for key, stream := range c.streams {
		dto = append(dto, &summariseDto{URI: fmt.Sprintf("/stream/%s/index.m3u8", key), Running: stream.Streak.IsActive()})
	}
	b, err := json.Marshal(dto)
	if err != nil {
		c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

// StartStreamHandler is an HTTP handler for the /start endpoint
func (c *Controller) StartStreamHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var dto streamDto
	if err := validateURI(&dto, r.Body); err != nil {
		logrus.Error(err)
		c.SendError(w, err, http.StatusBadRequest)
		return
	}
	// Calculate directory from URI
	dir, err := streaming.GetURIDirectory(dto.URI)
	if err != nil {
		logrus.Error(err)
		c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
		return
	}
	if stream, ok := c.streams[dir]; ok {
		c.handleAlreadyRunningStream(w, stream, c.spec, dir)
		return
	}
	streamResolved := c.startStream(dto.URI, dir, c.spec)
	defer close(streamResolved)
	select {
	case <-time.After(c.timeout):
		c.SendError(w, ErrTimeout, http.StatusRequestTimeout)
		return
	case success := <-streamResolved:
		if !success {
			c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
			return
		}
		s := c.streams[dir]
		b, err := json.Marshal(streamDto{URI: s.Path})
		if err != nil {
			c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)
	}
}

// ExitHandler is a function that can recognise when the application is being closed
// and cleans up all background running processes
func (c *Controller) ExitHandler() chan bool {
	done := make(chan bool)
	ch := make(chan os.Signal, 3)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-ch
		c.cleanUp()
		os.Exit(0)
		done <- true
	}()
	return done
}

// handleAlreadyRunningStream is for dealing with stream starts that are already initiated before
func (c *Controller) handleAlreadyRunningStream(w http.ResponseWriter, strm *streaming.Stream, spec *config.Specification, dir string) {
	// If transcoding is not running, spin it back up
	if !strm.Streak.IsActive() {
		err := c.processor.Restart(strm, dir)
		if err != nil {
			logrus.Error(err)
			c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
			return
		}
	}
	// If the stream is already running return its path
	b, err := json.Marshal(streamDto{URI: strm.Path})
	if err != nil {
		logrus.Error(err)
		c.SendError(w, ErrUnexpected, http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

// cleanUp stops all running processes
func (c *Controller) cleanUp() {
	for uri, strm := range c.streams {
		logrus.Debugf("Closing processing of %s", uri)
		if err := strm.CleanProcess(); err != nil {
			logrus.Debugf("Could not close %s", uri)
			logrus.Error(err)
			return
		}
		logrus.Debugf("Succesfully closed processing for %s", uri)
	}
}

// cleanUnused is for stopping all transcoding for streams that are not watched anymore
func (c *Controller) cleanUnused() {
	for name, data := range c.streams {
		// If the streak is active, there is no need for stopping
		if data.Streak.IsActive() {
			logrus.Debugf("%s is active, skipping cleaning process", name)
			continue
		}
		logrus.Infof("%s is getting cleaned", name)
		data.Mux.Lock()
		defer data.Mux.Unlock()
		if err := data.CMD.Process.Kill(); err != nil {
			if strings.Contains(err.Error(), "process already finished") {
				logrus.Infof("\n%s is cleaned", name)
				continue
			}
			logrus.Error(err)
		}
		logrus.Infof("\n%s is cleaned", name)
	}
}

// FileHandler is HTTP handler for direct file requests
func (c *Controller) FileHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	defer c.fileServer.ServeHTTP(w, req)
	filepath := ps.ByName("filepath")
	req.URL.Path = filepath
	hostKey := determineHost(filepath)
	s, ok := c.streams[hostKey]
	if !ok {
		return
	}
	if s.Streak.IsActive() {
		s.Streak.Hit()
		return
	}
	if err := c.processor.Restart(s, hostKey); err != nil {
		logrus.Error(err)
		return
	}
	s.Streak.Activate().Hit()
}

func (c *Controller) startStream(uri, dir string, spec *config.Specification) chan bool {
	logrus.Infof("%s started processing", dir)
	stream, physicalPath := c.processor.NewStream(uri)
	c.streams[dir] = stream
	ch := c.manager.Start(stream.CMD, physicalPath)
	return ch
}
