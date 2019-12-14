#include <proton/types.h>
#include <pthread.h>
#include <stdlib.h>

#include "rb.h"

const rb_rwbytes_t *init_rb(int count, int buf_size) {
    rb_rwbytes_t *rb = malloc(sizeof(rb_rwbytes_t));

    rb->count = count;
    rb->buf_size = buf_size;

    rb->ring_buffer = malloc(count * sizeof(pn_rwbytes_t));

    for (int i = 0; i < count; i++) {
        rb->ring_buffer[i].start = malloc(buf_size);
        rb->ring_buffer[i].size = 0;
    }
    rb->head = 0;
    rb->tail = count - 1;

    pthread_condition_init(&rb->rb_ready, NULL);
    pthread_mutex_init(&rb->rb_ready, NULL);

    return rb;
}

pn_rwbytes_t *rb_get_head(rb_rwbytes_t *rb) {
    return &rb->ring_buffer[rb->head];
}

pn_rwbytes_t *rb_get_tail(rb_rwbytes_t *rb) {
    return &rb->ring_buffer[rb->tail];
}

// TODO -- Really don't want to block on put
//   best to just drop the message
pn_rwbytes_t *rb_put(rb_rwbytes_t *rb) {
    int next;

    pthread_mutex_lock(&rb->rb_mutex);
    while ( (next = (rb->head + 1) % rb->count) && (next == rb->tail) ) {
        pthread_cond_wait(&rb->rb_ready,&rb->rb_mutex);
    }

    rb->head = next;

    pthread_cond_broadcast(&rb->rb_ready);

    pthread_mutex_unlock(&rb->rb_mutex);

    return &rb->ring_buffer[rb->head];
}

pn_rwbytes_t *rb_get(rb_rwbytes_t *rb) {
    int next;

    pthread_mutex_lock(&rb->rb_mutex);
    while ( (next = (rb->tail + 1) % rb->count) && (next == rb->head) ) {
        pthread_cond_wait(&rb->rb_ready,&rb->rb_mutex);
    }
    rb->tail = next;

    rb->ring_buffer[rb->tail].size = 0;

    pthread_cond_broadcast(&rb->rb_ready);

    pthread_mutex_unlock(&rb->rb_mutex);

    return &rb->ring_buffer[rb->tail];
}
