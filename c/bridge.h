#ifndef _BRIDGE_H
#define _BRIDGE_H 1

#include <proton/condition.h>
#include <proton/listener.h>
#include <proton/proactor.h>
#include <proton/sasl.h>

#include "rb.h"

typedef struct  {
    int standalone;
    int verbose;
    int max_amqp_queue_depth;

    pthread_t amqp_rcv_th;
    pthread_t socket_snd_th;

    int amqp_rcv_th_running;
    int socket_snd_th_running;

    const char *host, *port;
    char *peer_host, *peer_port;
    
    const char *amqp_address;
    const char *container_id;
    int message_count;

    pn_proactor_t *proactor;
    pn_listener_t *listener;
    pn_rwbytes_t msgout; /* Buffers for incoming/outgoing messages */

    rb_rwbytes_t *rbin;

    /* Sender values */
    int sent;
    int acknowledged;
    pn_link_t *sender;

    /* Receiver values */
    long received;
    long decore_errors;
    long would_block;
} app_data_t;

#endif
