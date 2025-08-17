package hook

import (
	"testing"
	"time"
)

const (
	TIMEOUT = 2 * time.Second
)

func TestKeyDown(t *testing.T) {
	done := make(chan bool)
	err := Register(KeyDown, []string{"a"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Keycode: Keycode["a"],
			Kind:    KeyDown,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for keydown")
	case <-done:
		t.Log("KeyDown received")
	}
}

func TestKeyDownWithModifier(t *testing.T) {
	done := make(chan bool)
	err := Register(KeyDown, []string{"ctrl", "a"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Keycode: Keycode["ctrl"],
			Kind:    KeyDown,
		}
		time.Sleep(100 * time.Millisecond)
		ch <- Event{
			Keycode: Keycode["a"],
			Kind:    KeyDown,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for keydown")
	case <-done:
		t.Log("KeyDown received")
	}
}

func TestKeyUp(t *testing.T) {
	done := make(chan bool)
	err := Register(KeyUp, []string{"delete"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Keycode: Keycode["delete"],
			Kind:    KeyUp,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for keyup")
	case <-done:
		t.Log("KeyUp received")
	}
}

func TestKeyUpWithModifier(t *testing.T) {
	done := make(chan bool)
	err := Register(KeyUp, []string{"ctrl", "a"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Keycode: Keycode["ctrl"],
			Kind:    KeyUp,
		}
		time.Sleep(100 * time.Millisecond)
		ch <- Event{
			Keycode: Keycode["a"],
			Kind:    KeyUp,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for keyup")
	case <-done:
		t.Log("KeyUp received")
	}
}

func TestMouseDown(t *testing.T) {
	done := make(chan bool)
	err := Register(MouseDown, []string{"mleft"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Button: MouseMap["left"],
			Kind:   MouseDown,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for mousedown")
	case <-done:
		t.Log("MouseDown received")
	}
}

func TestMouseUp(t *testing.T) {
	done := make(chan bool)
	err := Register(MouseUp, []string{"mright"}, func(e Event) {
		done <- true
	})
	if err != nil {
		t.Fatal(err)
	}
	ch := Start()
	defer End()
	Process(ch)

	go func() {
		ch <- Event{
			Button: MouseMap["right"],
			Kind:   MouseUp,
		}
	}()

	select {
	case <-time.After(TIMEOUT):
		t.Fatal("Timeout waiting for mouseup")
	case <-done:
		t.Log("MouseUp received")
	}
}

func TestCombinationsLimit(t *testing.T) {
	// Should fail if more than 4 keys are provided
	err := Register(KeyDown, []string{"ctrl", "a", "b", "c", "d"}, func(e Event) {})
	if err != nil {
		t.Log("That was expected")
	} else {
		t.Fatal("Expected error, got none")
	}

	// Should succeed if less than 4 keys are provided
	err = Register(KeyDown, []string{"ctrl", "a", "b", "c"}, func(e Event) {})
	if err != nil {
		t.Fatal("Expected no error, got", err)
	}
}
