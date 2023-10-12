package log

import (
	"io"
	"strings"

	"github.com/buildpacks/pack/pkg/logging"
	"github.com/xigxog/kubefox/libs/core/logkf"
)

func NewPackLogger() logging.Logger {
	l := Logger().Named("buildpack")
	return &packLogger{
		log: l,
		writer: &writer{
			log: l,
		},
	}
}

type packLogger struct {
	log    *logkf.Logger
	writer io.Writer
}

// Something in Buildpack closes the io.Writer of the logger, thus closing
// os.Stderr and preventing any further log messages from being displayed. This
// prevents that by hiding the underlying writer. Also cleans up all the extra
// newlines and spacing.
type writer struct {
	log *logkf.Logger
}

// Removes all extra spaces before writing.
func (w *writer) Write(p []byte) (n int, err error) {
	out := string(p)
	w.log.Debugf(strings.Join(strings.Fields(out), " "))
	return len(p), nil
}

func (l *packLogger) Debug(msg string) {
	l.log.Debugf(msg)
}

func (l *packLogger) Debugf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *packLogger) Info(msg string) {
	l.log.Debugf(msg)
}

func (l *packLogger) Infof(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *packLogger) Warn(msg string) {
	l.log.Debugf(msg)
}

func (l *packLogger) Warnf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *packLogger) Error(msg string) {
	l.log.Debugf(msg)
}

func (l *packLogger) Errorf(format string, v ...interface{}) {
	l.log.Debugf(format, v...)
}

func (l *packLogger) Writer() io.Writer {
	return l.writer
}

func (l *packLogger) IsVerbose() bool {
	return false
}
