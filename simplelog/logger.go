package simplelog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

type Logger struct {
	Slogger       *slog.Logger
	LogLevel      slog.Level
	LogMode       LogMode
	LogHandler    LogHandler
	UseGcpLogging bool
}

// メッセージ、ラベル、トレースを設定するためのインターフェース
type LogHandler interface {
	GetMessage(logger Logger, context context.Context, level string, file string, line int, originalOutput string) string
	GetLabels(logger Logger, context context.Context) map[string]string
	GetTrace(logger Logger, context context.Context) string
}

type LogLevel int

const (
	LOG_LEVEL_INFO LogLevel = iota
	LOG_LEVEL_DEBUG
)

type LogMode int

const (
	LOG_MODE_SLOGGER LogMode = iota
	// slogではなくfmtによって出力する。
	LOG_MODE_FMT
)

type logPrintLevel string

const (
	PRINT_DEBG  logPrintLevel = "DEBG"
	PRINT_INFO  logPrintLevel = "INFO"
	PRINT_WARN  logPrintLevel = "WARN"
	PRINT_ERROR logPrintLevel = "ERROR"
)

func New(logLevel LogLevel, logMode LogMode, logHandler LogHandler, useGcpLogging bool) Logger {
	l := Logger{}
	l.LogLevel = slog.LevelInfo
	if logLevel >= LOG_LEVEL_DEBUG {
		l.LogLevel = slog.LevelDebug
	}
	l.LogMode = logMode
	l.LogHandler = logHandler
	l.UseGcpLogging = useGcpLogging

	// slogをCloud Loggingで必要な形式にカスタマイズする。
	// https://cloud.google.com/logging/docs/structured-logging?hl=ja
	replacer := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			a.Key = "timestamp"
		}
		if a.Key == slog.MessageKey {
			a.Key = "message"
		}
		if a.Key == slog.LevelKey {
			a.Key = "severity"
		}
		return a
	}
	l.Slogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: replacer,
		Level:       l.LogLevel,
		// AddSource: true,
		// AddSource は使わずにruntime.Callerを使っている。
		// AddSourceは呼び出し元ではなく本ファイルおよびその行数が出力されてしまうため。
	}))
	return l
}

func (l Logger) l(context context.Context, level logPrintLevel, format string, skip int, args ...any) {
	// ファイル名と行数の情報を設定
	_, file, line, ok := runtime.Caller(skip)
	var callerAttr slog.Attr
	if ok {
		// https://cloud.google.com/logging/docs/agent/logging/configuration?hl=ja
		callerAttr = slog.Group("logging.googleapis.com/sourceLocation", slog.String("file", file), slog.Int("line", line))
	}

	// 「logging.googleapis.com/trace」はCloud Traceへの連携
	// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#FIELDS.trace
	var traceAttr slog.Attr
	if l.LogHandler != nil {
		traceAttr = slog.String("logging.googleapis.com/trace", l.LogHandler.GetTrace(l, context))
	}

	// ラベルの設定
	labels := []any{}
	if l.LogHandler != nil {
		labelMap := l.LogHandler.GetLabels(l, context)
		for k, v := range labelMap {
			labels = append(labels, slog.String(k, v))
		}
	}
	labelsAttr := slog.Group("logging.googleapis.com/labels", labels...)

	// 出力メッセージの設定
	out := fmt.Sprintf(format, args...)
	if l.LogHandler != nil {
		out = l.LogHandler.GetMessage(l, context, string(level), file, line, out)
	}

	if l.LogMode == LOG_MODE_FMT {
		fmt.Println(out)
		return
	}

	switch level {
	case PRINT_DEBG:
		if l.UseGcpLogging {
			l.Slogger.Debug(out, callerAttr, traceAttr, labelsAttr)
		} else {
			l.Slogger.Debug(out)
		}
	case PRINT_INFO:
		if l.UseGcpLogging {
			l.Slogger.Info(out, callerAttr, traceAttr, labelsAttr)
		} else {
			l.Slogger.Info(out)
		}
	case PRINT_WARN:
		if l.UseGcpLogging {
			l.Slogger.Warn(out, callerAttr, traceAttr, labelsAttr)
		} else {
			l.Slogger.Warn(out)
		}
	case PRINT_ERROR:
		if l.UseGcpLogging {
			l.Slogger.Error(out, slog.String("error", out), callerAttr, traceAttr, labelsAttr)
		} else {
			l.Slogger.Error(out)
		}
	}
}

func (l Logger) Debug(context context.Context, args ...any) {
	l.lArgs(context, PRINT_DEBG, 3, args...)
}

// コンテキストを指定せずにデバッグ出力
func (l Logger) D(args ...any) {
	l.lArgs(context.Background(), PRINT_DEBG, 3, args...)
}

// 呼び出し元のskip数を指定
func (l Logger) DebugWithSkip(context context.Context, skip int, args ...any) {
	l.lArgs(context, PRINT_DEBG, 3+skip, args...)
}

func (l Logger) Debugf(context context.Context, format string, args ...any) {
	l.l(context, PRINT_DEBG, format, 2, args...)
}

// コンテキストを指定せずにデバッグ出力（フォーマット指定）
func (l Logger) DF(format string, args ...any) {
	l.l(context.Background(), PRINT_DEBG, format, 2, args...)
}

// 呼び出し元のskip数を指定
func (l Logger) DebugfWithSkip(context context.Context, skip int, format string, args ...any) {
	l.l(context, PRINT_DEBG, format, 2+skip, args...)
}

// コンテキストを指定せずにデバッグ出力（JSONに変換して出力）
func (l Logger) DJ(a any) {
	jj, _ := json.MarshalIndent(&a, "", "    ")
	l.l(context.Background(), PRINT_DEBG, "\n%s", 2, string(jj))
}

func (l Logger) Info(context context.Context, args ...any) {
	l.lArgs(context, PRINT_INFO, 3, args...)
}

func (l Logger) Infof(context context.Context, format string, args ...any) {
	l.l(context, PRINT_INFO, format, 2, args...)
}

func (l Logger) Warn(context context.Context, args ...any) {
	l.lArgs(context, PRINT_WARN, 3, args...)
}

func (l Logger) Warnf(context context.Context, format string, args ...any) {
	l.l(context, PRINT_WARN, format, 2, args...)
}

func (l Logger) Error(context context.Context, args ...any) {
	l.lArgs(context, PRINT_ERROR, 3, args...)
}

func (l Logger) Errorf(context context.Context, format string, args ...any) {
	l.l(context, PRINT_ERROR, format, 2, args...)
}

func (l Logger) lArgs(context context.Context, printLevel logPrintLevel, skip int, args ...any) {
	format := ""
	for i := range args {
		if i != 0 {
			format += " "
		}
		format += "%+v"
		// if i == len(args)-1 {
		// 	format += "\n"
		// }
	}
	l.l(context, printLevel, format, skip, args...)
}
