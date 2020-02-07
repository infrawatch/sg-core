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
#define DEFAULT_AMQP_URL "amqp://127.0.0.1:5672/collectd/telemetry"
#define DEFAULT_INET_HOST "127.0.0.1"
#define DEFAULT_INET_PORT "30000"
#define DEFAULT_CID       "bridge-%x"
#define DEFAULT_CONTAINER_ID_PATTERN "sa-%x"
#define DEFAULT_STATS_PERIOD "0"
#define DEFAULT_SOCKET_BLOCK "false"
#define DEFAULT_STOP_COUNT "0"

#define RING_BUFFER_COUNT 1000
#define RING_BUFFER_SIZE  2048

#define AMQP_URL_REGEX "^amqp://(([a-z]+)(:([a-z]+))*@)*([a-zA-Z_0-9.]+)(:([0-9]+))*(.+)$"
//#define AMQP_URL_REGEX "^amqp://"

typedef struct {
    char *user;
    char *password;
    char *address;
    char *host;
    char *port;
    char *url;
} amqp_connection;


typedef struct  {
    // Parameters section
    int standalone;
    int verbose;
    int domain;         // connection to SG, AF_UNIX || AF_INET
    int stat_period;

    amqp_connection amqp_con;
    const char *container_id;
    int message_count;
    const char *unix_socket_name;
    int socket_flags;

    char *peer_host, *peer_port;

    // Runtime 
    pthread_t amqp_rcv_th;
    pthread_t socket_snd_th;

    int amqp_rcv_th_running;
    int socket_snd_th_running;
    
    pn_proactor_t *proactor;
    pn_listener_t *listener;
    pn_rwbytes_t msgout; /* Buffers for incoming/outgoing messages */

    rb_rwbytes_t *rbin;

    /* Rcv stats */
    long amqp_received;
    long amqp_partial;

    /* Ring buffer stats */
    int max_q_depth;

    /* Snd stats */
    long sock_sent;
    long amqp_decode_errs;
    long sock_would_block;

    // Use a struct big enough more most things
    struct sockaddr_un sa;
    socklen_t sa_len;
    int send_sock;
} app_data_t;

#endif
