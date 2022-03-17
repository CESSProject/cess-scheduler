package logger

import (
	"fmt"
	"os"
	"scheduler-mining/configs"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	InfoLogger *zap.Logger
	ErrLogger  *zap.Logger
	//WatcherLogger *zap.Logger
)

func LoggerInit() {
	_, err := os.Stat(configs.LogFilePath)
	if err != nil {
		err = os.MkdirAll(configs.LogFilePath, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(-1)
		}
	}
	initInfoLogger()
	initErrLogger()
	//initWatchLogger()
}

// func initWatchLogger() {
// 	infologpath := configs.LogFilePath + "/watcher.log"
// 	hook := lumberjack.Logger{
// 		Filename:   infologpath,
// 		MaxSize:    10,
// 		MaxAge:     180,
// 		MaxBackups: 0,
// 		LocalTime:  true,
// 		Compress:   true,
// 	}
// 	encoderConfig := zapcore.EncoderConfig{
// 		MessageKey:   "msg",
// 		TimeKey:      "time",
// 		CallerKey:    "file",
// 		LineEnding:   zapcore.DefaultLineEnding,
// 		EncodeLevel:  zapcore.LowercaseLevelEncoder,
// 		EncodeTime:   formatEncodeTime,
// 		EncodeCaller: zapcore.ShortCallerEncoder,
// 	}
// 	atomicLevel := zap.NewAtomicLevel()
// 	atomicLevel.SetLevel(zap.InfoLevel)
// 	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
// 	core := zapcore.NewCore(
// 		zapcore.NewJSONEncoder(encoderConfig),
// 		zapcore.NewMultiWriteSyncer(writes...),
// 		atomicLevel,
// 	)
// 	caller := zap.AddCaller()
// 	development := zap.Development()
// 	WatcherLogger = zap.New(core, caller, development)
// 	WatcherLogger.Sugar().Infof("The service has started and created a log file in the %v", infologpath)
// }

// info log
func initInfoLogger() {
	infologpath := configs.LogFilePath + "/info.log"
	hook := lumberjack.Logger{
		Filename:   infologpath,
		MaxSize:    10,
		MaxAge:     360,
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.InfoLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	InfoLogger = zap.New(core, caller, development)
	InfoLogger.Sugar().Infof("The service has started and created a log file in the %v", infologpath)
}

// error log
func initErrLogger() {
	errlogpath := configs.LogFilePath + "/error.log"
	hook := lumberjack.Logger{
		Filename:   errlogpath,
		MaxSize:    10,
		MaxAge:     360,
		MaxBackups: 0,
		LocalTime:  true,
		Compress:   true,
	}
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:   "msg",
		TimeKey:      "time",
		CallerKey:    "file",
		LineEnding:   zapcore.DefaultLineEnding,
		EncodeLevel:  zapcore.LowercaseLevelEncoder,
		EncodeTime:   formatEncodeTime,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zap.ErrorLevel)
	var writes = []zapcore.WriteSyncer{zapcore.AddSync(&hook)}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(writes...),
		atomicLevel,
	)
	caller := zap.AddCaller()
	development := zap.Development()
	ErrLogger = zap.New(core, caller, development)
	ErrLogger.Sugar().Errorf("The service has started and created a log file in the %v", errlogpath)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()))
}
