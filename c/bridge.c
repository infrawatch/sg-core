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

static void usage(void) {
    fprintf(stdout,
            "%s: bridge [OPTIONS] amqp_ip amqp_port sg_ip sg_port\n\n"
            "The missing link between AMQP and golang.\n\n"
            "positional args:\n"
            "  amqp_ip   ip address to bind AMQP listener\n"
            "  amqp_port port number to bind AMQP listener\n"
            "  sg_ip     ip address of smart gateway\n"
            "  sg_port   port number of smart gateway\n\n"
            "optional args:\n"
            " -v               verbose, print extra info (defaults no verbose)\n"
            " -s               standalone mode, no QDR (defaults QDR mode)\n"
            " -i container_id  should be unique (defaults to sa-RND)\n"
            " -a amqp_address  AMQP address for endpoint (defaults to "
            "collectd/telemetry)\n"
            " -c count         message count to stop (defaults to 0 for "
            "continous)\n"
            " -h show help\n\n"
            "\n",
            __func__);
}

int main(int argc, char **argv) {
    app_data_t app = {0};
    char cid_buf[100];
    int opt;

    srand(time(0));

    sprintf(cid_buf, "sa-%x", rand() % 1024);

    app.container_id = cid_buf; /* Should be unique */

    app.host = "127.0.0.1";
    app.port = "5672";
    app.amqp_address = "collectd/telemetry";
    app.message_count = 0;

    while ((opt = getopt(argc, argv, "i:a:c:shv")) != -1) {
        switch (opt) {
            case 'i':
                sprintf(cid_buf, optarg);
                break;
            case 'a':
                app.amqp_address = strdup(optarg);
                break;
            case 'c':
                app.message_count = atoi(optarg);
                break;
            case 's':
                app.standalone = 1;
                break;
            case 'v':
                app.verbose = 1;
                break;
            case 'h':
                usage();
                return 0;
            default:
                usage();
                return 1;
        }
    }

    if ((argc - optind) < 4) {
        fprintf(stderr, "Missing required arguments -- exiting!\n");
        usage();

        return 1;
    }

    app.host = strdup(argv[optind++]);
    app.port = strdup(argv[optind++]);
    app.peer_host = strdup(argv[optind++]);
    app.peer_port = strdup(argv[optind++]);

    app.rbin = rb_alloc(1000, 1024);

    app.amqp_rcv_th_running = 1;
    pthread_create(&app.amqp_rcv_th, NULL, amqp_rcv_th, (void *)&app);
    app.socket_snd_th_running = 1;
    pthread_create(&app.socket_snd_th, NULL, socket_snd_th, (void *)&app);

    long last_processed = 0;
    long last_overrun = 0;

    while (1) {
        sleep(1);

        printf("processed: %ld(%ld), overrun: %ld(%ld), decore_errors: %ld, free: %d, sock_qb: %ld\n", app.rbin->processed,
               app.rbin->processed - last_processed, app.rbin->overruns, app.rbin->overruns - last_overrun, app.decore_errors,
               rb_free_size(app.rbin), rb_get_queue_block(app.rbin));

        last_processed = app.rbin->processed;
        last_overrun = app.rbin->overruns;

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