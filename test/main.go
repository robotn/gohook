package main

import (
	"fmt"
	"time"

	"github.com/robotn/gohook"
)

func main() {
	s := hook.Start()
	defer hook.End()

	tout := time.After(time.Hour * 2)
	done := false
	for !done {
		select {
		case i := <-s:
			if i.Kind >= hook.KeyDown && i.Kind <= hook.KeyUp {
				if i.Keychar == 'q' {
					tout = time.After(0)
				}

				fmt.Printf("%v key: %c:%v\n", i.Kind, i.Keychar, i.Rawcode)
			} else if i.Kind >= hook.MouseDown && i.Kind < hook.MouseWheel {
				//fmt.Printf("x: %v, y: %v, button: %v\n", i.X, i.Y, i.Button)
			} else if i.Kind == hook.MouseWheel {
				//fmt.Printf("x: %v, y: %v, button: %v, wheel: %v, rotation: %v\n", i.X, i.Y, i.Button,i.Amount,i.Rotation)
			} else {
				fmt.Printf("%+v\n", i)
			}

		case <-tout:
			fmt.Print("Done.")
			done = true
			break
		}
	}

}
