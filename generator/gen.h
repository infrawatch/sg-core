#ifndef _GEN_H
#define _GEN_H 1

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
    int presettled;
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
    int logs;
    int ceilometer;
    int collectd;
    
    pthread_t amqp_snd_th;

    int amqp_snd_th_running;

    const char *host, *port;
    
    char *amqp_address;

    char container_id[20];

    pn_message_t *message;
    int message_count;

    pn_proactor_t *proactor;
    pn_connection_t *connection;
    pn_transport_t *transport;
    pn_link_t *sender;
    pn_rwbytes_t msgout; /* Buffers for incoming/outgoing messages */

    /* Sender values */
    volatile long metrics_sent;
    volatile long amqp_sent;
    volatile long acknowledged;
    volatile long metrics_sent_last;
    volatile long amqp_sent_last;
    volatile long acknowledged_last;

    host_info_t *host_list;
    int host_list_len;
    int curr_host;

    char MSG_BUFFER[4096];
    char now_buf[100];
} app_data_t;

#endif
