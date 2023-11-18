package log

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"time"

	iner "github.com/chentao-kernel/spycat/internal"
	"github.com/sirupsen/logrus"
)

var Loger *Logger

const (
	PATH = "/tmp/spycat"
)

type Logger struct {
	name     string
	level    logrus.Level
	keeydays int
	logger   *logrus.Logger
	file     *os.File
}

func NewLogger() *Logger {
	if !iner.Exists(PATH) {
		err := os.MkdirAll(PATH, 0755)
		if err != nil {
			log.Fatalf("mkdir %s failed.", PATH)
		}
	}

	fileName := time.Now().Format("20060102_15:04:05") + ".log"
	file, err := os.OpenFile(PATH+"/"+fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("New logger failed:%v", err))
	}

	return &Logger{
		name:     fileName,
		level:    logrus.InfoLevel,
		keeydays: 7,
		logger:   logrus.New(),
		file:     file,
	}
}

func init() {
	Loger = NewLogger()
	Loger.logger.SetOutput(Loger.file)
	Loger.logger.SetLevel(Loger.level)
	// 输出文件名及函数名
	Loger.logger.SetReportCaller(true)
	Loger.logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:03:04",

		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			//处理文件名
			fileName := path.Base(frame.File)
			return frame.Function, fileName
		},
	})
}

func (log *Logger) Info(format string, a ...any) {
	log.logger.Info(fmt.Sprintf(format, a...))
}

func (log *Logger) Error(format string, a ...any) {
	log.logger.Error(fmt.Sprintf(format, a...))
}

func (log *Logger) Warn(format string, a ...any) {
	log.logger.Warn(fmt.Sprintf(format, a...))
}

func (log *Logger) Debug(format string, a ...any) {
	log.logger.Debug(fmt.Sprintf(format, a...))
}
