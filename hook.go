// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.

package hook

/*
#cgo darwin CFLAGS: -x objective-c  -Wno-deprecated-declarations
#cgo darwin LDFLAGS: -framework Cocoa

#cgo linux CFLAGS:-I/usr/src
#cgo linux LDFLAGS: -L/usr/src -lX11 -lXtst
#cgo linux LDFLAGS: -lX11-xcb -lxcb -lxcb-xkb -lxkbcommon -lxkbcommon-x11
//#cgo windows LDFLAGS: -lgdi32 -luser32

// #include "event/hook_async.h"
#include "chan/src/chan.h"
#include "event/goEvent.h"

void go_send(char*);
void go_sleep(void);
extern chan_t* events;

void startev(){
	events = chan_init(1024);
	add_event("q");
}
bool done = false;
void pollEv(){
	while(!done){
		for(int i=chan_size(events); i >0;i--){
			char* tmp;
			chan_recv(events,(void**) &tmp);
			go_send(tmp);
		}
		//go_sleep();
	}
}

void endPoll(){
	done = true;
}

*/
import "C"

import (
	// 	"fmt"
	"unsafe"
)

var ev chan string = make(chan string,128)

// AddEvent add event listener
func AddEvent(key string) int {
	cs := C.CString(key)
	defer C.free(unsafe.Pointer(cs))

	eve := C.add_event(cs)
	geve := int(eve)

	return geve
}

func StartEvent() chan string{
	C.startev()
	go C.pollEv()
	return ev
}


// StopEvent stop event listener
func StopEvent() {
	C.endPoll()
	C.stop_event()
	C.chan_close(C.events);
}
