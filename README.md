# ghook (gohook fork)

[![CircleCI Status](https://circleci.com/gh/cauefcr/ghook.svg?style=shield)](https://circleci.com/gh/cauefcr/ghook)
![Appveyor](https://ci.appveyor.com/api/projects/status/github/cauefcr/ghook?branch=master&svg=true)
[![Go Report Card](https://goreportcard.com/badge/github.com/cauefcr/ghook)](https://goreportcard.com/report/github.com/cauefcr/ghook)
[![GoDoc](https://godoc.org/github.com/cauefcr/ghook?status.svg)](https://godoc.org/github.com/cauefcr/ghook)
<!-- This is a work in progress. -->

Based on [libuiohook](https://github.com/kwhat/libuiohook)

```Go
package main

import (
	"fmt"
	//"github.com/robotn/gohook"
	"github.com/cauefcr/ghook"
)

func main() {
	EvChan := hook.Start()
	defer hook.End()
	// drawing := false

	hook.Register(hook.KeyDown, []string{"alt", "p"}, func(e hook.Event) {
		fmt.Println("alt-p ", e)
	})

	hook.Register(hook.KeyDown, []string{}, func(e hook.Event) {
		fmt.Println(e.Keycode)
	})

	<-hook.Process(EvChan)
}
```