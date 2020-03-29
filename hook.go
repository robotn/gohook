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

#include "event/goEvent.h"

*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)
onst (
	HookEnabled  = 1 //iota
	HookDisabled = 2
	KeyDown      = 3
	KeyHold      = 4
	KeyUp        = 5
	MouseUp      = 6
	MouseHold    = 7
	MouseDown    = 8
	MouseMove    = 9
	MouseDrag    = 10
	MouseWheel   = 11
	FakeEvent    = 12
	//Keychar could be   v
	CharUndefined = 0xFFFF
	WheelUp       = -1
	WheelDown     = 1
)

//Holds a system event
//If it's a Keyboard event the relevant fields are: Mask, Keycode, Rawcode, and Keychar,
//Keychar is probably what you want. If it's a Mouse event the relevant fields are:
//Button, Clicks, X, Y, Amount, Rotation and Direction
type Event struct {
	Kind      uint8 `json:"id"`
	When      time.Time
	Mask      uint16 `json:"mask"`
	Reserved  uint16 `json:"reserved"`
	Keycode   uint16 `json:"keycode"`
	Rawcode   uint16 `json:"rawcode"`
	Keychar   rune   `json:"keychar"`
	Button    uint16 `json:"button"`
	Clicks    uint16 `json:"clicks"`
	X         int16  `json:"x"`
	Y         int16  `json:"y"`
	Amount    uint16 `json:"amount"`
	Rotation  int16  `json:"rotation"`
	Direction uint8  `json:"direction"`
}

var (
	ev      = make(chan Event, 1024)
	asyncon = false
	pressed = make(map[uint16]bool, 256)
	used    = []int{}
	keys    = map[int][]uint16{}
	cbs     = map[int]func(Event){}
	events  = map[uint8][]int{}
)

//from robotgo
type uMap map[string]uint16

// MouseMap robotgo hook mouse's code map
var MouseMap = uMap{
	"left":       1,
	"right":      2,
	"center":     3,
	"wheelDown":  4,
	"wheelUp":    5,
	"wheelLeft":  6,
	"wheelRight": 7,
}

// Keycode robotgo hook key's code map
var Keycode = uMap{
	"`": 41,
	"1": 2,
	"2": 3,
	"3": 4,
	"4": 5,
	"5": 6,
	"6": 7,
	"7": 8,
	"8": 9,
	"9": 10,
	"0": 11,
	"-": 12,
	"+": 13,
	//
	"q":  16,
	"w":  17,
	"e":  18,
	"r":  19,
	"t":  20,
	"y":  21,
	"u":  22,
	"i":  23,
	"o":  24,
	"p":  25,
	"[":  26,
	"]":  27,
	"\\": 43,
	//
	"a": 30,
	"s": 31,
	"d": 32,
	"f": 33,
	"g": 34,
	"h": 35,
	"j": 36,
	"k": 37,
	"l": 38,
	";": 39,
	"'": 40,
	//
	"z": 44,
	"x": 45,
	"c": 46,
	"v": 47,
	"b": 48,
	"n": 49,
	"m": 50,
	",": 51,
	".": 52,
	"/": 53,
	//
	"f1":  59,
	"f2":  60,
	"f3":  61,
	"f4":  62,
	"f5":  63,
	"f6":  64,
	"f7":  65,
	"f8":  66,
	"f9":  67,
	"f10": 68,
	"f11": 69,
	"f12": 70,
	// more
	"esc":     1,
	"delete":  14,
	"tab":     15,
	"ctrl":    29,
	"control": 29,
	"alt":     56,
	"space":   57,
	"shift":   42,
	"rshift":  54,
	"enter":   28,
	"cmd":     3675,
	"command": 3675,
	"rcmd":    3676,
	"ralt":    3640,
	"up":      57416,
	"down":    57424,
	"left":    57419,
	"right":   57421,
}

func allPressed(pressed map[uint16]bool, keys ...uint16) bool {

	for _, i := range keys {
		// fmt.Print(i)
		if pressed[i] == false {
			return false
		}
	}
	return true
}

func Register(when uint8, cmds []string, cb func(Event)) {
	key := len(used)
	used = append(used, key)
	tmp := []uint16{}
	for _, v := range cmds {
		tmp = append(tmp, Keycode[v])
	}
	keys[key] = tmp
	fmt.Println(tmp)
	cbs[key] = cb
	events[when] = append(events[when], key)
	return
}

func Process(EvChan <-chan Event) (out chan bool) {
	out = make(chan bool)
	go func() {
		for ev := range EvChan {
			if ev.Kind == KeyDown || ev.Kind == KeyHold {
				pressed[ev.Keycode] = true
			} else if ev.Kind == KeyUp {
				pressed[ev.Keycode] = false
			}
			for _, v := range events[ev.Kind] {
				if allPressed(pressed, keys[v]...) {
					cbs[v](ev)
				}
			}
		}
		out <- true
	}()
	return out
}

func (e Event) String() string {
	switch e.Kind {
	case HookEnabled:
		return fmt.Sprintf("%v - Event: {Kind: HookEnabled}", e.When)
	case HookDisabled:
		return fmt.Sprintf("%v - Event: {Kind: HookDisabled}", e.When)
	case KeyUp:
		return fmt.Sprintf("%v - Event: {Kind: KeyUp, Rawcode: %v, Keychar: %v}", e.When, e.Rawcode, e.Keychar)
	case KeyHold:
		return fmt.Sprintf("%v - Event: {Kind: KeyHold, Rawcode: %v, Keychar: %v}", e.When, e.Rawcode, e.Keychar)
	case KeyDown:
		return fmt.Sprintf("%v - Event: {Kind: KeyDown, Rawcode: %v, Keychar: %v}", e.When, e.Rawcode, e.Keychar)
	case MouseUp:
		return fmt.Sprintf("%v - Event: {Kind: MouseUp, Button: %v, X: %v, Y: %v, Clicks: %v}", e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseHold:
		return fmt.Sprintf("%v - Event: {Kind: MouseHold, Button: %v, X: %v, Y: %v, Clicks: %v}", e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseDown:
		return fmt.Sprintf("%v - Event: {Kind: MouseDown, Button: %v, X: %v, Y: %v, Clicks: %v}", e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseMove:
		return fmt.Sprintf("%v - Event: {Kind: MouseMove, Button: %v, X: %v, Y: %v, Clicks: %v}", e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseDrag:
		return fmt.Sprintf("%v - Event: {Kind: MouseDrag, Button: %v, X: %v, Y: %v, Clicks: %v}", e.When, e.Button, e.X, e.Y, e.Clicks)
	case MouseWheel:
		return fmt.Sprintf("%v - Event: {Kind: MouseWheel, Amount: %v, Rotation: %v, Direction: %v}", e.When, e.Amount, e.Rotation, e.Direction)
	case FakeEvent:
		return fmt.Sprintf("%v - Event: {Kind: FakeEvent}", e.When)
	}
	return "Unknown event, contact the mantainers"
}

func RawcodetoKeychar(r uint16) string {
	return raw2key[r]
}

func KeychartoRawcode(kc string) uint16 {
	return keytoraw[kc]
}

// Adds global event hook to OS
// returns event channel
func Start() chan Event {
	asyncon = true
	go C.startev()
	go func() {
		for {
			C.pollEv()
			time.Sleep(time.Millisecond * 50)
			//todo: find smallest time that does not destroy the cpu utilization
			if !asyncon {
				return
			}
		}
	}()
	return ev
}

// End removes global event hook
func End() {
	C.endPoll()
	C.stop_event()
	for len(ev) != 0 {
		<-ev
	}
	asyncon = false
}

// AddEvent add event listener
// func AddEvent(key string) int {
// 	cs := C.CString(key)
// 	defer C.free(unsafe.Pointer(cs))

// 	eve := C.add_event(cs)
// 	geve := int(eve)

// 	return geve
// }
