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
#cgo darwin CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework Cocoa

#cgo linux CFLAGS:-I/usr/src -std=gnu99
#cgo linux LDFLAGS: -L/usr/src -lX11 -lXtst
#cgo linux LDFLAGS: -lX11-xcb -lxcb -lxcb-xkb -lxkbcommon -lxkbcommon-x11
//#cgo windows LDFLAGS: -lgdi32 -luser32

#include "event/goEvent.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

const (
	// Version get the gohook version
	Version = "v0.41.0"

	// HookEnabled honk enable status
	HookEnabled  = 1 // iota
	HookDisabled = 2

	KeyDown = 4 // 3
	KeyHold = 3 // 4
	KeyUp   = 5 // 5

	MouseDown = 7 // 6
	MouseHold = 8 // 7
	MouseUp   = 6 // 8

	MouseMove  = 9
	MouseDrag  = 10
	MouseWheel = 11

	FakeEvent = 12

	// Keychar could be v
	CharUndefined = 0xFFFF
	WheelUp       = -1
	WheelDown     = 1
	Debug         = DebugLevel(13)
	Silent        = DebugLevel(14)
)

type Kind uint8
type Code uint16
type DebugLevel uint8

// Event Holds a system event
//
// If it's a Keyboard event the relevant fields are:
// Mask, Keycode, Rawcode, and Keychar,
// Keychar is probably what you want.
//
// If it's a Mouse event the relevant fields are:
// Button, Clicks, X, Y, Amount, Rotation and Direction
type Event struct {
	Kind     Kind `json:"id"`
	When     time.Time
	Mask     uint16 `json:"mask"`
	Reserved uint16 `json:"reserved"`

	Keycode uint16 `json:"keycode"`
	Rawcode uint16 `json:"rawcode"`
	Keychar rune   `json:"keychar"`

	Button uint16 `json:"button"`
	Clicks uint16 `json:"clicks"`

	X int16 `json:"x"`
	Y int16 `json:"y"`

	Amount    uint16 `json:"amount"`
	Rotation  int32  `json:"rotation"`
	Direction uint8  `json:"direction"`
}

var (
	/*
		{
			KeyDown: {
				[0,1,2,4]: func(e Event) {
					fmt.Println("a")
				},
			},
		}
	*/
	registry      = make(map[Kind]map[[4]Code]func(Event))
	mouseRegistry = make(map[Kind]map[[4]Code]func(Event))
	pressed       = make(map[Code]bool)
	mousePressed  = make(map[Code]bool)
	ev            = make(chan Event, 1024)
	lck           = sync.RWMutex{}
	debugLevel    = DebugLevel(0)
)

func allPressed(pressed map[Code]bool, keys [4]Code) bool {
	for _, key := range keys {
		if key != 0 && !pressed[key] {
			return false
		}
	}
	return true
}

func allUnpressed(pressed map[Code]bool, keys [4]Code) bool {
	for _, key := range keys {
		if key != 0 && pressed[key] {
			return false
		}
	}
	return true
}

func SetDebugLevel(level DebugLevel) {
	debugLevel = level
}

// Register gohook event
func Register(when Kind, cmds []string, cb func(Event)) error {
	if len(cmds) > 4 {
		return fmt.Errorf("too many keys. max 4")
	}

	tmp := [4]Code{}

	for i, v := range cmds {
		var code uint16
		var ok bool
		if when == KeyDown || when == KeyUp {
			code, ok = Keycode[v]
			if !ok {
				fmt.Printf("invalid key: %s\n", v)
				fmt.Println("skipping...")
				return nil
			}
		} else {
			if v == "mleft" {
				v = "left"
			}

			if v == "mright" {
				v = "right"
			}

			if v == "mcenter" {
				v = "center"
			}

			code, ok = MouseMap[v]
			if !ok {
				fmt.Printf("invalid mouse button: %s\n", v)
				fmt.Println("skipping..")
				return nil
			}
		}

		tmp[i] = Code(code)
	}

	if when == KeyDown || when == KeyUp {
		if _, ok := registry[when]; !ok {
			registry[when] = make(map[[4]Code]func(Event))
		}
		registry[when][tmp] = cb
	} else {
		if _, ok := mouseRegistry[when]; !ok {
			mouseRegistry[when] = make(map[[4]Code]func(Event))
		}
		mouseRegistry[when][tmp] = cb
	}

	hookLog("registered %v as %v\n", cmds, tmp)
	return nil
}

// Process return go hook process
func Process(evChan <-chan Event) (out chan bool) {
	out = make(chan bool)
	go func() {
		for ev := range evChan {
			if ev.Kind != KeyDown && ev.Kind != KeyUp && ev.Kind != MouseDown && ev.Kind != MouseUp {
				continue
			}

			hookLog("processing %v\n", ev)
			switch ev.Kind {
			case KeyDown, KeyHold:
				hookLog("setting pressed[%v] = true\n", ev.Keycode)
				pressed[Code(ev.Keycode)] = true
			case KeyUp:
				hookLog("setting pressed[%v] = false\n", ev.Keycode)
				pressed[Code(ev.Keycode)] = false
			case MouseDown:
				hookLog("setting mousePressed[%v] = true\n", ev.Button)
				mousePressed[Code(ev.Button)] = true
			case MouseUp:
				hookLog("setting mousePressed[%v] = false\n", ev.Button)
				mousePressed[Code(ev.Button)] = false
			}

			if ev.Kind == KeyDown || ev.Kind == KeyUp {
				for combination, v := range registry[ev.Kind] {
					switch ev.Kind {
					case KeyDown:
						hookLog("checking if %v is pressed\n", combination)
						if allPressed(pressed, combination) {
							hookLog("calling %v\n", combination)
							v(ev)
						} else {
							hookLog("not all keys are pressed\n")
						}
					case KeyUp:
						hookLog("checking if %v is pressed\n", combination)
						if allUnpressed(pressed, combination) {
							hookLog("calling %v\n", combination)
							v(ev)
						} else {
							hookLog("not all keys are pressed\n")
						}
					}
				}
			} else {
				for combination, v := range mouseRegistry[ev.Kind] {
					switch ev.Kind {
					case MouseDown:
						hookLog("checking if %v is pressed\n", combination)
						if allPressed(mousePressed, combination) {
							hookLog("calling %v\n", combination)
							v(ev)
						} else {
							hookLog("not all keys are pressed\n")
						}
					case MouseUp:
						hookLog("checking if %v is pressed\n", combination)
						if allUnpressed(mousePressed, combination) {
							hookLog("calling %v\n", combination)
							v(ev)
						} else {
							hookLog("not all keys are pressed\n")
						}
					}
				}
			}
		}

		out <- true
	}()

	return
}

// String return formatted hook kind string
func (e Event) String() string {
	switch e.Kind {
	case HookEnabled:
		return fmt.Sprintf("%v - Event: {Kind: HookEnabled}", e.When)
	case HookDisabled:
		return fmt.Sprintf("%v - Event: {Kind: HookDisabled}", e.When)
	case KeyDown:
		return fmt.Sprintf("%v - Event: {Kind: KeyDown, Rawcode: %v, Keychar: %v}",
			e.When, e.Rawcode, e.Keychar)
	case KeyHold:
		return fmt.Sprintf("%v - Event: {Kind: KeyHold, Rawcode: %v, Keychar: %v}",
			e.When, e.Rawcode, e.Keychar)
	case KeyUp:
		return fmt.Sprintf("%v - Event: {Kind: KeyUp, Rawcode: %v, Keychar: %v}",
			e.When, e.Rawcode, e.Keychar)
	case MouseDown:
		return fmt.Sprintf("%v - Event: {Kind: MouseDown, Button: %v, X: %v, Y: %v, Clicks: %v}",
			e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseHold:
		return fmt.Sprintf("%v - Event: {Kind: MouseHold, Button: %v, X: %v, Y: %v, Clicks: %v}",
			e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseUp:
		return fmt.Sprintf("%v - Event: {Kind: MouseUp, Button: %v, X: %v, Y: %v, Clicks: %v}",
			e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseMove:
		return fmt.Sprintf("%v - Event: {Kind: MouseMove, Button: %v, X: %v, Y: %v, Clicks: %v}",
			e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseDrag:
		return fmt.Sprintf("%v - Event: {Kind: MouseDrag, Button: %v, X: %v, Y: %v, Clicks: %v}",
			e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseWheel:
		return fmt.Sprintf("%v - Event: {Kind: MouseWheel, Amount: %v, Rotation: %v, Direction: %v}",
			e.When, e.Amount, e.Rotation, e.Direction)
	case FakeEvent:
		return fmt.Sprintf("%v - Event: {Kind: FakeEvent}", e.When)
	}

	return "Unknown event, contact the mantainers."
}

// Start adds global event hook to OS
// returns event channel
func Start() chan Event {
	ev = make(chan Event, 1024)
	hookLog("starting C.start_ev")
	go C.start_ev()

	go func() {
		for {
			C.pollEv()
			time.Sleep(time.Millisecond * 10)
		}
	}()

	return ev
}

// End removes global event hook
func End() {
	C.endPoll()
	C.stop_event()
	time.Sleep(time.Millisecond * 10)

	for len(ev) != 0 {
		<-ev
	}
	close(ev)

	pressed = make(map[Code]bool, 256)
	registry = make(map[Kind]map[[4]Code]func(Event))
}

// AddEvent add the block event listener
func addEvent(key string) int {
	cs := C.CString(key)
	defer C.free(unsafe.Pointer(cs))

	eve := C.add_event(cs)
	geve := int(eve)

	return geve
}

// StopEvent stop the block event listener
func StopEvent() {
	C.stop_event()
}

func hookLog(format string, args ...any) {
	if debugLevel == Debug {
		fmt.Printf(format, args...)
	}
}
