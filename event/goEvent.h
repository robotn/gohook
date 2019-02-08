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
#include "../chan/src/chan.h"

chan_t * events = NULL;

void dispatch_proc(iohook_event * const event) {
	char buffer[256] = { 0 };
	size_t length = snprintf(buffer, sizeof(buffer),
			"{id:%i,when:%" PRIu64 ",mask=0x%X",
			event->type, event->time, event->mask);

	switch (event->type) {
		case EVENT_KEY_PRESSED:
		case EVENT_KEY_RELEASED:
		case EVENT_KEY_TYPED:
            length = snprintf(buffer + length, sizeof(buffer) - length,
                ",keycode:%hu,rawcode:%hu,keychar:%hu}",
                event->data.keyboard.keycode,
                event->data.keyboard.rawcode,
                event->data.keyboard.keychar);
            break;
		case EVENT_MOUSE_PRESSED:
		case EVENT_MOUSE_RELEASED:
		case EVENT_MOUSE_CLICKED:
		case EVENT_MOUSE_MOVED:
		case EVENT_MOUSE_DRAGGED:
			snprintf(buffer + length, sizeof(buffer) - length,
				",x:%u,y:%u,button:%u,clicks:%u}",
				event->data.mouse.x,
				event->data.mouse.y,
				event->data.mouse.button,
				event->data.mouse.clicks);
			break;
		case EVENT_MOUSE_WHEEL:
			snprintf(buffer + length, sizeof(buffer) - length,
				",clicks:%hu,x:%hd,y:%hd,type:%hu,ammount:%hu,rotation:%hd,direction:%hu}",
				event->data.wheel.clicks,
				event->data.wheel.x,
				event->data.wheel.y,
   				event->data.wheel.type,
   				event->data.wheel.amount,
   				event->data.wheel.rotation,
   				event->data.wheel.direction);
			break;
		default:
		    fprintf(stderr,"\nError on file: %s, unusual event->type\n",__FILE__);
			return;
	}
	//to-do remove this for
	for(int i = 0; i < 5; i++){
	    switch(chan_select(NULL,0,NULL,&events,1,(void**) &buffer)){
            case 0:
            fprintf(stdout,"\nitem sent: %s",buffer);
            break;
            default:
            chan_dispose(events);
            fprintf(stdout,"\n%i",i);
            continue;
        }
    }
	// fprintf(stdout, "----%s\n",	 buffer);
}

int add_event(char *key_event) {
	// (uint16_t *)
	cevent = key_event;
	// Set the logger callback for library output.
	hook_set_logger(&loggerProc);

	// Set the event callback for IOhook events.
	hook_set_dispatch_proc(&dispatch_proc);
	// Start the hook and block.
	// NOTE If EVENT_HOOK_ENABLED was delivered, the status will always succeed.
	int status = hook_run();

	switch (status) {
		case IOHOOK_SUCCESS:
			// Everything is ok.
			break;

		// System level errors.
		case IOHOOK_ERROR_OUT_OF_MEMORY:
			loggerProc(LOG_LEVEL_ERROR, "Failed to allocate memory. (%#X)", status);
			break;


		// X11 specific errors.
		case IOHOOK_ERROR_X_OPEN_DISPLAY:
			loggerProc(LOG_LEVEL_ERROR, "Failed to open X11 display. (%#X)", status);
			break;

		case IOHOOK_ERROR_X_RECORD_NOT_FOUND:
			loggerProc(LOG_LEVEL_ERROR, "Unable to locate XRecord extension. (%#X)", status);
			break;

		case IOHOOK_ERROR_X_RECORD_ALLOC_RANGE:
			loggerProc(LOG_LEVEL_ERROR, "Unable to allocate XRecord range. (%#X)", status);
			break;

		case IOHOOK_ERROR_X_RECORD_CREATE_CONTEXT:
			loggerProc(LOG_LEVEL_ERROR, "Unable to allocate XRecord context. (%#X)", status);
			break;

		case IOHOOK_ERROR_X_RECORD_ENABLE_CONTEXT:
			loggerProc(LOG_LEVEL_ERROR, "Failed to enable XRecord context. (%#X)", status);
			break;


		// Windows specific errors.
		case IOHOOK_ERROR_SET_WINDOWS_HOOK_EX:
			loggerProc(LOG_LEVEL_ERROR, "Failed to register low level windows hook. (%#X)", status);
			break;


		// Darwin specific errors.
		case IOHOOK_ERROR_AXAPI_DISABLED:
			loggerProc(LOG_LEVEL_ERROR, "Failed to enable access for assistive devices. (%#X)", status);
			break;

		case IOHOOK_ERROR_CREATE_EVENT_PORT:
			loggerProc(LOG_LEVEL_ERROR, "Failed to create apple event port. (%#X)", status);
			break;

		case IOHOOK_ERROR_CREATE_RUN_LOOP_SOURCE:
			loggerProc(LOG_LEVEL_ERROR, "Failed to create apple run loop source. (%#X)", status);
			break;

		case IOHOOK_ERROR_GET_RUNLOOP:
			loggerProc(LOG_LEVEL_ERROR, "Failed to acquire apple run loop. (%#X)", status);
			break;

		case IOHOOK_ERROR_CREATE_OBSERVER:
			loggerProc(LOG_LEVEL_ERROR, "Failed to create apple run loop observer. (%#X)", status);
			break;

		// Default error.
		case IOHOOK_FAILURE:
		default:
			loggerProc(LOG_LEVEL_ERROR, "An unknown hook error occurred. (%#X)", status);
			break;
	}

	// return status;
	// printf("%d\n", status);
	return cstatus;
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