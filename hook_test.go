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
