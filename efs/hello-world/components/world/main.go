package main

import "github.com/xigxog/kubefox/libs/core/kit"

func main() {
	kit := kit.New()
	kit.Start()

	// kitSvc := kubefox.New()

	// kitSvc.DefaultEntrypoint(func(kit kubefox.Kit) (err error) {
	// 	who := kit.Env("who").String()
	// 	kit.Log().Infof("Letting hello know to say hello to %s!", who)
	// 	kit.Response().SetContent([]byte(who))
	// 	return
	// })

	// kitSvc.Start()
}
