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
