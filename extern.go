package hook

/*


// #include "event/hook_async.h"
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

//export go_send
func go_send(s *C.char) {
	str := []byte(C.GoString(s))
	fmt.Println(string(str))
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

////export go_sleep
//func go_sleep(){
//
//	time.Sleep(time.Millisecond*50)
//}
