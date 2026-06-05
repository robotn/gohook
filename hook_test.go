package hook

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/vcaesar/tt"
)

func TestAdd(t *testing.T) {
	fmt.Println("hook test...")

	e := Start()
	tt.NotNil(t, e)
}

func TestKey(t *testing.T) {
	k := RawcodeToKeychar(0)
	switch runtime.GOOS {
	case "darwin":
		tt.Equal(t, "a", k)
	case "windows":
		tt.Equal(t, "error", k)
	default: // linux: raw2keyLinux has no entry for rawcode 0
		tt.Equal(t, "", k)
	}

	r := KeycharToRawcode("error")
	tt.Equal(t, 0, r)
}
