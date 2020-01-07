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

#include "amqp_snd_th.h"
#include "gen.h"
#include "utils.h"

extern int batch_count;

static void usage(void) {
    fprintf(stdout,
            "%s: gen [OPTIONS] amqp_ip amqp_port\n\n"
            "The missing link between AMQP and golang.\n\n"
            "positional args:\n"
            "  amqp_ip   ip address of QDR\n"
            "  amqp_port port number of the QDR\n"
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

    app.amqp_address = "collectd/telemetry";
    app.message_count = 0;
    app.burst_size = 0;
    app.sleep_usec = 0;
    
    while ((opt = getopt(argc, argv, "i:a:c:hvb:s:")) != -1) {
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
            case 'v':
                app.verbose++;
                break;
            case 'h':
                usage();
                return 0;
            case 'b':
                app.burst_size = atoi(optarg);
                break;
            case 's':
                app.sleep_usec = atoi(optarg);
                break;
            default:
                usage();
                return 1;
        }
    }

    if ((argc - optind) < 2) {
        fprintf(stderr, "Missing required arguments -- exiting!\n");
        usage();

        return 1;
    }

    app.host = strdup(argv[optind++]);
    app.port = strdup(argv[optind++]);

    app.amqp_snd_th_running = true;
    pthread_create(&app.amqp_snd_th, NULL, amqp_snd_th, (void *)&app);

    long last_processed = 0;
    long last_acknowledged = 0;

    while (1) {
        sleep(1);

        printf("sent: %ld(%ld), ack'd: %ld(%ld), miss: %ld\n",
               app.sent, app.sent - last_processed,
               app.acknowledged, app.acknowledged - last_acknowledged,
               app.sent - app.acknowledged);

        last_processed = app.sent;
        last_acknowledged = app.acknowledged;

        if (app.amqp_snd_th_running == 0) {
            printf("Joining amqp_rcv_th...\n");
            pthread_join(app.amqp_snd_th, NULL);
            printf("Cancel socket_snd_th...\n");

            exit(0);
        }
    }

    return 0;
}