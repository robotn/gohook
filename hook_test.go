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
		// Register a hook
		Register(KeyDown, []string{"a", "b", "c"}, mockCallback)

		// Verify registration worked
		if len(events[KeyDown]) != 1 {
			t.Fatalf("Registration failed, expected 1 event, got %d", len(events[KeyDown]))
		}

		// Unregister the hook
		result := Unregister(KeyDown, []string{"a", "b", "c"})

		// Check return value
		if !result {
			t.Error("Unregister returned false for existing hook")
		}

		// Verify hook was removed
		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed to remove hook, %d events remain", len(events[KeyDown]))
		}
	})

	t.Run("UnregisterNonExistentHook", func(t *testing.T) {
		setupTest()

		// Unregister a hook that doesn't exist
		result := Unregister(KeyDown, []string{"x", "y", "z"})

		// Should return false
		if result {
			t.Error("Unregister returned true for non-existent hook")
		}
	})

	t.Run("UnregisterWithDifferentOrder", func(t *testing.T) {
		setupTest()

		// Register a hook
		Register(KeyDown, []string{"ctrl", "alt", "del"}, mockCallback)

		// Unregister with different order
		result := Unregister(KeyDown, []string{"alt", "ctrl", "del"})

		// Should succeed despite different order
		if !result {
			t.Error("Unregister failed when key order was different")
		}

		// Verify hook was removed
		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed to remove hook with different key order, %d events remain", len(events[KeyDown]))
		}
	})

	t.Run("UnregisterSpecificHook", func(t *testing.T) {
		setupTest()

		// Register multiple hooks
		Register(KeyDown, []string{"a", "b"}, mockCallback)
		Register(KeyDown, []string{"c", "d"}, mockCallback)
		Register(KeyUp, []string{"a", "b"}, mockCallback)

		// Unregister one specific hook
		result := Unregister(KeyDown, []string{"a", "b"})

		// Should succeed
		if !result {
			t.Error("Failed to unregister specific hook")
		}

		// Verify only the specific hook was removed
		if len(events[KeyDown]) != 1 {
			t.Errorf("Expected 1 KeyDown event to remain, got %d", len(events[KeyDown]))
		}

		if len(events[KeyUp]) != 1 {
			t.Errorf("Expected 1 KeyUp event to remain, got %d", len(events[KeyUp]))
		}
	})

	t.Run("RegisterAfterUnregister", func(t *testing.T) {
		setupTest()

		// Register a hook
		Register(KeyDown, []string{"a", "b"}, mockCallback)

		// Check original state
		// origKeyIndex := events[KeyDown][0]

		// Unregister
		Unregister(KeyDown, []string{"a", "b"})

		// Register a new hook
		Register(KeyDown, []string{"x", "y"}, mockCallback)

		// Verify new registration worked
		if len(events[KeyDown]) != 1 {
			t.Fatalf("Re-registration failed, expected 1 event, got %d", len(events[KeyDown]))
		}

		// Check that keys were properly registered
		newKeyIndex := events[KeyDown][0]

		keyAFound := false
		keyXFound := false

		for _, k := range keys[newKeyIndex] {
			// Check if key matches 'a' keycode
			if k == Keycode["a"] {
				keyAFound = true
			}
			// Check if key matches 'x' keycode
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

		// Mock callback functions to verify they're being called
		firstCallbackCalled := false
		secondCallbackCalled := false

		firstCallback := func(Event) {
			firstCallbackCalled = true
		}

		secondCallback := func(Event) {
			secondCallbackCalled = true
		}

		// First registration
		Register(KeyDown, []string{"ctrl", "alt", "del"}, firstCallback)

		// Store the original handler index
		if len(events[KeyDown]) != 1 {
			t.Fatalf("First registration failed")
		}
		originalHandlerIndex := events[KeyDown][0]

		// Verify first registration worked
		if len(keys[originalHandlerIndex]) != 3 {
			t.Errorf("Expected 3 keys, got %d", len(keys[originalHandlerIndex]))
		}

		// Simulate an event to verify the first callback works
		testEvent := Event{Kind: KeyDown}
		for _, keyIndex := range events[KeyDown] {
			// Simulate all keys being pressed
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

		// Unregister the hook
		success := Unregister(KeyDown, []string{"ctrl", "alt", "del"})
		if !success {
			t.Fatal("Failed to unregister hook")
		}

		// Verify unregistration worked
		if len(events[KeyDown]) != 0 {
			t.Errorf("Unregister failed, %d events still registered", len(events[KeyDown]))
		}

		// Re-register the same hook but with a different callback
		Register(KeyDown, []string{"ctrl", "alt", "del"}, secondCallback)

		// Verify re-registration worked
		if len(events[KeyDown]) != 1 {
			t.Fatalf("Re-registration failed")
		}
		newHandlerIndex := events[KeyDown][0]

		// Reset callbacks tracking
		firstCallbackCalled = false
		secondCallbackCalled = false

		// Simulate an event again
		for _, keyIndex := range events[KeyDown] {
			// Simulate all keys being pressed
			for _, keycode := range keys[keyIndex] {
				pressed[keycode] = true
			}

			if allPressed(pressed, keys[keyIndex]...) {
				cbs[keyIndex](testEvent)
			}
		}

		// Verify the correct callback was called
		if firstCallbackCalled {
			t.Error("First callback was incorrectly called after re-registration")
		}

		if !secondCallbackCalled {
			t.Error("Second callback was not called after re-registration")
		}

		// Verify the handler index was reused or is different
		t.Logf("Original handler index: %d, New handler index: %d", originalHandlerIndex, newHandlerIndex)
		if originalHandlerIndex == newHandlerIndex {
			// This is not necessarily an error, but good to verify the implementation's behavior
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
