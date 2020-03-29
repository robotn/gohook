# gohook

[![CircleCI Status](https://circleci.com/gh/cauefcr/gohook.svg?style=shield)](https://circleci.com/gh/cauefcr/gohook)
![Appveyor](https://ci.appveyor.com/api/projects/status/github/cauefcr/gohook?branch=master&svg=true)
[![Go Report Card](https://goreportcard.com/badge/github.com/cauefcr/gohook)](https://goreportcard.com/report/github.com/cauefcr/gohook)
[![GoDoc](https://godoc.org/github.com/cauefcr/gohook?status.svg)](https://godoc.org/github.com/cauefcr/gohook)
<!-- This is a work in progress. -->

Based on [libuiohook](https://github.com/kwhat/libuiohook)

```Go
package main

import (
	"fmt"
	//"github.com/robotn/gohook"
	"github.com/cauefcr/gohook"
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