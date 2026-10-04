// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

//go:build darwin && purego

package hook

import (
	"testing"

	"github.com/vcaesar/tt"
)

// TestDarwinFieldConstants pins the CGEventField ids to CGEventTypes.h;
// kCGMouseEventButtonNumber is 3 (23 is a tablet field).
func TestDarwinFieldConstants(t *testing.T) {
	tt.Equal(t, uint32(1), fieldMouseClickState)
	tt.Equal(t, uint32(3), fieldMouseButtonNumber)
	tt.Equal(t, uint32(9), fieldKeyboardKeycode)
	tt.Equal(t, uint32(11), fieldScrollWheelDelta1)
	tt.Equal(t, uint32(12), fieldScrollWheelDelta2)
}

// TestDarwinWheelSign verifies libuiohook parity: scrolling up (positive
// axis-1 delta) reports WheelUp (-1), like the CGo/X11/Windows backends.
func TestDarwinWheelSign(t *testing.T) {
	up := wheelFromDeltas(1, 0, 10, 20, 0)
	tt.Equal(t, MouseWheel, up.Kind)
	tt.Equal(t, wheelVertical, up.Direction)
	tt.Equal(t, WheelUp, up.Rotation)
	tt.Equal(t, uint16(1), up.Amount)
	tt.Equal(t, int16(10), up.X)
	tt.Equal(t, int16(20), up.Y)

	down := wheelFromDeltas(-3, 0, 0, 0, 0)
	tt.Equal(t, wheelVertical, down.Direction)
	tt.Equal(t, int32(3), down.Rotation)
	tt.Equal(t, uint16(3), down.Amount)

	left := wheelFromDeltas(0, 1, 0, 0, 0)
	tt.Equal(t, wheelHorizontal, left.Direction)
	tt.Equal(t, WheelUp, left.Rotation)
}

// TestDarwinMaskFromFlags maps Quartz modifier flags onto gohook's mask.
func TestDarwinMaskFromFlags(t *testing.T) {
	tt.Equal(t, maskShiftL, maskFromFlags(flagShift))
	tt.Equal(t, maskCtrlL, maskFromFlags(flagControl))
	tt.Equal(t, maskMetaL, maskFromFlags(flagCommand))
	tt.Equal(t, maskAltL, maskFromFlags(flagAlternate))
	tt.Equal(t, maskCapsLock, maskFromFlags(flagAlphaShift))
	tt.Equal(t, uint16(0), maskFromFlags(flagSecondaryFn))
	tt.Equal(t, maskShiftL|maskMetaL, maskFromFlags(flagShift|flagCommand))
}

// TestDarwinMakeKeyEvent checks Keycode/Keychar resolution via the static
// darwin keymap so Register() hotkeys match as on the CGo backend.
func TestDarwinMakeKeyEvent(t *testing.T) {
	e := makeKeyEvent(KeyDown, 0, flagShift) // kVK_ANSI_A
	tt.Equal(t, KeyDown, e.Kind)
	tt.Equal(t, uint16(0), e.Rawcode)
	tt.Equal(t, Keycode["a"], e.Keycode)
	tt.Equal(t, 'a', e.Keychar)
	tt.Equal(t, maskShiftL, e.Mask)

	// Unmapped rawcode: Keycode falls back to the rawcode, no char.
	u := makeKeyEvent(KeyUp, 0xFFF0, 0)
	tt.Equal(t, uint16(0xFFF0), u.Keycode)
	tt.Equal(t, rune(CharUndefined), u.Keychar)
}

// TestDarwinClickedEvent verifies the release of an undragged press derives
// the MouseUp (EVENT_MOUSE_CLICKED) event, while a dragged press (even one
// returned to its start) or an unmatched release does not.
func TestDarwinClickedEvent(t *testing.T) {
	st := &darwinState{}
	lck.Lock()
	mac = st
	lck.Unlock()
	defer func() {
		lck.Lock()
		mac = nil
		lck.Unlock()
	}()

	rel := Event{Kind: MouseHold, Button: 1, X: 5, Y: 6}

	st.clicks.press(1)
	c, ok := clickedEvent(rel)
	tt.Equal(t, true, ok)
	tt.Equal(t, MouseUp, c.Kind)
	tt.Equal(t, uint16(1), c.Button)

	// Unmatched release (the press was already consumed).
	_, ok = clickedEvent(rel)
	tt.Equal(t, false, ok)

	// Dragged press: no click even when released at the press point.
	st.clicks.press(1)
	st.clicks.drag()
	_, ok = clickedEvent(rel)
	tt.Equal(t, false, ok)
}

// TestDarwinEventMask ensures every subscribed type is in the tap mask.
func TestDarwinEventMask(t *testing.T) {
	m := cgEventMask()
	for _, typ := range []uint32{cgEventKeyDown, cgEventFlagsChanged,
		cgEventOtherMouseUp, cgEventScrollWheel, cgEventMouseMoved} {
		tt.Equal(t, true, m&(1<<typ) != 0)
	}
}

// TestDarwinStartEnd exercises the Start/End lifecycle: End() must return
// promptly and close the channel even when it races the tap thread start.
func TestDarwinStartEnd(t *testing.T) {
	for range 3 {
		c := Start()
		tt.NotNil(t, c)
		End()
		_, open := <-c
		tt.Equal(t, false, open)
	}
}
