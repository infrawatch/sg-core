#define _GNU_SOURCE

#include "bridge.h"

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
#include <regex.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <time.h>
#include <unistd.h>

#include "amqp_rcv_th.h"
#include "rb.h"
#include "socket_snd_th.h"
#include "utils.h"

extern int batch_count;

enum program_args {
    ARG_AMQP_URL,
    ARG_GW_UNIX,
    ARG_GW_INET,
    ARG_DOMAIN,
    ARG_UNIX,
    ARG_BLOCK,
    ARG_STANDALONE,
    ARG_STAT_PERIOD,
    ARG_CID,
    ARG_COUNT,
    ARG_VERBOSE,
    ARG_HELP
};

struct option_info {
    struct option lopt;
    char *arg_example;
    char *arg_help;
    char *arg_default;
};

struct option_info option_info[] = {
    {{"amqp_url", required_argument, 0, ARG_AMQP_URL}, "host[:port]/path", "URL of the AMQP endpoint (%s)", DEFAULT_AMQP_URL},
    {{"gw_unix", optional_argument, 0, ARG_GW_UNIX}, "/path/to/socket", "Connect to gateway with unix socket (default)", DEFAULT_UNIX_SOCKET_PATH},
    {{"gw_inet", optional_argument, 0, ARG_GW_INET}, "host[:port]", "Connect to gateway with inet socket (unix is default)", DEFAULT_UNIX_SOCKET_PATH},
    {{"block", no_argument, 0, ARG_BLOCK}, "", "Outgoing connection blocking (%s)", DEFAULT_SOCKET_BLOCK},
    {{"stat_period", required_argument, 0, ARG_STAT_PERIOD}, "period_in_seconds", "How often to print stats, 0 for no stats (%s)", DEFAULT_STATS_PERIOD},
    {{"cid", required_argument, 0, ARG_CID}, "connection_id", "AMQP container ID (should be unique) (%s)", DEFAULT_CONTAINER_ID_PATTERN},
    {{"count", required_argument, 0, ARG_COUNT}, "stop_count", "Number of AMQP mesg to rcv before exit, 0 for continous (%s)", DEFAULT_STOP_COUNT},
    {{"verbose", no_argument, 0, ARG_VERBOSE}, "", "Print extra info, multiple instance increase verbosity.", ""},
    {{"help", no_argument, 0, ARG_HELP}, "", "Print help.", ""}};

static void usage(char *program) {
    fprintf(stdout,
            "usage: %s [OPTIONS]\n\n"
            "The missing link between AMQP and golang.\n\n",
            program);

    fprintf(stdout, "args:\n");
    for (int i = 0; i < (sizeof(option_info) / sizeof(option_info)); i++) {
        fprintf(stdout, "--%s %s %s\n", option_info[i].lopt.name, option_info[i].arg_example, option_info[i].arg_help);
    }
}

static int match_regex(char *regmatch, char *matches[], int n_matches, const char *to_match) {
    /* "M" contains the matches found. */
    regmatch_t m[n_matches];
    regex_t regex;

    if (regcomp(&regex, regmatch, REG_EXTENDED)) {
        fprintf(stderr, "Could not compile regex: %s\n", regmatch);

        return -1;
    }

    int nomatch = regexec(&regex, to_match, n_matches, m, 0);
    if (nomatch == REG_NOMATCH) {
        return 0;
    }

    int match_count = 0;
    for (int i = 0; i < n_matches; i++) {
        if (m[i].rm_so == -1) {
            continue;
        }
        match_count++;

        int match_len = m[i].rm_eo - m[i].rm_so;

        matches[i] = malloc(match_len + 1);  // make room for '\0'

        int k = 0;
        for (int j = m[i].rm_so; j < m[i].rm_eo; j++) {
            matches[i][k++] = to_match[j];
        }
        matches[i][k] = '\0';
    }

    return match_count;
}

int main(int argc, char **argv) {
    app_data_t app = {0};
    char cid_buf[100];
    int opt, index;

    srand(time(0));

    sprintf(cid_buf, DEFAULT_CID, rand() % 1024);

    app.stat_period = 0;        /* disabled */
    app.container_id = cid_buf; /* Should be unique */
    app.amqp_con.url = DEFAULT_AMQP_URL;
    app.message_count = 0;
    app.unix_socket_name = DEFAULT_UNIX_SOCKET_PATH;
    app.domain = AF_UNIX;
    app.socket_flags = MSG_DONTWAIT;
    app.peer_host = DEFAULT_INET_HOST;
    app.peer_port = DEFAULT_INET_PORT;

    int num_args = sizeof(option_info) / sizeof(struct option_info);
    struct option *longopts = malloc(sizeof(struct option) * num_args);
    for (int i = 0; i < num_args; i++) {
        longopts[i] = option_info[i].lopt;
    }

    while ((opt = getopt_long(argc, argv, "hv",
                              longopts, &index)) != -1) {
        switch (opt) {
            case ARG_BLOCK:
                app.socket_flags ^= MSG_DONTWAIT;
                break;
            case ARG_AMQP_URL:
                app.amqp_con.url = strdup(optarg);
                break;
            case ARG_GW_UNIX:
                if (optarg != NULL) {
                    app.unix_socket_name = optarg;
                }
                app.domain = AF_UNIX;
                break;
            case ARG_GW_INET:
                if (optarg != NULL) {
                    char *matches[4];
                    if (match_regex("^([^:]*)(:([0-9]+))*$", matches, 4, optarg) <= 0) {
                        fprintf(stderr, "Invalid INET address: %s", optarg);
                        exit(1);
                    }
                    app.peer_host = matches[2];
                    app.peer_port = matches[3];
                }
                app.domain = AF_INET;
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

    char *matches[10];
    memset(matches, 0, sizeof(matches));

    match_regex(AMQP_URL_REGEX,
                matches, 10, app.amqp_con.url);
    if (matches[2] != NULL) {
        app.amqp_con.user = strdup(matches[2]);
    }
    if (matches[4] != NULL) {
        app.amqp_con.password = strdup(matches[4]);
    }
    if (matches[5] == NULL || matches[8] == NULL) {
        fprintf(stderr,"Invalid AMQP URL: %s", app.amqp_con.url);
        exit(1);
    }
    app.amqp_con.host = strdup(matches[5]);
    app.amqp_con.address = strdup(matches[8]);
    if (matches[7] != NULL) {
        app.amqp_con.port = strdup(matches[7]);
    }
    

    if (app.standalone) {
        printf("standalone mode\n");
    } 

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
            printf("in: %ld(%ld), overrun: %ld(%ld), out: %ld(%ld), would_block: %ld\n",
                   app.amqp_received, app.amqp_received - last_amqp_received,
                   app.rbin->overruns, app.rbin->overruns - last_overrun,
                   app.sock_sent, app.sock_sent - last_out,
                   app.sock_would_block);
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