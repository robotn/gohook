// Copyright 2016 The go-vgo Project Developers. See the COPYRIGHT
// file at the top-level directory of this distribution and at
// https://github.com/go-vgo/robotgo/blob/master/LICENSE
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// http://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or http://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
#ifndef goevent_h
#define goevent_h
#ifdef HAVE_CONFIG_H
	#include <config.h>
#endif

#include <stdlib.h>
#include "pub.h"
#include "../chan/eb_chan.h"

void dispatch_proc(iohook_event * const event);

eb_chan events;

void go_send(char*);
void go_sleep(void);

bool sending = false;

void startev(){
	// puts("start_ev");
    events = eb_chan_create(1024);
    eb_chan_retain(events);
	sending = true;
	// add_event("q");	
	hook_set_logger(&loggerProc);
	hook_set_dispatch_proc(&dispatch_proc);
	hook_run();
	// puts("done_ev");
}

void pollEv(){
    if(events == NULL) return;
    for(;eb_chan_buf_len(events)!=0;){
        char* tmp;
        if(eb_chan_try_recv(events,(const void**) &tmp) == eb_chan_res_ok){
            go_send(tmp);
            free(tmp);
        }else{
            //
        }
    }
}

void endPoll(){
	sending = false;
	pollEv();//remove last things from channel
	eb_chan_release(events);
}

void dispatch_proc(iohook_event * const event) {
    if(!sending) return;
//leaking memory? hope not
    char* buffer = calloc(200,sizeof(char));

	switch (event->type) {
	    case EVENT_HOOK_ENABLED:
	    case EVENT_HOOK_DISABLED:
	        sprintf(buffer,"{\"id\":%i,\"time\":%" PRIu64 ",\"mask\":%hu,\"reserved\":%hu}",
	        event->type, event->time, event->mask,event->reserved);
	    break;//send it?
		case EVENT_KEY_PRESSED:
		case EVENT_KEY_RELEASED:
		case EVENT_KEY_TYPED:
           sprintf(buffer,
                "{\"id\":%i,\"time\":%" PRIu64 ",\"mask\":%hu,\"reserved\":%hu,\"keycode\":%hu,\"rawcode\":%hu,\"keychar\":%hu}",
                event->type, event->time, event->mask,event->reserved,
                event->data.keyboard.keycode,
                event->data.keyboard.rawcode,
                event->data.keyboard.keychar);
            break;
		case EVENT_MOUSE_PRESSED:
		case EVENT_MOUSE_RELEASED:
		case EVENT_MOUSE_CLICKED:
		case EVENT_MOUSE_MOVED:
		case EVENT_MOUSE_DRAGGED:
			sprintf(buffer,
				"{\"id\":%i,\"time\":%" PRIu64 ",\"mask\":%hu,\"reserved\":%hu,\"x\":%hd,\"y\":%hd,\"button\":%u,\"clicks\":%u}",
				event->type, event->time, event->mask,event->reserved,
				event->data.mouse.x,
				event->data.mouse.y,
				event->data.mouse.button,
				event->data.mouse.clicks);
			break;
		case EVENT_MOUSE_WHEEL:
			sprintf(buffer,
				"{\"id\":%i,\"time\":%" PRIu64 ",\"mask\":%hu,\"reserved\":%hu,\"clicks\":%hu,\"x\":%hd,\"y\":%hd,\"type\":%hu,\"ammount\":%hu,\"rotation\":%hd,\"direction\":%hu}",
				event->type, event->time, event->mask, event->reserved,
				event->data.wheel.clicks,
				event->data.wheel.x,
				event->data.wheel.y,
   				event->data.wheel.type,
   				event->data.wheel.amount,
   				event->data.wheel.rotation,
   				event->data.wheel.direction);
			break;
		default:
		    fprintf(stderr,"\nError on file: %s, unusual event->type: %i\n",__FILE__,event->type);
			return;
	}
	//to-do remove this for
	for(int i = 0; i < 5; i++){
        switch(eb_chan_try_send(events,buffer)){ //never block the hook callback
            case eb_chan_res_ok:
            i=5;
            break;
            default:
            if (i == 4) {//let's not leak memory
                free(buffer);
            }
            continue;
        }
    }

	// fprintf(stdout, "----%s\n",	 buffer);
}

int stop_event(){
	int status = hook_stop();
	switch (status) {
		// System level errors.
		case IOHOOK_ERROR_OUT_OF_MEMORY:
			loggerProc(LOG_LEVEL_ERROR, "Failed to allocate memory. (%#X)", status);
			break;

		case IOHOOK_ERROR_X_RECORD_GET_CONTEXT:
			// NOTE This is the only platform specific error that occurs on hook_stop().
			loggerProc(LOG_LEVEL_ERROR, "Failed to get XRecord context. (%#X)", status);
			break;

		// Default error.
		case IOHOOK_FAILURE:
			default:
			// loggerProc(LOG_LEVEL_ERROR, "An unknown hook error occurred. (%#X)", status);
			break;
	}

	return status;
}

#endif