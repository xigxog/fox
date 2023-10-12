package main

import "github.com/xigxog/kubefox/libs/core/kit"

const html = `
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <title>Hello from KubeFox</title>
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
      .context {
        font-family: monospace;
        color: gray;
      }
    </style>
  </head>
  <body>
    <main class="container">
      <h1>%s</h1>
      <div class="context">
        <p>sys: %s</p>
        <p>env: %s</p>
      </div>
    </main>
  </body>
</html>
`

func main() {
	kit := kit.New()
	kit.Start()

	// kitSvc := kubefox.New()

	// kitSvc.DefaultEntrypoint(func(kit kubefox.Kit) (err error) {
	// 	resp, err := kit.Component("world").Invoke(kubefox.EmptyDataEvent())
	// 	if err != nil {
	// 		return err
	// 	}

	// 	who := resp.GetContent()
	// 	msg := fmt.Sprintf("👋 Hello %s!", who)
	// 	env := kit.Request().GetContext().GetEnvironment()
	// 	sys := kit.Request().GetContext().GetSystem()

	// 	acc := strings.ToLower(kit.Request().HTTP().GetHeader("accept"))
	// 	if strings.Contains(acc, "text/html") {
	// 		kit.Response().SetContentType("text/html; charset=utf-8")
	// 		msg = fmt.Sprintf(html, msg, env, sys)
	// 	} else {
	// 		kit.Response().SetContentType("text/plain; charset=utf-8")
	// 	}

	// 	kit.Log().Infof("Saying hello to %s...", who)
	// 	kit.Response().SetContent([]byte(msg))

	// 	return
	// })

	// kitSvc.Start()
}
