
#ifndef _RB_H
#define _RB_H 1

#include <proton/types.h>

#include <pthread.h>

typedef struct {
    pn_rwbytes_t *ring_buffer;

    int count;
    int buf_size;

    int head;
    int tail;

    pthread_mutex_t rb_mutex;
    pthread_cond_t rb_ready;

    // stats
    //
    // Buffer full
    long overruns;
    // Number of messages procesed
    long processed;
    long queue_block;
} rb_rwbytes_t;

extern rb_rwbytes_t *rb_alloc(int count, int buf_size);

extern pn_rwbytes_t *rb_get_head (rb_rwbytes_t *rb);

extern pn_rwbytes_t *rb_get_tail( rb_rwbytes_t *rb);

extern pn_rwbytes_t *rb_put( rb_rwbytes_t *rb);

extern pn_rwbytes_t *rb_get( rb_rwbytes_t *rb);

extern void rb_free(rb_rwbytes_t *rb);

extern int rb_free_size(rb_rwbytes_t *rb);

extern int rb_size(rb_rwbytes_t *rb);

extern long rb_get_queue_block(rb_rwbytes_t *rb);

#endif