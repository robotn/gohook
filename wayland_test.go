// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

//go:build linux && wayland

package hook

import (
	"testing"

	"github.com/vcaesar/tt"
)

// TestWaylandMouseButton verifies evdev button codes map to gohook buttons.
func TestWaylandMouseButton(t *testing.T) {
	tt.Equal(t, MouseMap["left"], mouseButton(btnLeft))
	tt.Equal(t, MouseMap["right"], mouseButton(btnRight))
	tt.Equal(t, MouseMap["center"], mouseButton(btnMiddle))
	tt.Equal(t, uint16(4), mouseButton(btnSide))
	tt.Equal(t, uint16(5), mouseButton(btnExtra))
}

// TestWaylandKeyEvent verifies evdev codes resolve to vcaesar Keycode values
// and a printable Keychar, keeping RawcodeToKeychar in sync.
func TestWaylandKeyEvent(t *testing.T) {
	e := keyEvent(KeyDown, 30) // KEY_A
	tt.Equal(t, KeyDown, e.Kind)
	tt.Equal(t, uint16(30), e.Rawcode)
	tt.Equal(t, Keycode["a"], e.Keycode)
	tt.Equal(t, 'a', e.Keychar)
	tt.Equal(t, "a", RawcodeToKeychar(30))

	u := keyEvent(KeyUp, 0xFFF0)
	tt.Equal(t, uint16(0xFFF0), u.Keycode)
	tt.Equal(t, rune(CharUndefined), u.Keychar)
}

// TestWaylandRelease verifies teardown ownership: only the live session is
// released, and a second release is a no-op.
func TestWaylandRelease(t *testing.T) {
	st := &waylandState{}
	lck.Lock()
	wl = st
	lck.Unlock()

	waylandRelease(st)
	lck.Lock()
	tt.Equal(t, true, wl == nil)
	lck.Unlock()

	waylandRelease(st) // no-op
}
