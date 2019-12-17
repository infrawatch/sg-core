#include <proton/types.h>
#include <pthread.h>
#include <stdlib.h>
#include <stdio.h>

#include "rb.h"

rb_rwbytes_t *rb_alloc(int count, int buf_size) {

    rb_rwbytes_t *rb = malloc(sizeof(rb_rwbytes_t));

    rb->count = count;
    rb->buf_size = buf_size;

    if ((rb->ring_buffer = malloc(count * sizeof(pn_rwbytes_t))) == NULL) {
        free(rb);

        return NULL;
    }

    for (int i = 0; i < count; i++) {
        if ( (rb->ring_buffer[i].start = malloc(buf_size)) == NULL ) {
            rb_free(rb);

            return NULL;
        }
        rb->ring_buffer[i].size = 0;
    }
    rb->head = 0;
    rb->tail = count - 1;

    rb->overruns = 0;
    rb->processed = 0;
    rb->queue_block = 0;

    pthread_cond_init(&rb->rb_ready, NULL);
    pthread_mutex_init(&rb->rb_mutex, NULL);

    return rb;
}

void rb_free(rb_rwbytes_t *rb) {
    if (rb == NULL) {
        return;
    }
    for (int i = 0; i < rb->count; i++) {
        free(rb->ring_buffer[i].start);
    }
    free(rb->ring_buffer);
    free(rb);
}

pn_rwbytes_t *rb_get_head(rb_rwbytes_t *rb) {
    if (rb == NULL) {
        return NULL;
    }
    return &rb->ring_buffer[rb->head];
}

pn_rwbytes_t *rb_get_tail(rb_rwbytes_t *rb) {
    if (rb == NULL) {
        return NULL;
    }
    return &rb->ring_buffer[rb->tail];
}

// Place the already allocated buffer entry in the
// queue.  The producer does not block as it needs
// to continually process the incoming AMQP messagaes.
// Just need to wake up the consumer if it is waiting
// for messages.
pn_rwbytes_t *rb_put(rb_rwbytes_t *rb) {
    if (rb == NULL) {
        return NULL;
    }
    pn_rwbytes_t *next_buffer = NULL;

    pthread_mutex_lock(&rb->rb_mutex);
    int next = (rb->head + 1) % rb->count;
    if (next != rb->tail) {
        rb->head = next;
        next_buffer = &rb->ring_buffer[rb->head];
    pthread_cond_broadcast(&rb->rb_ready);
//printf("ok   free %d  head: %d  tail: %d\n",rb_free_size(rb), rb->head, rb->tail);
    } else {
        rb->overruns++;
//printf("overrun free %d  head: %d  tail: %d\n",rb_free_size(rb), rb->head, rb->tail);
        rb->ring_buffer[rb->head].size = 0;
    }

    pthread_mutex_unlock(&rb->rb_mutex);

    return next_buffer;  // May be NULL
}

pn_rwbytes_t *rb_get(rb_rwbytes_t *rb) {
    if (rb == NULL) {
        return NULL;
    }
    
    int next;

    pthread_mutex_lock(&rb->rb_mutex);

    next = (rb->tail + 1) % rb->count;
    while (next == rb->head) {
        pthread_cond_wait(&rb->rb_ready, &rb->rb_mutex);
        next = (rb->tail + 1) % rb->count;
        rb->queue_block++;
    }
    // set data size to zero
    rb->ring_buffer[rb->tail].size = 0;

    rb->tail = next;

//    pthread_cond_broadcast(&rb->rb_ready);  

    pthread_mutex_unlock(&rb->rb_mutex);

    rb->processed++;

    return &rb->ring_buffer[rb->tail];
}

int rb_free_size(rb_rwbytes_t *rb) {
    int diff = rb->head - rb->tail;
    return diff >= 0 ? (diff) : (rb->count+diff);
}

int rb_size(rb_rwbytes_t *rb) {
    return rb->count;
}

long rb_get_overruns(rb_rwbytes_t *rb) {
    return rb->overruns;
}

long rb_get_processed(rb_rwbytes_t *rb) {
    return rb->processed;
}

long rb_get_queue_block(rb_rwbytes_t *rb) {
    return rb->queue_block;
}