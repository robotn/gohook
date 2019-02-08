package main

import (
	"fmt"
	"time"

	"github.com/cauefcr/gohook"
)

func main() {
	s := hook.Start()
	tout := time.After(time.Second * 10)
	done := false
	for !done {
		select {
		case i := <-s:
			if i.Keychar == uint16('q') {
				tout = time.After(1 * time.Millisecond)
			}
			fmt.Printf("%+v\n", i)
		case <-tout:
			fmt.Print("Done.")
			done = true
			break;
		}
	}
	hook.End()

}
