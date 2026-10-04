// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

//go:build wayland || (darwin && purego) || (windows && purego) || (linux && purego)

package hook

import (
	"testing"
	"time"

	"github.com/vcaesar/tt"
)

// TestSessionGeneration verifies a loop from an ended session stays stale
// after a later Start begins a new one, and never sends into its channel.
func TestSessionGeneration(t *testing.T) {
	old, oldDone := beginSession()
	tt.Equal(t, true, isCurrent(old))

	tt.Equal(t, true, endSession() == oldDone)
	tt.Equal(t, false, isCurrent(old))

	cur, _ := beginSession()
	tt.Equal(t, false, isCurrent(old)) // not revived by the new session
	tt.Equal(t, true, isCurrent(cur))

	ev = make(chan Event, 4)
	asyncon.Store(true)
	sendFor(old, Event{Kind: HookEnabled})
	sendFor(cur, Event{Kind: HookEnabled})
	asyncon.Store(false)
	tt.Equal(t, 1, len(ev))

	endSession()
}

// TestWaitSession verifies End's wait returns once the loop is done, and is
// bounded when the loop never finishes.
func TestWaitSession(t *testing.T) {
	waitSession(nil, 0) // no session: returns immediately

	done := make(chan struct{})
	close(done)
	start := time.Now()
	waitSession(done, 0)
	tt.Equal(t, true, time.Since(start) < 100*time.Millisecond)

	start = time.Now()
	waitSession(make(chan struct{}), 0)
	tt.Equal(t, true, time.Since(start) >= time.Second)
}

// TestClickTracker verifies a click needs a matching press of the same
// button with no drag in between.
func TestClickTracker(t *testing.T) {
	var c clickTracker

	tt.Equal(t, false, c.release(1)) // unmatched release

	c.press(1)
	tt.Equal(t, true, c.release(1))
	tt.Equal(t, false, c.release(1)) // consumed

	c.press(1)
	tt.Equal(t, false, c.release(3)) // other button
	tt.Equal(t, true, c.release(1))

	c.press(1)
	c.drag()
	tt.Equal(t, false, c.release(1)) // dragged

	c.press(40) // out of range: ignored, never clicks
	tt.Equal(t, false, c.release(40))
}
