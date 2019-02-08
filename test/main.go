package main

import (
	"fmt"

	"github.com/robotn/gohook"
)

func main() {
	s := hook.StartEvent()

	go func() {
		fmt.Print("woo!")
		for i:=range s {
			fmt.Println(i)
		}
	}()
	// hook.AsyncHook()
	veve := hook.AddEvent("v")
	if veve == 0 {
		fmt.Println("v...")
	}
}
