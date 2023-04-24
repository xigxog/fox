package log

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xigxog/kubefox/libs/core/admin"
	"github.com/xigxog/kubefox/libs/core/logger"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"
)

var (
	log    *logger.Log = logger.CLILogger().Named("fox")
	outFmt string      = "json"
)

func Setup(fmt string, verbose bool) {
	if !verbose {
		log = log.IncreaseLevel(zap.InfoLevel)
	}
	outFmt = fmt
}

func Logger() *logger.Log {
	return log
}

func Print(format string, v ...any) {
	fmt.Println(fmt.Sprintf(format, v...))
}

func Marshal(o any) {
	fmt.Println(marshal(o))
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

func VerboseResp(resp *admin.Response, err error) {
	printResp(resp, err, true)
}
func Resp(resp *admin.Response, err error) {
	printResp(resp, err, false)
}

func printResp(resp *admin.Response, err error, verboseOnly bool) {
	if err != nil {
		resp = &admin.Response{
			IsError: true,
			Msg:     err.Error(),
		}
	}
	if resp == nil {
		Fatal("Invalid response received")
	}

	VerboseMarshal(resp, "Response from KubeFox Admin API")

	if !verboseOnly {
		if resp.IsError {
			Marshal(resp)
		} else if resp.Data != nil {
			Marshal(resp.Data)
		}
	}

	if resp.IsError {
		Fatal("Command resulted in error: %s", resp.Msg)
	}
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
