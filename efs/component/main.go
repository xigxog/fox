package main

import "github.com/xigxog/kubefox/libs/core/kit"

func main() {
	k := kit.New()

	// TODO Add component routes and logic!
	k.Route("Path(`/hello`)", func(ktx kit.Kontext) error {
		return ktx.Resp().SendString("Hello 👋")
	})

	k.Start()
}
