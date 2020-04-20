package main

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"log"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"rtsp-stream/core"
	"rtsp-stream/core/config"
)

func main() {
	config := config.InitConfig()
	//config.Debug=true
	config.KeepFiles=false
	//config.Port=9580
	var logDir = "./logs"
	//os.MkdirAll(logDir, os.ModePerm)
	config.ProcessLogging.Enabled=true
	config.ProcessLogging.Directory=logDir;
	config.Audio=true
	config.JWTEnabled=false
	core.SetupLogger(config)
	router := httprouter.New()
	fileServer := http.FileServer(http.Dir(config.StoreDir))
	controllers := core.NewController(config, fileServer)
	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
	})
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
	//clear all viedos
	clearTempFiles(config.StoreDir)
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
//clearTempFiles
func clearTempFiles(root string){
	defer func() {
		//产生了panic异常
		if err := recover(); err != nil {
			logrus.Errorf("clearTempFiles Error: %v", err)
		}
	}()
	//logrus.Debugf("clearTempFiles root is %s",root)
   dir,err:=os.Open(root)
   defer dir.Close()
   if(err!=nil){
	   logrus.Errorf("clearTempFiles Error: %v", err)
	   return
   }
	children,err2:=dir.Readdir(0)
	if err2!=nil{
		logrus.Errorf("clearTempFiles Readdir Error: %v", err)
		return
	}
	for _,ch :=range children{
		if(ch.IsDir()){
			fullpath:=fmt.Sprintf("%s/%s",root,ch.Name())
			logrus.Debugf("find %s will remove",fullpath)
			os.RemoveAll(fullpath)
		}
	}
}