package main

import (
	"fmt"
	"strings"

	"github.com/xigxog/kubefox/libs/core/kit"
)

func main() {
	k := kit.New()
	k.Route("Path(`/hello`)", sayHello)
	k.Start()
}

func sayHello(k kit.Kontext) error {
	r, err := k.Component("world").Send()
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("👋 Hello %s!", r.String())
	k.Log().Info(msg)

	accVal := strings.ToLower(k.Header("accept"))
	switch {
	case strings.Contains(accVal, "application/json"):
		return k.Resp().SendJSON(map[string]any{"msg": msg})

	case strings.Contains(accVal, "text/html"):
		return k.Resp().SendHTML(fmt.Sprintf(html, msg))

	default:
		return k.Resp().SendString(msg)
	}
}

const html = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>Hello KubeFox</title>
    <style>
      html,
      body,
      p {
        height: 100%%;
        margin: 0;
      }
      .container {
        display: flex;
        flex-direction: column;
        min-height: 80%%;
        align-items: center;
        justify-content: center;
      }
    </style>
  </head>
  <body>
    <main class="container">
      <h1>%s</h1>
    </main>
  </body>
</html>
`
