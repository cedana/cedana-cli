package utils

import (
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

var once sync.Once

var log zerolog.Logger

func GetLogger() zerolog.Logger {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano

		var level int = 1
		logLevel, ok := os.LookupEnv("CEDANA_LOG_LEVEL")
		if ok {
			val, err := strconv.Atoi(logLevel)
			if err != nil {
				level = 1
			} else {
				level = val
			}
		}

		var output io.Writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}

		fileLogger := &lumberjack.Logger{
			Filename:   "cedana.log",
			MaxSize:    5, //
			MaxBackups: 10,
			MaxAge:     14,
			Compress:   true,
		}

		output = zerolog.MultiLevelWriter(os.Stderr, fileLogger)

		log = zerolog.New(output).
			Level(zerolog.Level(level)).
			With().
			Timestamp().
			Logger().
			Output(zerolog.ConsoleWriter{Out: os.Stdout})
	})

	return log
}

// a separate logger for SSH stdout/stderr,
// for finer control on output.
func CreateSSHLogger() {}
