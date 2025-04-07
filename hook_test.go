package hook

import (
	"fmt"
	"testing"

	"github.com/vcaesar/tt"
)

func TestAdd(t *testing.T) {
	fmt.Println("hook test...")

	e := Start()
	tt.NotNil(t, e)
}

func TestKey(t *testing.T) {
	k := RawcodetoKeychar(0)
	tt.Equal(t, "error", k)

	r := KeychartoRawcode("error")
	tt.Equal(t, 0, r)
}

func TestUnregister(t *testing.T) {
	// Setup for tests
	setupTest := func() {
		// Reset all global variables to ensure clean state
		pressed = make(map[uint16]bool, 256)
		used = []int{}
		keys = map[int][]uint16{}
		cbs = map[int]func(Event){}
		events = map[uint8][]int{}
	}

	// Mock callback function
	mockCallback := func(Event) {}

	t.Run("UnregisterExistingHook", func(t *testing.T) {
		setupTest()
		Register(KeyDown, []string{"a", "b", "c"}, mockCallback)

		if len(events[KeyDown]) != 1 {
			t.Fatalf("Registration failed, expected 1 event, got %d", len(events[KeyDown]))
		}

		result := Unregister(KeyDown, []string{"a", "b", "c"})

		if !result {
			t.Error("Unregister returned false for existing hook")
		}

		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed to remove hook, %d events remain", len(events[KeyDown]))
		}
	})

	t.Run("UnregisterNonExistentHook", func(t *testing.T) {
		setupTest()

		result := Unregister(KeyDown, []string{"x", "y", "z"})

		if result {
			t.Error("Unregister returned true for non-existent hook")
		}
	})

	t.Run("UnregisterWithDifferentOrder", func(t *testing.T) {
		setupTest()

		Register(KeyDown, []string{"ctrl", "alt", "del"}, mockCallback)

		result := Unregister(KeyDown, []string{"alt", "ctrl", "del"})

		if !result {
			t.Error("Unregister failed when key order was different")
		}

		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed to remove hook with different key order, %d events remain", len(events[KeyDown]))
		}
	})

	t.Run("UnregisterSpecificHook", func(t *testing.T) {
		setupTest()

		Register(KeyDown, []string{"a", "b"}, mockCallback)
		Register(KeyDown, []string{"c", "d"}, mockCallback)
		Register(KeyUp, []string{"a", "b"}, mockCallback)

		result := Unregister(KeyDown, []string{"a", "b"})

		if !result {
			t.Error("Failed to unregister specific hook")
		}

		if len(events[KeyDown]) != 1 {
			t.Errorf("Expected 1 KeyDown event to remain, got %d", len(events[KeyDown]))
		}

		if len(events[KeyUp]) != 1 {
			t.Errorf("Expected 1 KeyUp event to remain, got %d", len(events[KeyUp]))
		}
	})

	t.Run("RegisterAfterUnregister", func(t *testing.T) {
		setupTest()

		Register(KeyDown, []string{"a", "b"}, mockCallback)

		// origKeyIndex := events[KeyDown][0]

		Unregister(KeyDown, []string{"a", "b"})

		Register(KeyDown, []string{"x", "y"}, mockCallback)

		if len(events[KeyDown]) != 1 {
			t.Fatalf("Re-registration failed, expected 1 event, got %d", len(events[KeyDown]))
		}

		newKeyIndex := events[KeyDown][0]

		keyAFound := false
		keyXFound := false

		for _, k := range keys[newKeyIndex] {
			if k == Keycode["a"] {
				keyAFound = true
			}
			if k == Keycode["x"] {
				keyXFound = true
			}
		}

		if keyAFound {
			t.Error("Found old key 'a' in new registration")
		}

		if !keyXFound {
			t.Error("New key 'x' not found in registration")
		}
	})
	t.Run("ReregisterSameHook", func(t *testing.T) {
		setupTest()

		firstCallbackCalled := false
		secondCallbackCalled := false

		firstCallback := func(Event) {
			firstCallbackCalled = true
		}

		secondCallback := func(Event) {
			secondCallbackCalled = true
		}

		Register(KeyDown, []string{"ctrl", "alt", "del"}, firstCallback)

		if len(events[KeyDown]) != 1 {
			t.Fatalf("First registration failed")
		}
		originalHandlerIndex := events[KeyDown][0]

		if len(keys[originalHandlerIndex]) != 3 {
			t.Errorf("Expected 3 keys, got %d", len(keys[originalHandlerIndex]))
		}

		testEvent := Event{Kind: KeyDown}
		for _, keyIndex := range events[KeyDown] {
			for _, keycode := range keys[keyIndex] {
				pressed[keycode] = true
			}

			if allPressed(pressed, keys[keyIndex]...) {
				cbs[keyIndex](testEvent)
			}
		}

		if !firstCallbackCalled {
			t.Error("First callback was not called")
		}

		success := Unregister(KeyDown, []string{"ctrl", "alt", "del"})
		if !success {
			t.Fatal("Failed to unregister hook")
		}

		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed, %d events still registered", len(events[KeyDown]))
		}

		Register(KeyDown, []string{"ctrl", "alt", "del"}, secondCallback)

		if len(events[KeyDown]) != 1 {
			t.Fatalf("Re-registration failed")
		}
		newHandlerIndex := events[KeyDown][0]

		firstCallbackCalled = false
		secondCallbackCalled = false

		for _, keyIndex := range events[KeyDown] {
			for _, keycode := range keys[keyIndex] {
				pressed[keycode] = true
			}

			if allPressed(pressed, keys[keyIndex]...) {
				cbs[keyIndex](testEvent)
			}
		}

		if firstCallbackCalled {
			t.Error("First callback was incorrectly called after re-registration")
		}

		if !secondCallbackCalled {
			t.Error("Second callback was not called after re-registration")
		}

		t.Logf("Original handler index: %d, New handler index: %d", originalHandlerIndex, newHandlerIndex)
		if originalHandlerIndex == newHandlerIndex {
			t.Log("Handler index was reused (this is implementation-dependent)")
		} else {
			t.Log("A new handler index was assigned")
		}
	})
}

func TestEqualKeySlices(t *testing.T) {
	t.Run("EqualSlicesSameOrder", func(t *testing.T) {
		a := []uint16{1, 2, 3}
		b := []uint16{1, 2, 3}

		if !equalKeySlices(a, b) {
			t.Error("equalKeySlices returned false for identical slices")
		}
	})

	t.Run("EqualSlicesDifferentOrder", func(t *testing.T) {
		a := []uint16{1, 2, 3}
		b := []uint16{3, 1, 2}

		if !equalKeySlices(a, b) {
			t.Error("equalKeySlices returned false for slices with same elements in different order")
		}
	})

	t.Run("UnequalSlicesDifferentLength", func(t *testing.T) {
		a := []uint16{1, 2, 3}
		b := []uint16{1, 2}

		if equalKeySlices(a, b) {
			t.Error("equalKeySlices returned true for slices with different lengths")
		}
	})

	t.Run("UnequalSlicesDifferentElements", func(t *testing.T) {
		a := []uint16{1, 2, 3}
		b := []uint16{1, 2, 4}

		if equalKeySlices(a, b) {
			t.Error("equalKeySlices returned true for slices with different elements")
		}
	})

	t.Run("SlicesWithDuplicates", func(t *testing.T) {
		a := []uint16{1, 2, 2, 3}
		b := []uint16{3, 1, 2, 2}

		if !equalKeySlices(a, b) {
			t.Error("equalKeySlices returned false for slices with same elements including duplicates")
		}

		c := []uint16{1, 2, 3, 3}

		if equalKeySlices(a, c) {
			t.Error("equalKeySlices returned true for slices with different duplicate patterns")
		}
	})
}
