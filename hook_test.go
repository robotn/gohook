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
	if runtime.GOOS == "darwin" {
		tt.Equal(t, "a", k)
	} else {
		tt.Equal(t, "error", k)
	}

	r := KeycharToRawcode("error")
	tt.Equal(t, 0, r)
}
