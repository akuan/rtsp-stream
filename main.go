package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"

	"rtsp-stream/core"
	"rtsp-stream/core/config"
	"github.com/sirupsen/logrus"
)
var staticHandler http.Handler

func StaticServer(w http.ResponseWriter, req *http.Request,_ httprouter.Params) {
	logrus.Infof("StaticServer request path %s \n",req.URL.Path)
	if req.URL.Path != "/" {
		fmt.Printf("request path %s",req.URL.Path)
		logrus.Infof("request path %s \n",req.URL.Path)
		staticHandler.ServeHTTP(w, req)
		return
	}
	io.WriteString(w, "hello, world!\n")
}
func main() {
	config := config.InitConfig()
	//config.Debug=true
	config.KeepFiles=false
	//config.Port=9580
	config.Audio=true
	core.SetupLogger(config)
	fileServer := http.FileServer(http.Dir(config.StoreDir))
	router := httprouter.New()
	controllers := core.NewController(config, fileServer)
	var dir = path.Dir("D:\\cell\\incubator\\rtsp-stream")
	staticHandler = http.FileServer(http.Dir(dir))
	router.GET("/files/",  StaticServer)
	//router.GET("/html",  StaticServer)
	//router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//	w.WriteHeader(http.StatusOK)
	//})
	router.GET("/",  func(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
		logrus.Infoln("files,path=",r.URL.Path)
		var file=r.URL.Path;
		file=strings.TrimPrefix(file,"/")
		http.ServeFile(w, r, file)
	})
	//router.GET("/html",  StaticServer)
	//router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//	w.WriteHeader(http.StatusOK)
	//})
	if config.EndpointYML.Endpoints.List.Enabled {
		router.GET("/list", controllers.ListStreamHandler)
		logrus.Infoln("list endpoint enabled | MainProcess")
	}
	if config.EndpointYML.Endpoints.Start.Enabled {
		router.POST("/start", controllers.StartStreamHandler)
		logrus.Infoln("start endpoint enabled | MainProcess")
	}
	if config.EndpointYML.Endpoints.Static.Enabled {
		router.GET("/stream/*filepath", controllers.StaticFileHandler)
		logrus.Infoln("static endpoint enabled | MainProcess")
	}
	if config.EndpointYML.Endpoints.Stop.Enabled {
		router.POST("/stop", controllers.StopStreamHandler)
		logrus.Infoln("stop endpoint enabled | MainProcess")
	}
	router.GET("/test", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
		logrus.Infoln("test,path=",r.URL.Path)
		http.ServeFile(w, r, "test.html")
	})
	router.GET("/test2", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
		logrus.Infoln("test,path=",r.URL.Path)
		http.ServeFile(w, r, "test2.html")
	})
	router.GET("/hls", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
		logrus.Infoln("test,path=",r.URL.Path)
		http.ServeFile(w, r, "hls.html")
	})
	router.GET("/hls.js", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params){
		logrus.Infoln("hls.js,path=",r.URL.Path)
		http.ServeFile(w, r, "hls.js")
	})
	done := controllers.ExitPreHook()
	handler := cors.AllowAll().Handler(router)
	if config.CORS.Enabled {
		handler = cors.New(cors.Options{
			AllowCredentials: config.CORS.AllowCredentials,
			AllowedOrigins:   config.CORS.AllowedOrigins,
			MaxAge:           config.CORS.MaxAge,
		}).Handler(router)
	}
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: handler,
	}
	go func() {
		logrus.Infof("rtsp-stream transcoder started on %d | MainProcess", config.Port)
		log.Fatal(srv.ListenAndServe())
	}()
	<-done
	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("HTTP server Shutdown: %v", err)
	}
	os.Exit(0)
}
