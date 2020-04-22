package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	flog "github.com/akuan/logrus"
)

//var log *logrus.Logger //= logrus.New()

func init() {
	var textFmt = &flog.TextFormatter{
		//DisableQuoteFields: true,
		TimestampFormat:    "2006-01-02 15:04:05.000",
		//SkipFixFiledName:   true
		}
	flog.SetFormatter(textFmt)
	flog.SetLevel(flog.TraceLevel)
	hook, err := NewHook(
		&LogFile{
			Filename:   "logs/rtsp-general.log",
			MaxSize:    60,
			MaxBackups: 30,
			MaxAge:     30,
			Compress:   false,
			LocalTime:  true,
		},
		flog.TraceLevel,
		textFmt,
		&LogFileOpts{
			flog.InfoLevel: &LogFile{
				Filename: "logs/rtsp-info.log",
			},
			flog.ErrorLevel: &LogFile{
				Filename:   "logs/rtsp-error.log",
				MaxSize:    60,
				MaxBackups: 30,
				MaxAge:     30,
				Compress:   false,
				LocalTime:  true,
			},
		},
	)
	if err != nil {
		panic(err)
	}
	flog.AddHook(hook)
	flog.WithFields(flog.Fields{
		"File": "logfileUtil.go",
		"size": 10,
	}).Debug("logrus.Debug Start...")
	PrintObj("Now the Hook is", hook)
	//logrus.Flush()
}

/*
func GetLogger() *logrus.Logger {
	return log
}
*/
func PrintObj(msg string, obj interface{}) {
	js, _ := json.Marshal(obj)
	flog.Debugf("%s:%s", msg, string(js))
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//CheckOrCreatePath 判断目录是否存在，不存在则创建目录
func CheckOrCreatePath(_dir string) bool {
	exist, err := PathExists(_dir)
	if err != nil {
		fmt.Printf("ERROR get dir %v error![%v]", _dir, err)
		return false
	}
	if !exist {
		fmt.Printf("ERROR no dir![%v]", _dir)
		err := os.Mkdir(_dir, os.ModePerm)
		if err != nil {
			fmt.Printf("ERROR mkdir failed![%v]", err)
			return false
		} else {
			fmt.Printf("mkdir success!")
		}
	}
	return true
}

func InitLog() *flog.Logger {
	/*
		logrus.Debug("InitLog")
		log = logrus.StandardLogger()

			log.Level = logrus.TraceLevel
			log.SetLevel(logrus.TraceLevel)
			log.SetReportCaller(true)
			ct := time.Now().Local()
			dyStr := ct.Format("2006-01-02")
			lr := "logs"
			if !CheckOrCreatePath(lr) {
				lr = "."
			}
			logName := lr + "/service-" + dyStr + ".log"
			logfile, _ := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			log.Out = logfile
			cleanOutDateLog()

		return log
	*/
	return nil
}

func cleanOutDateLog() {
	lr := "logs"
	str := fmt.Sprintf("%dh", -90*24) //90天前
	d, _ := time.ParseDuration(str)
	nityDays := time.Now().Add(d)
	err := filepath.Walk(lr, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if f.ModTime().Before(nityDays) {
			fmt.Printf("remove %v \n", "./"+lr+"/"+f.Name())
			flog.Infof("remove %v ", "./"+lr+"/"+f.Name())
			err := os.Remove("./" + lr + "/" + f.Name())
			if err != nil {
				flog.Errorf("Remove  Error %v ", err)
			}
		}
		return nil
	})
	if err != nil {
		logrus.Printf("filepath.Walk() returned %v", err)
	}
}

func badCmdMethod() {
	strCmd := `/p logs /s /m *.log /d -1 /c "cmd /c del @file" `
	cmd := exec.Command("forfiles", strCmd)
	logrus.Infof("Start Clean...%v ", strCmd)
	//cmd.Stdin = strings.NewReader(strCmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Clean Log fail: %v,%v ", err, out)
	}
}

