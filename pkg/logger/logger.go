/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package logger

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Log(string, string, error)
}

type logs struct {
	logpath map[string]string
	log     map[string]*zap.Logger
}

func NewLogs(logfiles map[string]string) (Logger, error) {
	var (
		logpath = make(map[string]string, 0)
		logCli  = make(map[string]*zap.Logger)
	)
	for name, fpath := range logfiles {
		dir := getFilePath(fpath)
		_, err := os.Stat(dir)
		if err != nil {
			err = os.MkdirAll(dir, os.ModeDir)
			if err != nil {
				return nil, err
			}
		}
		Encoder := getEncoder()
		newCore := zapcore.NewTee(
			zapcore.NewCore(Encoder, getWriteSyncer(fpath), zap.NewAtomicLevel()),
		)
		logpath[name] = fpath
		logCli[name] = zap.New(newCore, zap.AddCaller())
		logCli[name].Sugar().Infof("%v", fpath)
	}
	return &logs{
		logpath: logpath,
		log:     logCli,
	}, nil
}

func (l *logs) Log(name, level string, err error) {
	v, ok := l.log[name]
	if ok {
		switch level {
		case "info":
			v.Sugar().Infof("%v", err)
		case "error":
			v.Sugar().Errorf("%v", err)
		case "warn":
			v.Sugar().Warnf("%v", err)
		}
	}
}

func getFilePath(fpath string) string {
	file, _ := exec.LookPath(fpath)
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))
	ret := path[:index]
	return ret
}

func getEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(
		zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller_line",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    cEncodeLevel,
			EncodeTime:     cEncodeTime,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   cEncodeCaller,
		})
}

func getWriteSyncer(fpath string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fpath,
		MaxSize:    10,
		MaxBackups: 99,
		MaxAge:     180,
		LocalTime:  true,
		Compress:   true,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func cEncodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func cEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + t.Format("2006-01-02 15:04:05") + "]")
}

func cEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + caller.TrimmedPath() + "]")
}
