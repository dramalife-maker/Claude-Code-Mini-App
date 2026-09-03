// Package logging 初始化全域 slog logger：console 格式，同時寫 stdout 與 rotate 檔案。
package logging

import (
	"log/slog"
	"os"
	"strings"

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
// log level 由環境變數 LOG_LEVEL 控制（debug/info/warn/error），預設 info；
// 逐行 protocol trace 之類的細節走 Debug，平常不落地，重現問題時用 LOG_LEVEL=debug 開。
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

	core := zapcore.NewCore(encoder, writer, level())
	handler := zapslog.NewHandler(core)
	slog.SetDefault(slog.New(handler))

	return func() { _ = rotator.Close() }
}

func level() zapcore.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
