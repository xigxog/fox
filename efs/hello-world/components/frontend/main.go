package main

import (
	"fmt"

	"github.com/xigxog/kubefox/kit"
)

var (
	backend kit.ComponentDep
)

func main() {
	k := kit.New()

	backend = k.Component("backend")
	k.Route("Path(`/{{.Vars.subPath}}/hello`)", sayHello)

	k.Start()
}

func sayHello(k kit.Kontext) error {
	r, err := k.Req(backend).Send()
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("👋 Hello %s!", r.Str())
	json := map[string]any{"msg": msg}
	html := fmt.Sprintf(htmlTmpl, msg)
	k.Log().Debug(msg)

	return k.Resp().SendAccepts(json, html, msg)
}

const htmlTmpl = `
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
