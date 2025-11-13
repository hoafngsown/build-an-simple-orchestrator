package logger

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

type ComponentFormatter struct {
	logrus.TextFormatter
}

func (f *ComponentFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// Component name with color and padding
	component := "system"
	if c, ok := entry.Data["component"]; ok {
		component = fmt.Sprintf("%v", c)
	}

	componentDisplay := strings.Title(component)

	// Level with color
	var levelColor int
	var levelText string
	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = 90 // Gray
		levelText = "DEBUG"
	case logrus.InfoLevel:
		levelColor = 36 // Cyan
		levelText = "INFO "
	case logrus.WarnLevel:
		levelColor = 33 // Yellow
		levelText = "WARN "
	case logrus.ErrorLevel:
		levelColor = 31 // Red
		levelText = "ERROR"
	case logrus.FatalLevel:
		levelColor = 35 // Magenta
		levelText = "FATAL"
	default:
		levelColor = 37 // White
		levelText = "     "
	}

	// Write timestamp
	fmt.Fprintf(b, "\x1b[37m%s\x1b[0m ", timestamp)
	// Write component with brackets
	fmt.Fprintf(b, "\x1b[1;34m[%s]\x1b[0m ", componentDisplay)
	// Write level
	fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m ", levelColor, levelText)
	// Write message
	fmt.Fprintf(b, "%s", entry.Message)

	// Write additional fields (except component)
	if len(entry.Data) > 0 {
		b.WriteString(" ")
		first := true
		for k, v := range entry.Data {
			if k == "component" {
				continue
			}
			if !first {
				b.WriteString(" ")
			}
			fmt.Fprintf(b, "\x1b[2m%s\x1b[0m=\x1b[33m%v\x1b[0m", k, v)
			first = false
		}
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func Initialize() {
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.DebugLevel)
	log.SetFormatter(&ComponentFormatter{
		TextFormatter: logrus.TextFormatter{
			ForceColors:   true,
			DisableColors: false,
		},
	})
}

func GetLogger(component string) *logrus.Entry {
	return log.WithField("component", component)
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Warn(args ...interface{}) {
	log.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Fatal(args ...interface{}) {
	log.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

func WithFields(fields map[string]interface{}) *logrus.Entry {
	return log.WithFields(fields)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return log.WithField(key, value)
}
