#ifndef _BRIDGE_H
#define _BRIDGE_H 1

#include <proton/condition.h>
#include <proton/listener.h>
#include <proton/proactor.h>
#include <proton/sasl.h>
#include <proton/connection.h>
#include <proton/message.h>

typedef struct {
    char *hostname;
    char *metric;
    long count;
} host_info_t;

typedef struct  {
    int standalone;
    int verbose;
    int max_q_depth;
    int burst_size;
    int sleep_usec;
    long total_bursts;
    long burst_credit;    
    int num_cd_per_mesg;
    int num_hosts;
    int num_metrics;
    int metrics_per_second;
    
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
    long metrics_sent;
    long amqp_sent;
    long acknowledged;

    host_info_t *host_list;
    int host_list_len;
    int curr_host;

} app_data_t;

#endif
