package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	hook "github.com/robotn/gohook"
)

func main() {
	keyBindingsArg := flag.String("keyBindings", "", "Key bindings")
	flag.Parse()

	if *keyBindingsArg == "" {
		fmt.Println("Usage: ptt.exe -keyBindings <keyBindings>")
		return
	}

	keyBindings := strings.Split(*keyBindingsArg, ",")
	now := time.Now()
	hook.Register(hook.KeyDown, keyBindings, func(e hook.Event) {
		fmt.Println(e)
	})
	hook.Register(hook.MouseDown, keyBindings, func(e hook.Event) {
		fmt.Println(e)
	})
	timepassed := time.Since(now)
	fmt.Printf("hook.Register: %v\n", timepassed)

	hook.SetDebugLevel(hook.Silent)
	fmt.Println("starting hook")
	s := hook.Start()
	defer hook.End()

	fmt.Println("processing")
	<-hook.Process(s)
}
