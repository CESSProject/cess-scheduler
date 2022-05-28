package logger

import (
	"cess-scheduler/configs"
	"fmt"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Out  *zap.Logger
	Err  *zap.Logger
	Uld  *zap.Logger
	Tvp  *zap.Logger
	Trf  *zap.Logger
	Tsmi *zap.Logger
	Gpnc *zap.Logger
)

func LoggerInit() {
	_, err := os.Stat(configs.LogFileDir)
	if err != nil {
		err = os.MkdirAll(configs.LogFileDir, os.ModeDir)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			os.Exit(1)
		}
	}
	initOutLogger()
	initErrLogger()
	initTvpLogger()
	initTrfLogger()
	initTsmiLogger()
	initGpncLogger()
	initUldLogger()
}

// out log
func initOutLogger() {
	outlogpath := configs.LogFileDir + "/out.log"
	hook := lumberjack.Logger{
		Filename:   outlogpath,
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
	Out = zap.New(core, caller, development)
	Out.Sugar().Infof("The service has started and created a log file in the %v", outlogpath)
}

// err log
func initErrLogger() {
	errlogpath := configs.LogFileDir + "/err.log"
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
	Err = zap.New(core, caller, development)
	Err.Sugar().Errorf("The service has started and created a log file in the %v", errlogpath)
}

// tvp log
func initTvpLogger() {
	tvplogpath := configs.LogFileDir + "/t_vp.log"
	hook := lumberjack.Logger{
		Filename:   tvplogpath,
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
	Tvp = zap.New(core, caller, development)
	Tvp.Sugar().Infof("The service has started and created a log file in the %v", tvplogpath)
}

// trf log
func initTrfLogger() {
	trflogpath := configs.LogFileDir + "/t_rf.log"
	hook := lumberjack.Logger{
		Filename:   trflogpath,
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
	Trf = zap.New(core, caller, development)
	Trf.Sugar().Infof("The service has started and created a log file in the %v", trflogpath)
}

// tsmi log
func initTsmiLogger() {
	tsmilogpath := configs.LogFileDir + "/t_smi.log"
	hook := lumberjack.Logger{
		Filename:   tsmilogpath,
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
	Tsmi = zap.New(core, caller, development)
	Tsmi.Sugar().Infof("The service has started and created a log file in the %v", tsmilogpath)
}

// gpnc log
func initGpncLogger() {
	gpnclogpath := configs.LogFileDir + "/g_pnc.log"
	hook := lumberjack.Logger{
		Filename:   gpnclogpath,
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
	Gpnc = zap.New(core, caller, development)
	Gpnc.Sugar().Infof("The service has started and created a log file in the %v", gpnclogpath)
}

// uld log
func initUldLogger() {
	uldlogpath := configs.LogFileDir + "/uld.log"
	hook := lumberjack.Logger{
		Filename:   uldlogpath,
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
	Uld = zap.New(core, caller, development)
	Uld.Sugar().Infof("The service has started and created a log file in the %v", uldlogpath)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second()))
}
