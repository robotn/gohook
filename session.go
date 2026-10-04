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

import "time"

// Session bookkeeping shared by the pure-Go backends. Start and End both bump
// sessionID, so a backend loop is current only while its id still matches:
// a loop that outlives its session (End ran during its setup, then a new
// Start) never goes live, publishes state or sends into the new channel.
var (
	sessionID   uint64        // guarded by lck
	sessionDone chan struct{} // closed when the current loop exits; guarded by lck
)

// beginSession starts a new session for Start and returns its id and the
// channel its loop must close on exit.
func beginSession() (uint64, chan struct{}) {
	done := make(chan struct{})

	lck.Lock()
	sessionID++
	sess := sessionID
	sessionDone = done
	lck.Unlock()

	return sess, done
}

// endSession invalidates the current session for End and returns the done
// channel of its loop (nil if Start was never called).
func endSession() chan struct{} {
	lck.Lock()
	sessionID++
	done := sessionDone
	sessionDone = nil
	lck.Unlock()

	return done
}

// isCurrent reports whether sess is still the live session.
func isCurrent(sess uint64) bool {
	lck.Lock()
	defer lck.Unlock()
	return sess == sessionID
}

// sendFor sends e only while sess is the live session.
func sendFor(sess uint64, e Event) {
	if isCurrent(sess) {
		send(e)
	}
}

// waitSession waits for the ended loop to finish its teardown, so End
// returns with the native hook removed. The wait is bounded (at least the
// caller's grace period tm ms, minimum 1s) because a loop still in setup,
// e.g. connecting to a display server, may take a while to notice.
func waitSession(done chan struct{}, tm int) {
	if done == nil {
		return
	}

	timeout := time.Duration(tm) * time.Millisecond
	if timeout < time.Second {
		timeout = time.Second
	}

	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// clickTracker decides libuiohook's EVENT_MOUSE_CLICKED (MouseUp): a release
// is a click only for a matching press of the same button with no drag in
// between, as in the CGo darwin/x11 backends. One bit per button.
type clickTracker uint32

func (c *clickTracker) press(btn uint16) {
	if btn < 32 {
		*c |= 1 << btn
	}
}

// drag cancels the click of every held button.
func (c *clickTracker) drag() { *c = 0 }

// release reports whether releasing btn completes a click.
func (c *clickTracker) release(btn uint16) bool {
	if btn >= 32 {
		return false
	}
	bit := clickTracker(1) << btn
	clicked := *c&bit != 0
	*c &^= bit
	return clicked
}
