package log

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xigxog/kubefox/libs/core/logkf"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/yaml"
)

var (
	log    *logkf.Logger = logkf.BuildLoggerOrDie("cli", "debug")
	outFmt string        = "json"
)

func Setup(fmt string, verbose bool) {
	if !verbose {
		log = log.IncreaseLevel(zapcore.InfoLevel)
	}
	outFmt = fmt
}

func Logger() *logkf.Logger {
	return log
}

func Printf(format string, v ...any) {
	fmt.Printf(format, v...)
}

func Marshal(o any) {
	fmt.Print(marshal(o))
}

func Newline() {
	fmt.Fprintln(os.Stderr)
}

func Info(format string, v ...any) {
	log.Infof(format, v...)
}

func InfoMarshal(o any, format string, v ...any) {
	format = format + "\n%s"
	out, _ := yaml.Marshal(o)
	v = append(v, out)
	log.Infof(format, v...)
}

func Verbose(format string, v ...any) {
	log.Debugf(format, v...)
}

func VerboseMarshal(o any, format string, v ...any) {
	format = format + "\n%s"
	out, _ := yaml.Marshal(o)
	v = append(v, out)
	log.Debugf(format, v...)
}

func Error(format string, v ...any) {
	log.Errorf(format, v...)
}

func Fatal(format string, v ...any) {
	format = "😖 " + format
	log.Errorf(format, v...)
	os.Exit(1)
}

func marshal(o any) string {
	var output []byte
	var err error
	if outFmt == "yaml" {
		output, err = yaml.Marshal(o)
	} else {
		output, err = json.MarshalIndent(o, "", "  ")
	}
	if err != nil {
		Fatal("Error marshaling response: %v", err)
	}

	return string(output)
}
