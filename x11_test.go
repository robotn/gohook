// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

//go:build linux && purego && !wayland

package hook

import (
	"testing"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/vcaesar/tt"
)

// TestKeysymToRune covers the keysym -> rune classification used to fill
// Event.Keychar: ASCII/Latin-1 keysyms map to their code point, the direct
// Unicode keysym range is masked, and non-character keysyms report
// CharUndefined.
func TestKeysymToRune(t *testing.T) {
	tt.Equal(t, 'a', keysymToRune(0x0061))     // XK_a
	tt.Equal(t, 'A', keysymToRune(0x0041))     // XK_A
	tt.Equal(t, '0', keysymToRune(0x0030))     // XK_0
	tt.Equal(t, ' ', keysymToRune(0x0020))     // XK_space
	tt.Equal(t, ';', keysymToRune(0x003b))     // XK_semicolon
	tt.Equal(t, 'ä', keysymToRune(0x00e4))     // XK_adiaeresis (Latin-1)
	tt.Equal(t, '€', keysymToRune(0x010020ac)) // direct Unicode keysym

	// Non-character keysyms -> CharUndefined.
	tt.Equal(t, rune(CharUndefined), keysymToRune(0))      // NoSymbol
	tt.Equal(t, rune(CharUndefined), keysymToRune(0xff0d)) // XK_Return
	tt.Equal(t, rune(CharUndefined), keysymToRune(0xffe1)) // XK_Shift_L
}

// TestX11Button verifies the X core button -> gohook MouseMap translation,
// including the X middle button (2) mapping to "center".
func TestX11Button(t *testing.T) {
	tt.Equal(t, MouseMap["left"], x11Button(1))
	tt.Equal(t, MouseMap["center"], x11Button(2))
	tt.Equal(t, MouseMap["right"], x11Button(3))
}

// TestX11Wheel verifies scroll pseudo-buttons map to MouseWheel with the right
// direction and rotation sign.
func TestX11Wheel(t *testing.T) {
	up := x11Wheel(4, 10, 20, 0)
	tt.Equal(t, MouseWheel, up.Kind)
	tt.Equal(t, wheelVertical, up.Direction)
	tt.Equal(t, WheelUp, up.Rotation)
	tt.Equal(t, int16(10), up.X)
	tt.Equal(t, int16(20), up.Y)

	down := x11Wheel(5, 0, 0, 0)
	tt.Equal(t, wheelVertical, down.Direction)
	tt.Equal(t, WheelDown, down.Rotation)

	left := x11Wheel(6, 0, 0, 0)
	tt.Equal(t, wheelHorizontal, left.Direction)

	right := x11Wheel(7, 0, 0, 0)
	tt.Equal(t, wheelHorizontal, right.Direction)
}

// TestMaskFromState verifies X modifier bits map onto gohook's virtual mask.
func TestMaskFromState(t *testing.T) {
	tt.Equal(t, maskShiftL, maskFromState(xShiftMask))
	tt.Equal(t, maskCtrlL, maskFromState(xControlMask))
	tt.Equal(t, maskAltL, maskFromState(xMod1Mask))
	tt.Equal(t, maskMetaL, maskFromState(xMod4Mask))
	tt.Equal(t, maskCapsLock, maskFromState(xLockMask))
	tt.Equal(t, maskShiftL|maskCtrlL, maskFromState(xShiftMask|xControlMask))
}

// TestKeysymAt exercises the keyboard-mapping index math, including
// out-of-range guards.
func TestKeysymAt(t *testing.T) {
	st := &x11State{
		minKeycode: 8,
		perCode:    2,
		// keycode 8 -> {0x61,0x41}, keycode 9 -> {0x62,0x42}
		keysyms: []xproto.Keysym{0x61, 0x41, 0x62, 0x42},
	}

	tt.Equal(t, xproto.Keysym(0x61), st.keysymAt(8, 0))
	tt.Equal(t, xproto.Keysym(0x41), st.keysymAt(8, 1))
	tt.Equal(t, xproto.Keysym(0x62), st.keysymAt(9, 0))

	// Out of range -> 0.
	tt.Equal(t, xproto.Keysym(0), st.keysymAt(200, 0))

	// keychar picks the shifted column when Shift is set.
	tt.Equal(t, 'a', st.keychar(8, 0))
	tt.Equal(t, 'A', st.keychar(8, xShiftMask))

	// keysymFor is what fills Event.Rawcode (CGo backend parity: rawcode is
	// the state-resolved keysym, not the evdev code).
	tt.Equal(t, xproto.Keysym(0x61), st.keysymFor(8, 0))
	tt.Equal(t, xproto.Keysym(0x41), st.keysymFor(8, xShiftMask))
	tt.Equal(t, xproto.Keysym(0), st.keysymFor(200, 0))
}

// TestX11Dial verifies DISPLAY parsing for the common local and TCP forms.
func TestX11Dial(t *testing.T) {
	// Bad display strings error out.
	_, _, _, err := x11Dial("not-a-display")
	tt.NotNil(t, err)

	_, _, _, err = x11Dial(":")
	tt.NotNil(t, err)
}

// xEvent packs a 32-byte core KeyButtonPointer-style event (KeyPress,
// ButtonPress, MotionNotify ...) as the RECORD data stream carries it.
func xEvent(typ, detail byte, rootX, rootY int16, state uint16) []byte {
	buf := make([]byte, 32)
	buf[0] = typ
	buf[1] = detail
	xgb.Put16(buf[20:], uint16(rootX))
	xgb.Put16(buf[22:], uint16(rootY))
	xgb.Put16(buf[28:], state)
	return buf
}

// captureEvents runs fn with a fresh event channel and returns everything it
// sent.
func captureEvents(fn func()) []Event {
	ev = make(chan Event, 16)
	asyncon = true
	defer func() { asyncon = false }()

	fn()

	out := []Event{}
	for len(ev) != 0 {
		out = append(out, <-ev)
	}
	return out
}

// TestX11ButtonExtra verifies X side/extra buttons 8/9 map to gohook 4/5.
func TestX11ButtonExtra(t *testing.T) {
	tt.Equal(t, uint16(4), x11Button(8))
	tt.Equal(t, uint16(5), x11Button(9))
}

// TestX11OnButton verifies CGo parity: press -> MouseDown; release ->
// MouseHold (RELEASED) followed by MouseUp (CLICKED) only when released at the
// press position.
func TestX11OnButton(t *testing.T) {
	st := &x11State{down: map[byte]bool{}}

	got := captureEvents(func() {
		x11OnButton(st, xEvent(xproto.ButtonPress, 1, 10, 20, 0), true)
		x11OnButton(st, xEvent(xproto.ButtonRelease, 1, 10, 20, xShiftMask), false)
	})
	tt.Equal(t, 3, len(got))
	tt.Equal(t, MouseDown, got[0].Kind)
	tt.Equal(t, MouseHold, got[1].Kind)
	tt.Equal(t, MouseUp, got[2].Kind)
	tt.Equal(t, MouseMap["left"], got[2].Button)
	tt.Equal(t, int16(10), got[2].X)
	tt.Equal(t, maskShiftL, got[2].Mask)

	// Released elsewhere: no click.
	got = captureEvents(func() {
		x11OnButton(st, xEvent(xproto.ButtonPress, 3, 1, 1, 0), true)
		x11OnButton(st, xEvent(xproto.ButtonRelease, 3, 50, 1, 0), false)
	})
	tt.Equal(t, 2, len(got))
	tt.Equal(t, MouseHold, got[1].Kind)
	tt.Equal(t, MouseMap["right"], got[1].Button)

	// Wheel: single MouseWheel on press, release dropped.
	got = captureEvents(func() {
		x11OnButton(st, xEvent(xproto.ButtonPress, 4, 0, 0, 0), true)
		x11OnButton(st, xEvent(xproto.ButtonRelease, 4, 0, 0, 0), false)
	})
	tt.Equal(t, 1, len(got))
	tt.Equal(t, MouseWheel, got[0].Kind)
}

// TestX11OnMotion verifies motion with a held button is a MouseDrag and the
// modifier mask is carried.
func TestX11OnMotion(t *testing.T) {
	got := captureEvents(func() {
		x11OnMotion(xEvent(xproto.MotionNotify, 0, 3, 4, 0))
		x11OnMotion(xEvent(xproto.MotionNotify, 0, 5, 6, xproto.ButtonMask1|xControlMask))
	})
	tt.Equal(t, 2, len(got))
	tt.Equal(t, MouseMove, got[0].Kind)
	tt.Equal(t, int16(3), got[0].X)
	tt.Equal(t, MouseDrag, got[1].Kind)
	tt.Equal(t, int16(6), got[1].Y)
	tt.Equal(t, maskCtrlL, got[1].Mask)
}

// TestX11OnKey verifies KeyDown / KeyHold (auto-repeat) / KeyUp and the
// evdev Keycode + keysym Rawcode split.
func TestX11OnKey(t *testing.T) {
	st := &x11State{
		down:       map[byte]bool{},
		minKeycode: 8,
		perCode:    2,
		keysyms:    []xproto.Keysym{0x61, 0x41}, // keycode 8: a / A
	}

	got := captureEvents(func() {
		x11OnKey(st, xEvent(xproto.KeyPress, 8, 0, 0, 0), true)
		x11OnKey(st, xEvent(xproto.KeyPress, 8, 0, 0, 0), true)
		x11OnKey(st, xEvent(xproto.KeyRelease, 8, 0, 0, xShiftMask), false)
	})
	tt.Equal(t, 3, len(got))
	tt.Equal(t, KeyDown, got[0].Kind)
	tt.Equal(t, KeyHold, got[1].Kind)
	tt.Equal(t, KeyUp, got[2].Kind)
	tt.Equal(t, uint16(0), got[0].Keycode) // evdev = X keycode - 8
	tt.Equal(t, uint16(0x61), got[0].Rawcode)
	tt.Equal(t, 'a', got[0].Keychar)
	tt.Equal(t, 'A', got[2].Keychar)
	tt.Equal(t, "a", RawcodeToKeychar(0x61))
}

// TestX11Release verifies teardown ownership: only the live session is torn
// down, and a second release is a no-op.
func TestX11Release(t *testing.T) {
	st := &x11State{}
	lck.Lock()
	xst = st
	lck.Unlock()

	x11Release(st)
	lck.Lock()
	tt.Equal(t, true, xst == nil)
	lck.Unlock()

	x11Release(st) // no-op, must not panic on nil connections
}
