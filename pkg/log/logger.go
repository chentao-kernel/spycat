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
	keepdays int
	logger   *logrus.Logger
	file     *os.File
}

func (l *Logger) SetLevel(level logrus.Level) {
	Loger.logger.SetLevel(Loger.level)
}

func LevelTransform(level string) logrus.Level {
	switch level {
	case "PANIC":
		return logrus.InfoLevel
	case "FATAL":
		return logrus.FatalLevel
	case "ERROR":
		return logrus.ErrorLevel
	case "WARN":
		return logrus.WarnLevel
	case "INFO":
		return logrus.InfoLevel
	case "DEBUG":
		return logrus.DebugLevel
	case "TRACE":
		return logrus.TraceLevel
	}
	return logrus.InfoLevel
}

func NewLogger(filePath string, level logrus.Level) *Logger {
	if !iner.Exists(filePath) {
		err := os.MkdirAll(filePath, 0755)
		if err != nil {
			log.Fatalf("mkdir %s failed.", filePath)
		}
	}

	fileName := time.Now().Format("20060102_15:04:05") + ".log"
	file, err := os.OpenFile(filePath+"/"+fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("New logger failed:%v", err))
	}

	return &Logger{
		name:     fileName,
		level:    level,
		keepdays: 7,
		logger:   logrus.New(),
		file:     file,
	}
}

func LogInit(filePath string, level logrus.Level) {
	Loger = NewLogger(filePath, level)
	Loger.logger.SetOutput(Loger.file)
	Loger.logger.SetLevel(Loger.level)
	// output file name and function name
	Loger.logger.SetReportCaller(true)
	Loger.logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:03:04",

		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			// handle file name
			fileName := path.Base(frame.File)
			return frame.Function, fileName
		},
	})
}

func (l *Logger) Info(format string, a ...any) {
	l.logger.Info(fmt.Sprintf(format, a...))
}

func (l *Logger) Error(format string, a ...any) {
	l.logger.Error(fmt.Sprintf(format, a...))
}

func (l *Logger) Warn(format string, a ...any) {
	l.logger.Warn(fmt.Sprintf(format, a...))
}

func (l *Logger) Debug(format string, a ...any) {
	l.logger.Debug(fmt.Sprintf(format, a...))
}
