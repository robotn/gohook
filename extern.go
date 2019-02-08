package hook

/*


// #include "event/hook_async.h"
*/
import "C"
import "time"

//export go_send
func go_send(s *C.char) {
	str := C.GoString(s)
	ev <- str
}

//export go_sleep
func go_sleep(){
	//todo: find smallest time that does not destroy the cpu utilization
	time.Sleep(time.Millisecond*50)
}