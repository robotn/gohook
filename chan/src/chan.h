#ifndef chan_h
#define chan_h

#include <pthread.h>
#include <stdint.h>

#ifndef queue_h
#define queue_h


// Defines a circular buffer which acts as a FIFO queue.
typedef struct queue_t
{
    int    size;
    int    next;
    int    capacity;
    void** data;
} queue_t;

// Allocates and returns a new queue. The capacity specifies the maximum
// number of items that can be in the queue at one time. A capacity greater
// than INT_MAX / sizeof(void*) is considered an error. Returns NULL if
// initialization failed.
queue_t* queue_init(size_t capacity);

// Releases the queue resources.
void queue_dispose(queue_t* queue);

// Enqueues an item in the queue. Returns 0 if the add succeeded or -1 if it
// failed. If -1 is returned, errno will be set.
int queue_add(queue_t* queue, void* value);

// Dequeues an item from the head of the queue. Returns NULL if the queue is
// empty.
void* queue_remove(queue_t* queue);

// Returns, but does not remove, the head of the queue. Returns NULL if the
// queue is empty.
void* queue_peek(queue_t*);

#endif


// Defines a thread-safe communication pipe. Channels are either buffered or
// unbuffered. An unbuffered channel is synchronized. Receiving on either type
// of channel will block until there is data to receive. If the channel is
// unbuffered, the sender blocks until the receiver has received the value. If
// the channel is buffered, the sender only blocks until the value has been
// copied to the buffer, meaning it will block if the channel is full.
typedef struct chan_t
{
    // Buffered channel properties
    queue_t*         queue;
    
    // Unbuffered channel properties
    pthread_mutex_t  r_mu;
    pthread_mutex_t  w_mu;
    void*            data;

    // Shared properties
    pthread_mutex_t  m_mu;
    pthread_cond_t   r_cond;
    pthread_cond_t   w_cond;
    int              closed;
    int              r_waiting;
    int              w_waiting;
} chan_t;

// Allocates and returns a new channel. The capacity specifies whether the
// channel should be buffered or not. A capacity of 0 will create an unbuffered
// channel. Sets errno and returns NULL if initialization failed.
chan_t* chan_init(size_t capacity);

// Releases the channel resources.
void chan_dispose(chan_t* chan);

// Once a channel is closed, data cannot be sent into it. If the channel is
// buffered, data can be read from it until it is empty, after which reads will
// return an error code. Reading from a closed channel that is unbuffered will
// return an error code. Closing a channel does not release its resources. This
// must be done with a call to chan_dispose. Returns 0 if the channel was
// successfully closed, -1 otherwise.
int chan_close(chan_t* chan);

// Returns 0 if the channel is open and 1 if it is closed.
int chan_is_closed(chan_t* chan);

// Sends a value into the channel. If the channel is unbuffered, this will
// block until a receiver receives the value. If the channel is buffered and at
// capacity, this will block until a receiver receives a value. Returns 0 if
// the send succeeded or -1 if it failed.
int chan_send(chan_t* chan, void* data);

// Receives a value from the channel. This will block until there is data to
// receive. Returns 0 if the receive succeeded or -1 if it failed.
int chan_recv(chan_t* chan, void** data);

// Returns the number of items in the channel buffer. If the channel is
// unbuffered, this will return 0.
int chan_size(chan_t* chan);

// A select statement chooses which of a set of possible send or receive
// operations will proceed. The return value indicates which channel's
// operation has proceeded. If more than one operation can proceed, one is
// selected randomly. If none can proceed, -1 is returned. Select is intended
// to be used in conjunction with a switch statement. In the case of a receive
// operation, the received value will be pointed to by the provided pointer. In
// the case of a send, the value at the same index as the channel will be sent.
int chan_select(chan_t* recv_chans[], int recv_count, void** recv_out,
    chan_t* send_chans[], int send_count, void* send_msgs[]);

// Typed interface to send/recv chan.
int chan_send_int32(chan_t*, int32_t);
int chan_send_int64(chan_t*, int64_t);
#if ULONG_MAX == 4294967295UL
# define chan_send_int(c, d) chan_send_int64(c, d)
#else
# define chan_send_int(c, d) chan_send_int32(c, d)
#endif
int chan_send_double(chan_t*, double);
int chan_send_buf(chan_t*, void*, size_t);
int chan_recv_int32(chan_t*, int32_t*);
int chan_recv_int64(chan_t*, int64_t*);
#if ULONG_MAX == 4294967295UL
# define chan_recv_int(c, d) chan_recv_int64(c, d)
#else
# define chan_recv_int(c, d) chan_recv_int32(c, d)
#endif
int chan_recv_double(chan_t*, double*);
int chan_recv_buf(chan_t*, void*, size_t);
#define _GNU_SOURCE
#undef __STRICT_ANSI__

#ifdef __APPLE__
#define _XOPEN_SOURCE
#endif

#include <errno.h>
#include <pthread.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

#include <time.h>
#include <sys/time.h>

#ifdef __MACH__
#include <mach/clock.h>
#include <mach/mach.h>
#endif

#include "chan.h"
#include "queue.h"

#ifdef _WIN32
#include <windows.h>
#define CLOCK_REALTIME 0
//static int clock_gettime (int __attribute__((__unused__)) clockid, struct timespec *tp) {
//    FILETIME ft;
//    ULARGE_INTEGER t64;
//    GetSystemTimeAsFileTime (&ft);
//    t64.LowPart = ft.dwLowDateTime;
//    t64.HighPart = ft.dwHighDateTime;
//    tp->tv_sec = t64.QuadPart / 10000000 - 11644473600;
//    tp->tv_nsec = t64.QuadPart % 10000000 * 100;
//    return 0;
//}
#endif

static int buffered_chan_init(chan_t* chan, size_t capacity);
static int buffered_chan_send(chan_t* chan, void* data);
static int buffered_chan_recv(chan_t* chan, void** data);

static int unbuffered_chan_init(chan_t* chan);
static int unbuffered_chan_send(chan_t* chan, void* data);
static int unbuffered_chan_recv(chan_t* chan, void** data);

static int chan_can_recv(chan_t* chan);
static int chan_can_send(chan_t* chan);
static int chan_is_buffered(chan_t* chan);

void current_utc_time(struct timespec *ts) {
#ifdef __MACH__
    clock_serv_t cclock;
    mach_timespec_t mts;
    host_get_clock_service(mach_host_self(), CALENDAR_CLOCK, &cclock);
    clock_get_time(cclock, &mts);
    mach_port_deallocate(mach_task_self(), cclock);
    ts->tv_sec = mts.tv_sec;
    ts->tv_nsec = mts.tv_nsec;
#else
    clock_gettime(CLOCK_REALTIME, ts);
#endif
}

// Allocates and returns a new channel. The capacity specifies whether the
// channel should be buffered or not. A capacity of 0 will create an unbuffered
// channel. Sets errno and returns NULL if initialization failed.
chan_t* chan_init(size_t capacity)
{
    chan_t* chan = (chan_t*) malloc(sizeof(chan_t));
    if (!chan)
    {
        errno = ENOMEM;
        return NULL;
    }

    if (capacity > 0)
    {
        if (buffered_chan_init(chan, capacity) != 0)
        {
            free(chan);
            return NULL;
        }
    }
    else
    {
        if (unbuffered_chan_init(chan) != 0)
        {
            free(chan);
            return NULL;
        }
    }

    return chan;
}

static int buffered_chan_init(chan_t* chan, size_t capacity)
{
    queue_t* queue = queue_init(capacity);
    if (!queue)
    {
        return -1;
    }

    if (unbuffered_chan_init(chan) != 0)
    {
        queue_dispose(queue);
        return -1;
    }

    chan->queue = queue;
    return 0;
}

static int unbuffered_chan_init(chan_t* chan)
{
    if (pthread_mutex_init(&chan->w_mu, NULL) != 0)
    {
        return -1;
    }

    if (pthread_mutex_init(&chan->r_mu, NULL) != 0)
    {
        pthread_mutex_destroy(&chan->w_mu);
        return -1;
    }

    if (pthread_mutex_init(&chan->m_mu, NULL) != 0)
    {
        pthread_mutex_destroy(&chan->w_mu);
        pthread_mutex_destroy(&chan->r_mu);
        return -1;
    }

    if (pthread_cond_init(&chan->r_cond, NULL) != 0)
    {
        pthread_mutex_destroy(&chan->m_mu);
        pthread_mutex_destroy(&chan->w_mu);
        pthread_mutex_destroy(&chan->r_mu);
        return -1;
    }

    if (pthread_cond_init(&chan->w_cond, NULL) != 0)
    {
        pthread_mutex_destroy(&chan->m_mu);
        pthread_mutex_destroy(&chan->w_mu);
        pthread_mutex_destroy(&chan->r_mu);
        pthread_cond_destroy(&chan->r_cond);
        return -1;
    }

    chan->closed = 0;
    chan->r_waiting = 0;
    chan->w_waiting = 0;
    chan->queue = NULL;
    chan->data = NULL;
    return 0;
}

// Releases the channel resources.
void chan_dispose(chan_t* chan)
{
    if (chan_is_buffered(chan))
    {
        queue_dispose(chan->queue);
    }

    pthread_mutex_destroy(&chan->w_mu);
    pthread_mutex_destroy(&chan->r_mu);

    pthread_mutex_destroy(&chan->m_mu);
    pthread_cond_destroy(&chan->r_cond);
    pthread_cond_destroy(&chan->w_cond);
    free(chan);
}

// Once a channel is closed, data cannot be sent into it. If the channel is
// buffered, data can be read from it until it is empty, after which reads will
// return an error code. Reading from a closed channel that is unbuffered will
// return an error code. Closing a channel does not release its resources. This
// must be done with a call to chan_dispose. Returns 0 if the channel was
// successfully closed, -1 otherwise. If -1 is returned, errno will be set.
int chan_close(chan_t* chan)
{
    int success = 0;
    pthread_mutex_lock(&chan->m_mu);
    if (chan->closed)
    {
        // Channel already closed.
        success = -1;
        errno = EPIPE;
    }
    else
    {
        // Otherwise close it.
        chan->closed = 1;
        pthread_cond_broadcast(&chan->r_cond);
        pthread_cond_broadcast(&chan->w_cond);
    }
    pthread_mutex_unlock(&chan->m_mu);
    return success;
}

// Returns 0 if the channel is open and 1 if it is closed.
int chan_is_closed(chan_t* chan)
{
    pthread_mutex_lock(&chan->m_mu);
    int closed = chan->closed;
    pthread_mutex_unlock(&chan->m_mu);
    return closed;
}

// Sends a value into the channel. If the channel is unbuffered, this will
// block until a receiver receives the value. If the channel is buffered and at
// capacity, this will block until a receiver receives a value. Returns 0 if
// the send succeeded or -1 if it failed. If -1 is returned, errno will be set.
int chan_send(chan_t* chan, void* data)
{
    if (chan_is_closed(chan))
    {
        // Cannot send on closed channel.
        errno = EPIPE;
        return -1;
    }

    return chan_is_buffered(chan) ?
        buffered_chan_send(chan, data) :
        unbuffered_chan_send(chan, data);
}

// Receives a value from the channel. This will block until there is data to
// receive. Returns 0 if the receive succeeded or -1 if it failed. If -1 is
// returned, errno will be set.
int chan_recv(chan_t* chan, void** data)
{
    return chan_is_buffered(chan) ?
        buffered_chan_recv(chan, data) :
        unbuffered_chan_recv(chan, data);
}

static int buffered_chan_send(chan_t* chan, void* data)
{
    pthread_mutex_lock(&chan->m_mu);
    while (chan->queue->size == chan->queue->capacity)
    {
        // Block until something is removed.
        chan->w_waiting++;
        pthread_cond_wait(&chan->w_cond, &chan->m_mu);
        chan->w_waiting--;
    }

    int success = queue_add(chan->queue, data);

    if (chan->r_waiting > 0)
    {
        // Signal waiting reader.
        pthread_cond_signal(&chan->r_cond);
    }

    pthread_mutex_unlock(&chan->m_mu);
    return success;
}

static int buffered_chan_recv(chan_t* chan, void** data)
{
    pthread_mutex_lock(&chan->m_mu);
    while (chan->queue->size == 0)
    {
        if (chan->closed)
        {
            pthread_mutex_unlock(&chan->m_mu);
            errno = EPIPE;
            return -1;
        }

        // Block until something is added.
        chan->r_waiting++;
        pthread_cond_wait(&chan->r_cond, &chan->m_mu);
        chan->r_waiting--;
    }

    void* msg = queue_remove(chan->queue);
    if (data)
    {
        *data = msg;
    }

    if (chan->w_waiting > 0)
    {
        // Signal waiting writer.
        pthread_cond_signal(&chan->w_cond);
    }

    pthread_mutex_unlock(&chan->m_mu);
    return 0;
}

static int unbuffered_chan_send(chan_t* chan, void* data)
{
    pthread_mutex_lock(&chan->w_mu);
    pthread_mutex_lock(&chan->m_mu);

    if (chan->closed)
    {
        pthread_mutex_unlock(&chan->m_mu);
        pthread_mutex_unlock(&chan->w_mu);
        errno = EPIPE;
        return -1;
    }

    chan->data = data;
    chan->w_waiting++;

    if (chan->r_waiting > 0)
    {
        // Signal waiting reader.
        pthread_cond_signal(&chan->r_cond);
    }

    // Block until reader consumed chan->data.
    pthread_cond_wait(&chan->w_cond, &chan->m_mu);

    pthread_mutex_unlock(&chan->m_mu);
    pthread_mutex_unlock(&chan->w_mu);
    return 0;
}

static int unbuffered_chan_recv(chan_t* chan, void** data)
{
    pthread_mutex_lock(&chan->r_mu);
    pthread_mutex_lock(&chan->m_mu);

    while (!chan->closed && !chan->w_waiting)
    {
        // Block until writer has set chan->data.
        chan->r_waiting++;
        pthread_cond_wait(&chan->r_cond, &chan->m_mu);
        chan->r_waiting--;
    }

    if (chan->closed)
    {
        pthread_mutex_unlock(&chan->m_mu);
        pthread_mutex_unlock(&chan->r_mu);
        errno = EPIPE;
        return -1;
    }

    if (data)
    {
        *data = chan->data;
    }
    chan->w_waiting--;

    // Signal waiting writer.
    pthread_cond_signal(&chan->w_cond);

    pthread_mutex_unlock(&chan->m_mu);
    pthread_mutex_unlock(&chan->r_mu);
    return 0;
}

// Returns the number of items in the channel buffer. If the channel is
// unbuffered, this will return 0.
int chan_size(chan_t* chan)
{
    int size = 0;
    if (chan_is_buffered(chan))
    {
        pthread_mutex_lock(&chan->m_mu);
        size = chan->queue->size;
        pthread_mutex_unlock(&chan->m_mu);
    }
    return size;
}

typedef struct
{
    int     recv;
    chan_t* chan;
    void*   msg_in;
    int     index;
} select_op_t;

// A select statement chooses which of a set of possible send or receive
// operations will proceed. The return value indicates which channel's
// operation has proceeded. If more than one operation can proceed, one is
// selected randomly. If none can proceed, -1 is returned. Select is intended
// to be used in conjunction with a switch statement. In the case of a receive
// operation, the received value will be pointed to by the provided pointer. In
// the case of a send, the value at the same index as the channel will be sent.
int chan_select(chan_t* recv_chans[], int recv_count, void** recv_out,
    chan_t* send_chans[], int send_count, void* send_msgs[])
{
    // TODO: Add support for blocking selects.

    select_op_t candidates[recv_count + send_count];
    int count = 0;
    int i;

    // Determine receive candidates.
    for (i = 0; i < recv_count; i++)
    {
        chan_t* chan = recv_chans[i];
        if (chan_can_recv(chan))
        {
            select_op_t op;
            op.recv = 1;
            op.chan = chan;
            op.index = i;
            candidates[count++] = op;
        }
    }

    // Determine send candidates.
    for (i = 0; i < send_count; i++)
    {
        chan_t* chan = send_chans[i];
        if (chan_can_send(chan))
        {
            select_op_t op;
            op.recv = 0;
            op.chan = chan;
            op.msg_in = send_msgs[i];
            op.index = i + recv_count;
            candidates[count++] = op;
        }
    }

    if (count == 0)
    {
        return -1;
    }

    // Seed rand using current time in nanoseconds.
    struct timespec ts;
    current_utc_time(&ts);
    srand(ts.tv_nsec);

    // Select candidate and perform operation.
    select_op_t select = candidates[rand() % count];
    if (select.recv && chan_recv(select.chan, recv_out) != 0)
    {
        return -1;
    }
    else if (!select.recv && chan_send(select.chan, select.msg_in) != 0)
    {
        return -1;
    }

    return select.index;
}

static int chan_can_recv(chan_t* chan)
{
    if (chan_is_buffered(chan))
    {
        return chan_size(chan) > 0;
    }

    pthread_mutex_lock(&chan->m_mu);
    int sender = chan->w_waiting > 0;
    pthread_mutex_unlock(&chan->m_mu);
    return sender;
}

static int chan_can_send(chan_t* chan)
{
    int send;
    if (chan_is_buffered(chan))
    {
        // Can send if buffered channel is not full.
        pthread_mutex_lock(&chan->m_mu);
        send = chan->queue->size < chan->queue->capacity;
        pthread_mutex_unlock(&chan->m_mu);
    }
    else
    {
        // Can send if unbuffered channel has receiver.
        pthread_mutex_lock(&chan->m_mu);
        send = chan->r_waiting > 0;
        pthread_mutex_unlock(&chan->m_mu);
    }

    return send;
}

static int chan_is_buffered(chan_t* chan)
{
    return chan->queue != NULL;
}

int chan_send_int32(chan_t* chan, int32_t data)
{
    int32_t* wrapped = malloc(sizeof(int32_t));
    if (!wrapped)
    {
        return -1;
    }

    *wrapped = data;

    int success = chan_send(chan, wrapped);
    if (success != 0)
    {
        free(wrapped);
    }

    return success;
}

int chan_recv_int32(chan_t* chan, int32_t* data)
{
    int32_t* wrapped = NULL;
    int success = chan_recv(chan, (void*) &wrapped);
    if (wrapped != NULL)
    {
        *data = *wrapped;
        free(wrapped);
    }

    return success;
}

int chan_send_int64(chan_t* chan, int64_t data)
{
    int64_t* wrapped = malloc(sizeof(int64_t));
    if (!wrapped)
    {
        return -1;
    }

    *wrapped = data;

    int success = chan_send(chan, wrapped);
    if (success != 0)
    {
        free(wrapped);
    }

    return success;
}

int chan_recv_int64(chan_t* chan, int64_t* data)
{
    int64_t* wrapped = NULL;
    int success = chan_recv(chan, (void*) &wrapped);
    if (wrapped != NULL)
    {
        *data = *wrapped;
        free(wrapped);
    }

    return success;
}

int chan_send_double(chan_t* chan, double data)
{
    double* wrapped = malloc(sizeof(double));
    if (!wrapped)
    {
        return -1;
    }

    *wrapped = data;

    int success = chan_send(chan, wrapped);
    if (success != 0)
    {
        free(wrapped);
    }

    return success;
}

int chan_recv_double(chan_t* chan, double* data)
{
    double* wrapped = NULL;
    int success = chan_recv(chan, (void*) &wrapped);
    if (wrapped != NULL)
    {
        *data = *wrapped;
        free(wrapped);
    }

    return success;
}

int chan_send_buf(chan_t* chan, void* data, size_t size)
{
    void* wrapped = malloc(size);
    if (!wrapped)
    {
        return -1;
    }

    memcpy(wrapped, data, size);

    int success = chan_send(chan, wrapped);
    if (success != 0)
    {
        free(wrapped);
    }

    return success;
}

int chan_recv_buf(chan_t* chan, void* data, size_t size)
{
    void* wrapped = NULL;
    int success = chan_recv(chan, (void*) &wrapped);
    if (wrapped != NULL)
    {
        memcpy(data, wrapped, size);
        free(wrapped);
    }

    return success;
}
#define _GNU_SOURCE

#ifdef __APPLE__
#define _XOPEN_SOURCE
#endif

#include <errno.h>
#include <limits.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>

#include "queue.h"

#if defined(_WIN32) && !defined(ENOBUFS)
#include <winsock.h>
#define ENOBUFS WSAENOBUFS
#endif

// Returns 0 if the queue is not at capacity. Returns 1 otherwise.
static inline int queue_at_capacity(queue_t* queue)
{
    return queue->size >= queue->capacity;
}

// Allocates and returns a new queue. The capacity specifies the maximum
// number of items that can be in the queue at one time. A capacity greater
// than INT_MAX / sizeof(void*) is considered an error. Returns NULL if
// initialization failed.
queue_t* queue_init(size_t capacity)
{
    if (capacity > INT_MAX / sizeof(void*))
    {
        errno = EINVAL;
        return NULL;
    }

    queue_t* queue = (queue_t*) malloc(sizeof(queue_t));
    void**   data  = (void**) malloc(capacity * sizeof(void*));
    if (!queue || !data)
    {
        // In case of free(NULL), no operation is performed.
        free(queue);
        free(data);
        errno = ENOMEM;
        return NULL;
    }

    queue->size = 0;
    queue->next = 0;
    queue->capacity = capacity;
    queue->data = data;
    return queue;
}

// Releases the queue resources.
void queue_dispose(queue_t* queue)
{
    free(queue->data);
    free(queue);
}

// Enqueues an item in the queue. Returns 0 is the add succeeded or -1 if it
// failed. If -1 is returned, errno will be set.
int queue_add(queue_t* queue, void* value)
{
    if (queue_at_capacity(queue))
    {
        errno = ENOBUFS;
        return -1;
    }

    int pos = queue->next + queue->size;
    if (pos >= queue->capacity)
    {
       pos -= queue->capacity;
    }

    queue->data[pos] = value;

    queue->size++;
    return 0;
}

// Dequeues an item from the head of the queue. Returns NULL if the queue is
// empty.
void* queue_remove(queue_t* queue)
{
    void* value = NULL;

    if (queue->size > 0)
    {
        value = queue->data[queue->next];
        queue->next++;
        queue->size--;
        if (queue->next >= queue->capacity)
        {
            queue->next -= queue->capacity;
        }
    }

    return value;
}

// Returns, but does not remove, the head of the queue. Returns NULL if the
// queue is empty.
void* queue_peek(queue_t* queue)
{
    return queue->size ? queue->data[queue->next] : NULL;
}

#endif
