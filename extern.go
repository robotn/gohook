package hook

/*


// #include "event/hook_async.h"
*/
import "C"
import (
	"encoding/json"
	"log"
	"time"
)

//export go_send
func go_send(s *C.char) {
	str := []byte(C.GoString(s))
	out := Event{}
	err := json.Unmarshal(str, &out)
	if err != nil{
		log.Fatal(err)
	}
	out.When = time.Now() //at least it's consistent
	if err != nil {
		log.Fatal(err)
	}
	//todo: maybe make non-bloking
	ev <- out
}