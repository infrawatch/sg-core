#ifndef _BRIDGE_H
#define _BRIDGE_H 1

#include <proton/condition.h>
#include <proton/listener.h>
#include <proton/proactor.h>
#include <proton/sasl.h>
#include <proton/connection.h>

typedef struct  {
    int standalone;
    int verbose;
    int max_q_depth;
    int burst_size;
    int sleep_usec;
    int num_cd_per_mesg;
    
    pthread_t amqp_snd_th;

    int amqp_snd_th_running;

    const char *host, *port;
    
    const char *amqp_address;
    const char *container_id;

    pn_message_t *message;
    int message_count;

    pn_proactor_t *proactor;
    pn_connection_t *connection;
    pn_transport_t *transport;
    pn_link_t *sender;
    pn_rwbytes_t msgout; /* Buffers for incoming/outgoing messages */

    /* Sender values */
    long sent;
    long acknowledged;
} app_data_t;

#endif
