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

// TestWinXButton checks extra-mouse-button decoding from MSLLHOOKSTRUCT.mouseData.
func TestWinXButton(t *testing.T) {
	winModifiers = 0

	ms1 := &msLLHookStruct{mouseData: uint32(xbutton1) << 16}
	tt.Equal(t, uint16(4), xButton(ms1))
	tt.Equal(t, true, winModifiers&maskButton4 != 0)

	ms2 := &msLLHookStruct{mouseData: uint32(xbutton2) << 16}
	tt.Equal(t, uint16(5), xButton(ms2))
	tt.Equal(t, true, winModifiers&maskButton5 != 0)

	winModifiers = 0
}
