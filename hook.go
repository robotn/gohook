// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

package hook

/*
#cgo darwin CFLAGS: -x objective-c  -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework Cocoa

#cgo linux CFLAGS:-I/usr/src
#cgo linux LDFLAGS: -L/usr/src -lX11 -lXtst
#cgo linux LDFLAGS: -lX11-xcb -lxcb -lxcb-xkb -lxkbcommon -lxkbcommon-x11
//#cgo windows LDFLAGS: -lgdi32 -luser32

// #include "event/hook_async.h"
#include "event/goEvent.h"

*/
import "C"

import (
	"time"
)

//todo: add enums
const (
	HOOK_ENABLED   = 1 //iota
	HOOK_DISABLED  = 2
	KEY_TYPED      = 3
	KEY_PRESSED    = 4
	KEY_RELEASED   = 5
	MOUSE_CLICKED  = 6
	MOUSE_PRESSED  = 7
	MOUSE_RELEASED = 8
	MOUSE_MOVED    = 9
	MOUSE_DRAGGED  = 10
	MOUSE_WHEEL    = 11
)

type Event struct {
	Kind      uint8  `json:"id"`
	When      time.Time
	Mask      uint16 `json:"mask"`
	Reserved  uint16 `json:"reserved"`
	Keycode   uint16 `json:"keycode"`
	Rawcode   uint16 `json:"rawcode"`
	Keychar   uint16 `json:"keychar"`
	Button    uint16 `json:"button"`
	Clicks    uint16 `json:"clicks"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Ammount   uint16 `json:"ammount"`
	Rotation  int16  `json:"rotation"`
	Direction uint8  `json:"direction"`
}

var (
	ev      chan Event = make(chan Event, 1024)
	asyncon bool       = false
)

func Start() chan Event {
	//fmt.Print("Here1")
	asyncon = true
	go C.startev()
	go func() {
		for {
			C.pollEv()
			time.Sleep(time.Millisecond * 50)
			//todo: find smallest time that does not destroy the cpu utilization
			//fmt.Println("_here_")
			if ! asyncon {
				return
			}
		}
		//fmt.Print("WOOOOOOOOOT")
	}()
	//fmt.Print("Here2")
	return ev
}

// StopEvent stop event listener
func End() {
	C.endPoll()
	C.stop_event()
	for len(ev) != 0 {
		<-ev
	}
	asyncon = false
	//C.chan_close(C.events);
}
