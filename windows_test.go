// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

//go:build windows && purego

package hook

import (
	"testing"
	"time"

	"github.com/vcaesar/tt"
)

// TestWinVKToKeycode verifies the generated vkCode -> VC keycode table agrees
// with github.com/vcaesar/keycode for the keys whose VC value the Register/
// Process matcher relies on. (f11/f12 and a few media keys intentionally differ
// in vcaesar itself, matching the CGo backend, so they are not asserted here.)
func TestWinVKToKeycode(t *testing.T) {
	cases := map[uint16]string{
		0x41: "a",
		0x5A: "z",
		0x30: "0",
		0x39: "9",
		0x20: "space",
		0x0D: "enter",
		0x09: "tab",
		0x1B: "esc",
		0x70: "f1",
		0x79: "f10",
		0x25: "left",
		0x26: "up",
		0x27: "right",
		0x28: "down",
	}

	for vk, name := range cases {
		tt.Equal(t, Keycode[name], vkToKeycode(vk, 0))
	}

	// An unmapped virtual key resolves to VC_UNDEFINED (0).
	tt.Equal(t, uint16(0), vkToKeycode(0x07, 0))
}

// TestWinModifierMask checks modifier bookkeeping mirrors set/unset semantics.
func TestWinModifierMask(t *testing.T) {
	winModifiers = 0

	setKeyModifier(vkLShift, true)
	tt.Equal(t, true, winModifiers&maskShiftL != 0)

	setKeyModifier(vkLControl, true)
	tt.Equal(t, true, winModifiers&maskCtrlL != 0)

	setKeyModifier(vkLShift, false)
	tt.Equal(t, false, winModifiers&maskShiftL != 0)
	tt.Equal(t, true, winModifiers&maskCtrlL != 0)

	// A non-modifier key must not touch the mask.
	before := winModifiers
	setKeyModifier(0x41 /* 'A' */, true)
	tt.Equal(t, before, winModifiers)

	winModifiers = 0
}

// TestWinXButton checks extra-mouse-button decoding from MSLLHOOKSTRUCT.mouseData
// and that the button mask bits are set on press and cleared on release.
func TestWinXButton(t *testing.T) {
	winModifiers = 0

	ms1 := &msLLHookStruct{mouseData: uint32(xbutton1) << 16}
	tt.Equal(t, uint16(4), xButton(ms1, true))
	tt.Equal(t, true, winModifiers&maskButton4 != 0)

	ms2 := &msLLHookStruct{mouseData: uint32(xbutton2) << 16}
	tt.Equal(t, uint16(5), xButton(ms2, true))
	tt.Equal(t, true, winModifiers&maskButton5 != 0)

	// Release clears the bits again, so MouseMove is not stuck as MouseDrag.
	tt.Equal(t, uint16(4), xButton(ms1, false))
	tt.Equal(t, false, winModifiers&maskButton4 != 0)

	tt.Equal(t, uint16(5), xButton(ms2, false))
	tt.Equal(t, false, winModifiers&maskButton5 != 0)
	tt.Equal(t, uint16(0), winModifiers&maskButtons)

	winModifiers = 0
}

// captureEvents runs fn with a fresh event channel and returns everything it
// sent. It first ends any hook session left running (e.g. TestAdd calls
// Start without End), whose real hook events would otherwise leak in.
func captureEvents(fn func()) []Event {
	End()

	ev = make(chan Event, 16)
	asyncon.Store(true)
	defer func() { asyncon.Store(false) }()

	fn()

	out := []Event{}
	for len(ev) != 0 {
		out = append(out, <-ev)
	}
	return out
}

// TestWinMouseWheel verifies the signed HIWORD wheel delta maps to libuiohook
// rotation: +120 (forward/up) -> WheelUp (-1), -120 -> WheelDown (1).
func TestWinMouseWheel(t *testing.T) {
	got := captureEvents(func() {
		processMouseWheel(&msLLHookStruct{mouseData: uint32(uint16(120)) << 16}, wheelVerticalDir)
		processMouseWheel(&msLLHookStruct{mouseData: uint32(uint16(0xFF88)) << 16}, wheelHorizontalDir) // -120
	})
	tt.Equal(t, 2, len(got))
	tt.Equal(t, MouseWheel, got[0].Kind)
	tt.Equal(t, WheelUp, got[0].Rotation)
	tt.Equal(t, uint8(wheelVerticalDir), got[0].Direction)
	tt.Equal(t, WheelDown, got[1].Rotation)
	tt.Equal(t, uint8(wheelHorizontalDir), got[1].Direction)
}

// TestWinButtonRelease verifies release -> MouseHold (RELEASED) then MouseUp
// (CLICKED) only when released at the press position.
func TestWinButtonRelease(t *testing.T) {
	winModifiers = 0
	clickCount, clickTime, clickButton = 0, 0, 0

	got := captureEvents(func() {
		processButtonPressed(&msLLHookStruct{pt: point{10, 20}, time: 1000}, MouseMap["left"])
		processButtonReleased(&msLLHookStruct{pt: point{10, 20}, time: 1050}, MouseMap["left"])
		processButtonPressed(&msLLHookStruct{pt: point{10, 20}, time: 5000}, MouseMap["left"])
		processButtonReleased(&msLLHookStruct{pt: point{30, 20}, time: 5050}, MouseMap["left"])
	})
	tt.Equal(t, 5, len(got))
	tt.Equal(t, MouseDown, got[0].Kind)
	tt.Equal(t, MouseHold, got[1].Kind)
	tt.Equal(t, MouseUp, got[2].Kind)
	tt.Equal(t, uint16(1), got[2].Clicks)
	tt.Equal(t, MouseDown, got[3].Kind)
	tt.Equal(t, MouseHold, got[4].Kind) // moved: no click
}

// TestWinStaleSession verifies a winLoop whose session was ended before it
// went live does not go live once asyncon is set again by a later session:
// no HookEnabled leaks into the new channel and no hook stays installed.
func TestWinStaleSession(t *testing.T) {
	Start()
	End(0) // usually lands before winLoop has installed its hooks

	// A new session's channel; not captureEvents, whose End would re-close ev.
	ev = make(chan Event, 16)
	asyncon.Store(true)
	time.Sleep(300 * time.Millisecond)
	asyncon.Store(false)
	tt.Equal(t, 0, len(ev))

	lck.Lock()
	defer lck.Unlock()
	tt.Equal(t, true, win == nil)
}
