#ifndef _BRIDGE_H
#define _BRIDGE_H 1

#include <sys/socket.h>
#include <sys/un.h>

#include <proton/condition.h>
#include <proton/listener.h>
#include <proton/proactor.h>
#include <proton/sasl.h>

#include "rb.h"

#define DEFAULT_UNIX_SOCKET_PATH "/tmp/smartgateway"
#define DEFAULT_AMQP_HOST "127.0.0.1"
#define DEFAULT_AMQP_PORT "5672"
#define DEFAULT_AMQP_ADDR "collectd/telemetry"
#define DEFAULT_INET_HOST "127.0.0.1"
#define DEFAULT_INET_PORT "30000"
#define DEFAULT_CID       "bridge-%x"
#define RING_BUFFER_COUNT 1000
#define RING_BUFFER_SIZE  2048

typedef struct  {
    int standalone;
    int verbose;
    int domain;  // connection to SG, AF_UNIX || AF_INET
    int max_q_depth;
    int stat_period;

    pthread_t amqp_rcv_th;
    pthread_t socket_snd_th;

    int amqp_rcv_th_running;
    int socket_snd_th_running;

    const char *unix_socket_name;

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
    long amqp_received;

    /* Receiver values */
    long sock_sent;
    long amqp_decode_errs;
    long sock_would_block;

    // Use a struct big enough more most things
    struct sockaddr_un sa;
    socklen_t sa_len;
    int send_sock;
} app_data_t;

#endif
