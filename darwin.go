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

// Package hook (macOS pure-Go backend).
//
// This is a *CGo-free* event source built on github.com/ebitengine/purego.
// It dlopen()s the system frameworks (CoreGraphics, CoreFoundation,
// ApplicationServices) and drives a Quartz CGEventTap directly from Go,
// so it needs no C toolchain. Select it at build time with the "purego"
// tag:
//
//	go build -tags purego .
//
// It replaces the default CGo/libuiohook backend (hook_cgo.go) on macOS.
//
// ┌─────────────────────────────────────────────────────────────────────────┐
// │  IMPORTANT — macOS security model                                        │
// │                                                                          │
// │  A session-level CGEventTap only receives events when the process has    │
// │  been granted the Accessibility privilege (System Settings ▸ Privacy &   │
// │  Security ▸ Accessibility). Start() reports an immediate HookDisabled     │
// │  event and stops if the privilege is missing (AXIsProcessTrusted).       │
// │                                                                          │
// │  Unlike the CGo backend this listener is passive (ListenOnly): it        │
// │  observes events but never consumes/rewrites them. Keychar is derived    │
// │  from the static darwin keymap (tables.go) and is not layout/modifier    │
// │  aware (e.g. it reports the lowercase base character).                    │
// └──────────────────────────────────────────────────────────────────────────┘
package hook

import (
	"runtime"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/ebitengine/purego"
)

// System framework dylibs loaded via dlopen.
const (
	frameworkCoreGraphics = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"
	frameworkCoreFound    = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
	frameworkAppServices  = "/System/Library/Frameworks/ApplicationServices.framework/ApplicationServices"
)

// CGEventType values (CGEventTypes.h). NOTE: these are the *native* Quartz
// ids, distinct from gohook's remapped KeyDown/MouseDown constants.
const (
	cgEventLeftMouseDown     uint32 = 1
	cgEventLeftMouseUp       uint32 = 2
	cgEventRightMouseDown    uint32 = 3
	cgEventRightMouseUp      uint32 = 4
	cgEventMouseMoved        uint32 = 5
	cgEventLeftMouseDragged  uint32 = 6
	cgEventRightMouseDragged uint32 = 7
	cgEventKeyDown           uint32 = 10
	cgEventKeyUp             uint32 = 11
	cgEventFlagsChanged      uint32 = 12
	cgEventScrollWheel       uint32 = 22
	cgEventOtherMouseDown    uint32 = 25
	cgEventOtherMouseUp      uint32 = 26
	cgEventOtherMouseDragged uint32 = 27

	cgEventTapDisabledByTimeout   uint32 = 0xFFFFFFFE
	cgEventTapDisabledByUserInput uint32 = 0xFFFFFFFF
)

// CGEventField values (CGEventTypes.h).
const (
	fieldMouseClickState   uint32 = 1
	fieldMouseButtonNumber uint32 = 3
	fieldKeyboardKeycode   uint32 = 9
	fieldScrollWheelDelta1 uint32 = 11
	fieldScrollWheelDelta2 uint32 = 12
)

// CGEventFlags modifier masks (CGEventTypes.h).
const (
	flagAlphaShift  uint64 = 0x00010000 // caps lock
	flagShift       uint64 = 0x00020000
	flagControl     uint64 = 0x00040000
	flagAlternate   uint64 = 0x00080000 // option/alt
	flagCommand     uint64 = 0x00100000
	flagSecondaryFn uint64 = 0x00800000 // fn
)

// gohook virtual modifier masks (mirrors hook/iohook.h MASK_* for Event.Mask).
const (
	maskShiftL   uint16 = 1 << 0
	maskCtrlL    uint16 = 1 << 1
	maskMetaL    uint16 = 1 << 2
	maskAltL     uint16 = 1 << 3
	maskCapsLock uint16 = 1 << 14
)

// CGEventTap creation parameters (CGEventTypes.h).
const (
	cgSessionEventTap          uint32 = 1
	cgHeadInsertEventTap       uint32 = 0
	cgEventTapOptionListenOnly uint32 = 1
)

// libuiohook-compatible scroll directions (matches the CGo backend).
const (
	wheelVertical   uint8 = 3
	wheelHorizontal uint8 = 4
)

// Native darwin virtual keycodes for modifier keys (HIToolbox kVK_*), used to
// turn kCGEventFlagsChanged into discrete KeyDown/KeyUp events.
const (
	vkShift    uint16 = 56
	vkShiftR   uint16 = 60
	vkControl  uint16 = 59
	vkControlR uint16 = 62
	vkCommand  uint16 = 55
	vkCommandR uint16 = 54
	vkOption   uint16 = 58
	vkOptionR  uint16 = 61
	vkCapsLock uint16 = 57
	vkFunction uint16 = 63
)

// Run-loop slice for the poll in darwinLoop (see there for why we poll).
const runLoopSlice = 0.25

// cgPoint mirrors the C CGPoint struct (two CGFloat == float64). purego
// returns it by value from CGEventGetLocation (darwin amd64/arm64).
type cgPoint struct {
	x, y float64
}

// Lazily-bound framework functions. Resolved once by initDarwin().
var (
	cgEventTapCreate            func(tap, place, options uint32, mask uint64, cb, userInfo uintptr) uintptr
	cgEventTapEnable            func(tap uintptr, enable bool)
	cgEventGetIntegerValueField func(event uintptr, field uint32) int64
	cgEventGetFlags             func(event uintptr) uint64
	cgEventGetLocation          func(event uintptr) cgPoint

	cfMachPortCreateRunLoopSource func(allocator, port uintptr, order int) uintptr
	cfMachPortInvalidate          func(port uintptr)
	cfRunLoopGetCurrent           func() uintptr
	cfRunLoopAddSource            func(rl, source, mode uintptr)
	cfRunLoopRemoveSource         func(rl, source, mode uintptr)
	cfRunLoopSourceInvalidate     func(source uintptr)
	cfRunLoopRunInMode            func(mode uintptr, seconds float64, returnAfterSourceHandled bool) int32
	cfRunLoopStop                 func(rl uintptr)
	cfRelease                     func(cf uintptr)

	axIsProcessTrusted func() bool

	kCFRunLoopCommonModes uintptr
	kCFRunLoopDefaultMode uintptr
	cgCallbackPtr         uintptr

	darwinOnce    sync.Once
	darwinInitErr error
)

// darwinState holds the live tap objects so End() can tear them down.
// Guarded by the package-level lck mutex.
type darwinState struct {
	tapPort uintptr
	source  uintptr
	runLoop uintptr

	// held, undragged buttons, used to emit MouseUp (libuiohook
	// EVENT_MOUSE_CLICKED) only for a release without an intervening drag.
	clicks clickTracker
}

var mac *darwinState

// initDarwin resolves every framework symbol and the event callback exactly
// once. Subsequent calls return the cached result.
func initDarwin() error {
	darwinOnce.Do(func() {
		cg, err := purego.Dlopen(frameworkCoreGraphics, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			darwinInitErr = err
			return
		}
		cf, err := purego.Dlopen(frameworkCoreFound, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			darwinInitErr = err
			return
		}
		as, err := purego.Dlopen(frameworkAppServices, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			darwinInitErr = err
			return
		}

		purego.RegisterLibFunc(&cgEventTapCreate, cg, "CGEventTapCreate")
		purego.RegisterLibFunc(&cgEventTapEnable, cg, "CGEventTapEnable")
		purego.RegisterLibFunc(&cgEventGetIntegerValueField, cg, "CGEventGetIntegerValueField")
		purego.RegisterLibFunc(&cgEventGetFlags, cg, "CGEventGetFlags")
		purego.RegisterLibFunc(&cgEventGetLocation, cg, "CGEventGetLocation")

		purego.RegisterLibFunc(&cfMachPortCreateRunLoopSource, cf, "CFMachPortCreateRunLoopSource")
		purego.RegisterLibFunc(&cfMachPortInvalidate, cf, "CFMachPortInvalidate")
		purego.RegisterLibFunc(&cfRunLoopGetCurrent, cf, "CFRunLoopGetCurrent")
		purego.RegisterLibFunc(&cfRunLoopAddSource, cf, "CFRunLoopAddSource")
		purego.RegisterLibFunc(&cfRunLoopRemoveSource, cf, "CFRunLoopRemoveSource")
		purego.RegisterLibFunc(&cfRunLoopSourceInvalidate, cf, "CFRunLoopSourceInvalidate")
		purego.RegisterLibFunc(&cfRunLoopRunInMode, cf, "CFRunLoopRunInMode")
		purego.RegisterLibFunc(&cfRunLoopStop, cf, "CFRunLoopStop")
		purego.RegisterLibFunc(&cfRelease, cf, "CFRelease")

		purego.RegisterLibFunc(&axIsProcessTrusted, as, "AXIsProcessTrusted")

		// kCFRunLoopCommonModes is an exported CFStringRef variable; the symbol
		// address points at the pointer, so dereference once to get the value.
		// The double indirection through &sym keeps `go vet` (unsafeptr) happy:
		// sym is a dlsym'd C address, never a Go pointer.
		sym, err := purego.Dlsym(cf, "kCFRunLoopCommonModes")
		if err != nil {
			darwinInitErr = err
			return
		}
		kCFRunLoopCommonModes = **(**uintptr)(unsafe.Pointer(&sym))

		sym, err = purego.Dlsym(cf, "kCFRunLoopDefaultMode")
		if err != nil {
			darwinInitErr = err
			return
		}
		kCFRunLoopDefaultMode = **(**uintptr)(unsafe.Pointer(&sym))

		cgCallbackPtr = purego.NewCallback(eventCallback)
	})

	return darwinInitErr
}

// Start adds the macOS event tap and returns the event channel.
//
// The optional timeout argument is accepted for API parity with the CGo
// backend but is ignored: this backend is event-driven (it blocks on the
// CFRunLoop) rather than polled.
func Start(tm ...int) chan Event {
	_ = tm

	ev = make(chan Event, 1024)
	asyncon.Store(true)

	sess, done := beginSession()
	go darwinLoop(sess, done)

	return ev
}

// End removes the event tap and resets all hook state.
func End(tm ...int) {
	tm1 := 10
	if len(tm) > 0 {
		tm1 = tm[0]
	}

	asyncon.Store(false)
	done := endSession()

	lck.Lock()
	st := mac
	lck.Unlock()

	// Stopping the run loop wakes darwinLoop promptly; the loop also polls
	// its session, so a stop that lands before the loop is running is not
	// lost. CFRunLoopStop is thread-safe.
	if st != nil && st.runLoop != 0 {
		cfRunLoopStop(st.runLoop)
	}

	waitSession(done, tm1)
	time.Sleep(time.Millisecond * time.Duration(tm1))

	for len(ev) != 0 {
		<-ev
	}
	close(ev)

	resetState()
}

// addEvent: the single-shot *blocking* listener (AddEvent/StopEvent) is a
// CGo/libuiohook-only feature with no direct equivalent here. The supported
// path on this backend is the channel API (Start + Register/Process). A
// non-zero code makes the public AddEvent report failure rather than silently
// pretending to register a hook.
func addEvent(key string) int {
	_ = key
	return -1
}

// StopEvent is a no-op on the pure-Go macOS backend (see addEvent).
func StopEvent() {}

// darwinLoop creates the event tap, wires it into a CFRunLoop and pumps the
// loop until End() stops it. sess is the Start() session it belongs to; done
// is closed once the tap is torn down.
func darwinLoop(sess uint64, done chan struct{}) {
	defer close(done)

	// The tap, its run-loop source and CFRunLoopRun must all live on one OS
	// thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := initDarwin(); err != nil {
		sendFor(sess, Event{Kind: HookDisabled})
		return
	}

	// A session tap only delivers events with the Accessibility privilege.
	if !axIsProcessTrusted() {
		sendFor(sess, Event{Kind: HookDisabled})
		return
	}

	port := cgEventTapCreate(cgSessionEventTap, cgHeadInsertEventTap,
		cgEventTapOptionListenOnly, cgEventMask(), cgCallbackPtr, 0)
	if port == 0 {
		sendFor(sess, Event{Kind: HookDisabled})
		return
	}

	source := cfMachPortCreateRunLoopSource(0, port, 0)
	if source == 0 {
		cfMachPortInvalidate(port)
		cfRelease(port)
		sendFor(sess, Event{Kind: HookDisabled})
		return
	}

	runLoop := cfRunLoopGetCurrent()
	cfRunLoopAddSource(runLoop, source, kCFRunLoopCommonModes)
	cgEventTapEnable(port, true)

	// Publish the state only if this is still the current session, under the
	// same lock End() uses, so an End() that ran before this point is not
	// missed and a later Start() cannot revive this stale tap.
	st := &darwinState{tapPort: port, source: source, runLoop: runLoop}
	lck.Lock()
	live := sess == sessionID
	if live {
		mac = st
	}
	lck.Unlock()

	if live {
		send(Event{Kind: HookEnabled})
	}

	// CFRunLoopStop only affects a loop that is already running, so pump in
	// short slices and re-check the session instead of blocking in
	// CFRunLoopRun.
	for live && isCurrent(sess) {
		cfRunLoopRunInMode(kCFRunLoopDefaultMode, runLoopSlice, false)
	}

	// Teardown; only clear our own state, not that of a newer session.
	lck.Lock()
	if mac == st {
		mac = nil
	}
	lck.Unlock()

	cgEventTapEnable(port, false)
	cfRunLoopRemoveSource(runLoop, source, kCFRunLoopCommonModes)
	cfRunLoopSourceInvalidate(source)
	cfRelease(source)
	cfMachPortInvalidate(port)
	cfRelease(port)

	sendFor(sess, Event{Kind: HookDisabled})
}

// cgEventMask returns the bitmask of Quartz event types we subscribe to.
func cgEventMask() uint64 {
	types := []uint32{
		cgEventKeyDown, cgEventKeyUp, cgEventFlagsChanged,
		cgEventLeftMouseDown, cgEventLeftMouseUp, cgEventLeftMouseDragged,
		cgEventRightMouseDown, cgEventRightMouseUp, cgEventRightMouseDragged,
		cgEventOtherMouseDown, cgEventOtherMouseUp, cgEventOtherMouseDragged,
		cgEventMouseMoved, cgEventScrollWheel,
	}

	var mask uint64
	for _, t := range types {
		mask |= 1 << t
	}
	return mask
}

// eventCallback is the C CGEventTapCallBack. It runs on the run-loop thread.
// Signature: CGEventRef (CGEventTapProxy, CGEventType, CGEventRef, void *).
func eventCallback(proxy, typ, event, refcon uintptr) uintptr {
	_, _ = proxy, refcon

	t := uint32(typ)

	// The OS disables a tap on timeout/user-input; just re-enable it.
	if t == cgEventTapDisabledByTimeout || t == cgEventTapDisabledByUserInput {
		lck.Lock()
		st := mac
		lck.Unlock()
		if st != nil {
			cgEventTapEnable(st.tapPort, true)
		}
		return event
	}

	if !asyncon.Load() {
		return event
	}

	dispatchEvent(t, event)

	// ListenOnly taps ignore the return value, but pass the event through.
	return event
}

// dispatchEvent translates a native Quartz event into gohook Events and
// sends them.
func dispatchEvent(t uint32, event uintptr) {
	switch t {
	case cgEventKeyDown:
		send(keyEvent(KeyDown, event))
	case cgEventKeyUp:
		send(keyEvent(KeyUp, event))
	case cgEventFlagsChanged:
		send(modifierEvent(event))
	case cgEventLeftMouseDown, cgEventRightMouseDown, cgEventOtherMouseDown:
		send(buttonEvent(t, event))
	case cgEventLeftMouseUp, cgEventRightMouseUp, cgEventOtherMouseUp:
		// libuiohook parity: MOUSE_RELEASED (MouseHold) always, followed by
		// MOUSE_CLICKED (MouseUp) when the press was not dragged.
		e := buttonEvent(t, event)
		send(e)
		if released, ok := clickedEvent(e); ok {
			send(released)
		}
	case cgEventLeftMouseDragged, cgEventRightMouseDragged, cgEventOtherMouseDragged:
		lck.Lock()
		if mac != nil {
			mac.clicks.drag()
		}
		lck.Unlock()
		if e, ok := mouseEvent(t, event); ok {
			send(e)
		}
	case cgEventMouseMoved, cgEventScrollWheel:
		if e, ok := mouseEvent(t, event); ok {
			send(e)
		}
	}
}

// keyEvent builds a key Event from a native event of a given kind.
func keyEvent(kind uint8, event uintptr) Event {
	raw := uint16(cgEventGetIntegerValueField(event, fieldKeyboardKeycode))
	return makeKeyEvent(kind, raw, cgEventGetFlags(event))
}

// modifierEvent turns a kCGEventFlagsChanged event into a discrete KeyDown or
// KeyUp by checking whether the relevant modifier flag is now set.
func modifierEvent(event uintptr) Event {
	raw := uint16(cgEventGetIntegerValueField(event, fieldKeyboardKeycode))
	flags := cgEventGetFlags(event)

	kind := uint8(KeyUp)
	switch raw {
	case vkShift, vkShiftR:
		if flags&flagShift != 0 {
			kind = KeyDown
		}
	case vkControl, vkControlR:
		if flags&flagControl != 0 {
			kind = KeyDown
		}
	case vkCommand, vkCommandR:
		if flags&flagCommand != 0 {
			kind = KeyDown
		}
	case vkOption, vkOptionR:
		if flags&flagAlternate != 0 {
			kind = KeyDown
		}
	case vkCapsLock:
		if flags&flagAlphaShift != 0 {
			kind = KeyDown
		}
	case vkFunction:
		if flags&flagSecondaryFn != 0 {
			kind = KeyDown
		}
	default:
		kind = KeyDown
	}

	return makeKeyEvent(kind, raw, flags)
}

// makeKeyEvent fills in Keycode/Keychar from the static darwin keymap so that
// Register() (which matches against Keycode[name]) works identically to the
// CGo backend. Mirrors the Wayland backend's keyEvent.
func makeKeyEvent(kind uint8, raw uint16, flags uint64) Event {
	e := Event{
		Kind:    kind,
		Rawcode: raw,
		Keychar: CharUndefined,
		Mask:    maskFromFlags(flags),
	}

	name, ok := rawToKeyDarwin[raw]
	if !ok {
		e.Keycode = raw
		return e
	}

	if kc, ok := Keycode[name]; ok {
		e.Keycode = kc
	} else {
		e.Keycode = raw
	}

	// A single-rune name is a printable character (e.g. "a", "1", ";").
	if utf8.RuneCountInString(name) == 1 {
		r, _ := utf8.DecodeRuneInString(name)
		e.Keychar = r
	}

	return e
}

// buttonEvent builds a MouseDown (pressed) or MouseHold (released) Event and
// records the press for clickedEvent.
func buttonEvent(t uint32, event uintptr) Event {
	loc := cgEventGetLocation(event)
	e := Event{
		Kind:   MouseDown,
		X:      int16(loc.x),
		Y:      int16(loc.y),
		Mask:   maskFromFlags(cgEventGetFlags(event)),
		Clicks: uint16(cgEventGetIntegerValueField(event, fieldMouseClickState)),
	}

	switch t {
	case cgEventLeftMouseDown, cgEventLeftMouseUp:
		e.Button = MouseMap["left"]
	case cgEventRightMouseDown, cgEventRightMouseUp:
		e.Button = MouseMap["right"]
	default:
		// kCGMouseEventButtonNumber is 0-based (2 == middle).
		e.Button = uint16(cgEventGetIntegerValueField(event, fieldMouseButtonNumber)) + 1
	}

	switch t {
	case cgEventLeftMouseUp, cgEventRightMouseUp, cgEventOtherMouseUp:
		e.Kind = MouseHold
	default:
		lck.Lock()
		if mac != nil {
			mac.clicks.press(e.Button)
		}
		lck.Unlock()
	}

	return e
}

// clickedEvent derives the MouseUp (libuiohook EVENT_MOUSE_CLICKED) Event
// from a MouseHold when it completes an undragged press of the same button
// (CGo darwin parity: hook/darwin/hook_c.h suppresses clicks after a drag).
func clickedEvent(released Event) (Event, bool) {
	lck.Lock()
	clicked := mac != nil && mac.clicks.release(released.Button)
	lck.Unlock()
	if !clicked {
		return Event{}, false
	}

	released.Kind = MouseUp
	return released, true
}

// mouseEvent builds a mouse Event from a native motion/scroll event.
func mouseEvent(t uint32, event uintptr) (Event, bool) {
	loc := cgEventGetLocation(event)
	x, y := int16(loc.x), int16(loc.y)
	mask := maskFromFlags(cgEventGetFlags(event))

	switch t {
	case cgEventMouseMoved:
		return Event{Kind: MouseMove, X: x, Y: y, Mask: mask}, true
	case cgEventLeftMouseDragged, cgEventRightMouseDragged, cgEventOtherMouseDragged:
		return Event{Kind: MouseDrag, X: x, Y: y, Mask: mask}, true
	case cgEventScrollWheel:
		return wheelEvent(event, x, y, mask), true
	}

	return Event{}, false
}

// wheelEvent builds a scroll-wheel Event.
func wheelEvent(event uintptr, x, y int16, mask uint16) Event {
	d1 := cgEventGetIntegerValueField(event, fieldScrollWheelDelta1)
	d2 := cgEventGetIntegerValueField(event, fieldScrollWheelDelta2)
	return wheelFromDeltas(d1, d2, x, y, mask)
}

// wheelFromDeltas builds a MouseWheel Event from the two Quartz axis deltas.
func wheelFromDeltas(d1, d2 int64, x, y int16, mask uint16) Event {
	// libuiohook parity: rotation = delta * -1, so scrolling up yields
	// WheelUp (-1) like the CGo, X11 and Windows backends.
	dir := wheelVertical
	rot := int32(-d1)
	if d1 == 0 && d2 != 0 {
		dir = wheelHorizontal
		rot = int32(-d2)
	}

	amt := rot
	if amt < 0 {
		amt = -amt
	}

	return Event{
		Kind:      MouseWheel,
		X:         x,
		Y:         y,
		Clicks:    1,
		Amount:    uint16(amt),
		Rotation:  rot,
		Direction: dir,
		Mask:      mask,
	}
}

// maskFromFlags maps Quartz CGEventFlags to gohook's virtual modifier mask.
func maskFromFlags(flags uint64) uint16 {
	var m uint16
	if flags&flagShift != 0 {
		m |= maskShiftL
	}
	if flags&flagControl != 0 {
		m |= maskCtrlL
	}
	if flags&flagCommand != 0 {
		m |= maskMetaL
	}
	if flags&flagAlternate != 0 {
		m |= maskAltL
	}
	if flags&flagAlphaShift != 0 {
		m |= maskCapsLock
	}
	return m
}

// send timestamps and pushes an event onto the global channel. It drops the
// event (rather than blocking the run-loop thread) if no consumer keeps up and
// the buffer is full. The recover guards the small shutdown window where End()
// may have closed ev while a tap callback is still in flight.
func send(e Event) {
	if !asyncon.Load() {
		return
	}
	defer func() { _ = recover() }() // ev closed by End(): drop silently

	e.When = time.Now()
	select {
	case ev <- e:
	default:
		// channel full: drop to avoid stalling the event tap.
	}
}
