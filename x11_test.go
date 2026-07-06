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
}

// TestX11Dial verifies DISPLAY parsing for the common local and TCP forms.
func TestX11Dial(t *testing.T) {
	// Bad display strings error out.
	_, _, _, err := x11Dial("not-a-display")
	tt.NotNil(t, err)

	_, _, _, err = x11Dial(":")
	tt.NotNil(t, err)
}
