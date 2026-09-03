// Package logging 初始化全域 slog logger：console 格式，同時寫 stdout 與 rotate 檔案。
package logging

import (
	"log/slog"
	"os"

	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logDir         = "logs"
	logFile        = logDir + "/server.log"
	rotateMaxMB    = 100
	rotateBackups  = 7
	rotateMaxDays  = 14
	rotateCompress = true
)

// Init 設定 slog 的 default logger，回傳 sync 函式，main 應 defer 呼叫。
func Init() func() {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		panic("logging: 建立 " + logDir + " 失敗: " + err.Error())
	}

	rotator := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    rotateMaxMB,
		MaxBackups: rotateBackups,
		MaxAge:     rotateMaxDays,
		Compress:   rotateCompress,
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	encoder := zapcore.NewConsoleEncoder(encoderCfg)

	writer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(rotator),
	)

	core := zapcore.NewCore(encoder, writer, zapcore.InfoLevel)
	handler := zapslog.NewHandler(core)
	slog.SetDefault(slog.New(handler))

	return func() { _ = rotator.Close() }
}
