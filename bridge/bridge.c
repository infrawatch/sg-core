#include <proton/condition.h>
#include <proton/connection.h>
#include <proton/delivery.h>
#include <proton/link.h>
#include <proton/listener.h>
#include <proton/message.h>
#include <proton/netaddr.h>
#include <proton/proactor.h>
#include <proton/sasl.h>
#include <proton/session.h>
#include <proton/transport.h>

#include <arpa/inet.h>
#include <ctype.h>
#include <errno.h>
#include <getopt.h>
#include <inttypes.h>
#include <netdb.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>

#include "amqp_rcv_th.h"
#include "bridge.h"
#include "rb.h"
#include "socket_snd_th.h"
#include "utils.h"

extern int batch_count;

enum program_args {
    ARG_QDR_HOST,
    ARG_QDR_PORT,
    ARG_AMQP_ADDR,
    ARG_DOMAIN,
    ARG_UNIX,
    ARG_INET_HOST,
    ARG_INET_PORT,
    ARG_STANDALONE,
    ARG_STAT_PERIOD,
    ARG_CID,
    ARG_COUNT,
    ARG_VERBOSE,
    ARG_HELP
};

struct option longopts[] = {
    {"qdr_host", required_argument, 0, ARG_QDR_HOST},    // --qdr_host 127.0.0.1
    {"qdr_port", required_argument, 0, ARG_QDR_PORT},    // --qdr_host 5672
    {"amqp_addr", required_argument, 0, ARG_AMQP_ADDR},  // --amqp_addr collectd/telemetry
    {"inet_host", required_argument, 0, ARG_INET_HOST},  // --inet 127.0.0.1
    {"inet_port", required_argument, 0, ARG_INET_PORT},  // --inet 30000
    {"unix", required_argument, 0, ARG_UNIX},            // --unix /tmp/sgw_socket
    {"standalone", no_argument, 0, ARG_STANDALONE},
    {"stat_period", required_argument, 0, ARG_STAT_PERIOD},
    {"cid", required_argument, 0, ARG_CID},      // --cid sa-sender-00
    {"count", required_argument, 0, ARG_COUNT},  // --count 1000000
    {"verbose", no_argument, 0, ARG_VERBOSE},
    {"help", no_argument, 0, ARG_HELP}};

static void usage(char *program) {
    fprintf(stdout,
            "usage: %s [OPTIONS]\n\n"
            "The missing link between AMQP and golang.\n\n"
            "optional args:\n"
            " --qdr_host dns_or_ip   DNS name or IP of the Qpid Dispatch Router (%s)\n"
            " --qdr_port port_num    numeric port of the Qpid Dispatch Router (%s)\n"
            " --amqp_addr addr       AMQP address for the bridge endpoint (%s)\n"
            " --domain unix|inet     SG socket domain (unix)\n"
            " --unix socket_path     unix socket location (%s)\n"
            " --inet_host dns_or_ip  SmartGateway Socket IP (%s)\n"
            " --inet_port port_num   SmartGateway Socket port (%s)\n"
            " --cid name             AMQP container ID (%s)\n"
            " --count num            Number of AMQP mesg to rcv before exit, 0 for continous (0)\n"
            " --standalone           no QDR mode (QDR mode)\n"
            " --stat_period seconds  How often to print stats, 0 for no stats (0)\n"
            " --(v)erbose            print extra info, multiple instance increase verbosity.\n"
            " --(h)elp               print usage.\n"
            "\n",
            program, DEFAULT_AMQP_HOST, DEFAULT_AMQP_PORT, DEFAULT_AMQP_ADDR,
            DEFAULT_UNIX_SOCKET_PATH, DEFAULT_INET_HOST, DEFAULT_INET_PORT,
            DEFAULT_CID);
}

int main(int argc, char **argv) {
    app_data_t app = {0};
    char cid_buf[100];
    int opt, index;

    srand(time(0));

    sprintf(cid_buf, "sa-%x", rand() % 1024);

    app.stat_period = 0;        /* disabled */
    app.container_id = cid_buf; /* Should be unique */
    app.host = DEFAULT_AMQP_HOST;
    app.port = DEFAULT_AMQP_PORT;
    app.amqp_address = DEFAULT_AMQP_ADDR;
    app.message_count = 0;
    app.unix_socket_name = DEFAULT_UNIX_SOCKET_PATH;
    app.domain = AF_UNIX;

    while ((opt = getopt_long(argc, argv, "hv",
                              longopts, &index)) != -1) {
        switch (opt) {
            case ARG_QDR_HOST:
                app.host = strdup(optarg);
                break;
            case ARG_QDR_PORT:
                app.host = strdup(optarg);
                break;
            case ARG_AMQP_ADDR:
                app.amqp_address = strdup(optarg);
                break;
            case ARG_INET_HOST:
                app.peer_host = strdup(optarg);
                app.domain = AF_INET;
                break;
            case ARG_INET_PORT:
                app.peer_port = strdup(optarg);
                app.domain = AF_INET;
                break;
            case ARG_UNIX:
                app.unix_socket_name = strdup(optarg);
                app.domain = AF_UNIX;
                break;
            case ARG_CID:
                sprintf(cid_buf, optarg);
                break;
            case ARG_COUNT:
                app.message_count = atoi(optarg);
                break;
            case ARG_STANDALONE:
                app.standalone = 1;
                break;
            case ARG_STAT_PERIOD:
                app.stat_period = atoi(optarg);
                break;
            case ARG_VERBOSE:
            case 'v':
                app.verbose = 1;
                break;
            case 'h':
            case ARG_HELP:
                usage(argv[0]);
                return 0;
            default:
                usage(argv[0]);
                return 1;
        }
    }

    if (app.standalone) {
        printf("standalone mode\n");
    } else {
        printf("QDR %s:%s\n", app.host, app.port);
    }

    if (app.domain == AF_UNIX) {
        printf("Unix Socket: %s\n", app.unix_socket_name);
    } else {
        printf("Inet Socket at %s:%s\n", app.peer_host, app.peer_port);
    }

    printf("AMQP Address: %s, CID %s\n", app.amqp_address, app.container_id);

    app.rbin = rb_alloc(RING_BUFFER_COUNT, RING_BUFFER_SIZE);

    app.amqp_rcv_th_running = true;
    pthread_create(&app.amqp_rcv_th, NULL, amqp_rcv_th, (void *)&app);
    app.socket_snd_th_running = true;
    pthread_create(&app.socket_snd_th, NULL, socket_snd_th, (void *)&app);

    long last_amqp_received = 0;
    long last_overrun = 0;
    long last_out = 0;

    long sleep_count = 1;

    while (1) {
        sleep(1);

        if (sleep_count == app.stat_period) {
            printf("in: %ld(%ld), overrun: %ld(%ld), out: %ld(%ld)\n",
                   app.amqp_received, app.amqp_received - last_amqp_received,
                   app.rbin->overruns, app.rbin->overruns - last_overrun,
                   app.sock_sent, app.sock_sent - last_out);
            sleep_count = 1;
        }
        last_amqp_received = app.amqp_received;
        last_overrun = app.rbin->overruns;
        last_out = app.sock_sent;

        if (app.socket_snd_th_running == 0) {
            pthread_join(app.socket_snd_th, NULL);

            pthread_cancel(app.amqp_rcv_th);

            pthread_join(app.amqp_rcv_th, NULL);

            exit(0);
        }
        if (app.amqp_rcv_th_running == 0) {
            printf("Joining amqp_rcv_th...\n");
            pthread_join(app.amqp_rcv_th, NULL);
            printf("Cancel socket_snd_th...\n");
            pthread_cancel(app.socket_snd_th);
            printf("Joining socket_snd_th...\n");
            pthread_join(app.socket_snd_th, NULL);

            exit(0);
        }
    }

    return 0;
}