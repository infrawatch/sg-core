#define _GNU_SOURCE

#include "gen.h"

#include <arpa/inet.h>
#include <ctype.h>
#include <errno.h>
#include <getopt.h>
#include <inttypes.h>
#include <netdb.h>
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
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>

#include "amqp_snd_th.h"
#include "utils.h"

extern int batch_count;

static void usage(void) {
    fprintf(stdout,
            "%s: gen [OPTIONS] amqp_ip amqp_port\n\n"
            "Generate Collectd traffic on AMQP...\n\n"
            "positional args:\n"
            " amqp_ip          ip address of QDR\n"
            " amqp_port        port number of the QDR\n"
            "optional args:\n"
            " -p               pre-settle AMQP mode.  Non-reliable delivery (defaults to unsettled)\n"
            " -i container_id  should be unique (defaults to sa-RND)\n"
            " -a amqp_address  AMQP address for endpoint (defaults to collectd/telemetry)\n"
            " -c count         message count to stop (defaults to 0 for continuous)\n"
            " -n cd_per_mesg   number of collectd messages per AMQP message (defaults to 1)\n"
            " -o num_hosts     number of hosts to simulate (defaults to 1)\n"
            " -m metrics_hosts number of metrics per hosts to simulate (defaults to 100)\n"
            " -t num_threads   number of independent send pthreads (defaults to 1)\n"
            " -b burst_size    maximum number of AMQP msgs to send per credit interval (defaults to # of credits)\n"
            " -s sleep_usec    number of usec to sleep per credit interval (defaults to 0 for no sleep)\n"
            " -v               verbose, print extra info (additional -v increases verbosity)\n"
            " -h               show help\n\n"
            "\n",
            __func__);
}

void gen_hosts(app_data_t *app, char *host_prefix) {
    app->curr_host = 0;

    app->host_list_len = app->num_hosts * app->num_metrics;

    // Allocate the host array
    app->host_list = malloc(sizeof(host_info_t) * app->host_list_len);
    for (int i = 0; i < app->num_hosts; i++) {
        for (int j = 0; j < app->num_metrics; j++) {
            int index = (i * app->num_metrics) + j;
            asprintf(&app->host_list[index].hostname, "host_%s_%d", host_prefix, i);
            asprintf(&app->host_list[index].metric, "metric_%d", j);
            app->host_list[index].count = 0;
        }
    }

    srand(time(NULL));

    host_info_t tmp_host;
    // Random swap of list items
    for (int i = 0; i < app->host_list_len; i++) {
        int swap_host = rand() % app->host_list_len;

        tmp_host = app->host_list[i];
        app->host_list[i] = app->host_list[swap_host];

        app->host_list[swap_host] = tmp_host;
    }
}

int main(int argc, char **argv) {
    app_data_t app = {0};
    char cid_buf[100];
    int opt;

    srand(time(0));

    sprintf(app.container_id, "sagen-%x", rand() % 1024);

    asprintf(&app.amqp_address, "collectd/telemetry");
    app.message_count = 0;
    app.burst_size = 0;
    app.sleep_usec = 0;
    app.num_cd_per_mesg = 1;
    app.num_hosts = 1;
    app.num_metrics = 100;
    app.presettled = false;

    int num_threads = 1;

    while ((opt = getopt(argc, argv, "i:a:c:hvb:s:n:o:m:pt:")) != -1) {
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
            case 'o':
                app.num_hosts = atoi(optarg);
                break;
            case 't':
                num_threads = atoi(optarg);
                break;
            case 'm':
                app.num_metrics = atoi(optarg);
                break;
            case 'b':
                app.burst_size = atoi(optarg);
                break;
            case 'p':
                app.presettled = true;
                break;
            case 's':
                app.sleep_usec = atoi(optarg);
                break;
            case 'n':
                app.num_cd_per_mesg = atoi(optarg);
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

    app_data_t apps[num_threads];
    char host_prefix[100];

    for (int i = 0; i < num_threads; i++) {
        app.amqp_snd_th_running = true;
        apps[i] = app;

        snprintf(apps[i].container_id, sizeof(apps[i].container_id), "sagen-%03d", i);
        printf("%s\n",apps[i].container_id);
        snprintf(host_prefix, sizeof(host_prefix), "%d", i);
        gen_hosts(&apps[i], host_prefix);

        pthread_create(&apps[i].amqp_snd_th, NULL, amqp_snd_th, (void *)&apps[i]);
        pthread_setname_np(apps[i].amqp_snd_th, apps[i].container_id);
    }

    while (1) {
        sleep(1);
        for (int i = 0; i < num_threads; i++) {
            printf("%*s: %ld(%ld), amqp_sent: %ld(%ld), ack'd: %ld(%ld), miss: %ld, burst_size: %f\n",
                   (int)strlen(apps[i].container_id), apps[i].container_id,
                   apps[i].metrics_sent, apps[i].metrics_sent - apps[i].metrics_sent_last,
                   apps[i].amqp_sent, apps[i].amqp_sent - apps[i].amqp_sent_last,
                   apps[i].acknowledged, apps[i].acknowledged - apps[i].acknowledged_last,
                   apps[i].metrics_sent - apps[i].acknowledged,
                   apps[i].amqp_sent / (float)apps[i].total_bursts);

            sample_app_metrics(&apps[i]);
        }
        printf("----------------------------------------\n");
        if (app.amqp_snd_th_running == 0) {
            printf("Joining amqp_rcv_th...\n");
            pthread_join(app.amqp_snd_th, NULL);
            printf("Cancel socket_snd_th...\n");

            exit(0);
        }
    }

    return 0;
}