package main

import (
	"fmt"

	hook "github.com/robotn/gohook"
)

func registerEvent() {
	fmt.Println("--- Please press ctrl + shift + q to stop hook ---")
	hook.Register(hook.KeyDown, []string{"q", "ctrl", "shift"}, func(e hook.Event) {
		fmt.Println("ctrl-shift-q")
		hook.End()
	})

	fmt.Println("--- Please press w ---")
	hook.Register(hook.KeyDown, []string{"w"}, func(e hook.Event) {
		fmt.Println("w-")
	})

	s := hook.Start()
	<-hook.Process(s)
}

// hook listen and return values using detailed examples
func add() {
	fmt.Println("hook add...")
	s := hook.Start()
	defer hook.End()

	ct := false
	for {
		i := <-s

		if i.Kind == hook.KeyHold && i.Rawcode == 59 {
			ct = true
		}

		if ct && i.Rawcode == 12 {
			break
		}
	}
}

// base hook example
func base() {
	fmt.Println("hook start...")
	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		fmt.Println("hook: ", ev)
	}
}

func main() {
	registerEvent()

	base()

	add()
}
