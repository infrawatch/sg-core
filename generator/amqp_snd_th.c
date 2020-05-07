#define _GNU_SOURCE
#include <assert.h>
#include <features.h>
#include <proton/connection.h>
#include <proton/delivery.h>
#include <proton/link.h>
#include <proton/listener.h>
#include <proton/message.h>
#include <proton/netaddr.h>
#include <proton/proactor.h>
#include <proton/session.h>
#include <proton/transport.h>
#include <proton/types.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include "gen.h"
#include "utils.h"

#define LISTEN_BACKLOG 16

static int exit_code = 0;

static time_t start_time;
int batch_count = 0;

static const pn_bytes_t SEND_TIME = {sizeof("SendTime") - 1, "SendTime"};

/* Close the connection and the listener so so we will get a
 * PN_PROACTOR_INACTIVE event and exit, once all outstanding events
 * are processed.
 */
static void close_all(pn_connection_t *c, app_data_t *app) {
    if (c) pn_connection_close(c);
}

static void check_condition(pn_event_t *e, pn_condition_t *cond,
                            app_data_t *app) {
    if (pn_condition_is_set(cond)) {
        fprintf(stderr, "%s: %s: %s\n", pn_event_type_name(pn_event_type(e)),
                pn_condition_get_name(cond),
                pn_condition_get_description(cond));
        close_all(pn_event_connection(e), app);
        exit_code = 1;
    }
}

char *CD_VALUES[] = {"1.0", "2.0", "3.0"};

char *CD_MSG =
    "{\"values\": [0.4593], \"dstypes\": [\"derive\"], \"dsnames\": [\"samples\"], \"time\": 1578337518.8668, \"interval\": 1,\
  \"host\": \"hostname270\", \"plugin\": \"metrics000\",\"plugin_instance\": \"pluginInst71\",\"type\": \"type0\",\"type_instance\": \"typInst0\"}";

char *JSON_MSG =
    "[{\"values\": [0.4593], \"dstypes\": [\"derive\"], \"dsnames\": [\"samples\"], \"time\": 1578337518.8668, \"interval\": 1,\
  \"host\": \"hostname270\", \"plugin\": \"metrics000\",\"plugin_instance\": \"pluginInst71\",\"type\": \"type0\",\"type_instance\": \"typInst0\"}]";

char *CD_MSG1 = "{\"values\": [";
char *CD_MSG2 = "], \"dstypes\": [\"derive\"], \"dsnames\": [\"samples\"], \"time\": ";
char *CD_MSG3 = ", \"interval\": 1,\"host\": \"";
char *CD_MSG4 = "\", \"plugin\": \"";
char *CD_MSG5 = "\", \"plugin_instance\": \"pluginInst0\",\"type\": \"type0\",\"type_instance\": \"typInst0\"}";

static char MSG_BUFFER[4096];
static char now_buf[100];

static char *build_mesg(app_data_t *app, char *time_buf) {
    char *p = MSG_BUFFER;
    char val_buff[20];

    *p++ = '[';

    for (int i = 0; i < app->num_cd_per_mesg;) {
        p = memccpy(p, CD_MSG1, '\0', sizeof(MSG_BUFFER));
        p--;
        sprintf(val_buff, "%ld", app->host_list[app->curr_host].count++);
        p = memccpy(p, val_buff, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, CD_MSG2, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, time_buf, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, CD_MSG3, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, app->host_list[app->curr_host].hostname, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, CD_MSG4, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, app->host_list[app->curr_host].metric, '\0', sizeof(MSG_BUFFER));
        p--;
        p = memccpy(p, CD_MSG5, '\0', sizeof(MSG_BUFFER));
        p--;

        if (++i < app->num_cd_per_mesg) {
            *p++ = ',';
        }

        app->curr_host++;
        if (app->curr_host == (app->host_list_len - 1))
            app->curr_host = 0;
    }

    *p++ = ']';
    *p = '\0';

    return MSG_BUFFER;
}

static void gen_mesg(pn_rwbytes_t *buf, app_data_t *app, char *time_buf) {
    buf->start = build_mesg(app, time_buf);

    buf->size = strlen(buf->start);
}

/* Create a message with a map { "sequence" : number } encode it and return the
 * encoded buffer. */
static void send_message(app_data_t *app, pn_link_t *sender, pn_rwbytes_t *data) {
    /* Construct a message with the map { "sequence": app.sent } */
    // Use a static message with pn_message_clear(...)
    pn_message_t *message;

    if ((message = app->message) == NULL) {
        app->message = pn_message();
        message = app->message;
    } else {
        pn_message_clear(message);
    }

    int64_t stime = now();

    pn_data_t *props = pn_message_properties(message);
    pn_data_clear(props);
    pn_data_put_map(props);
    pn_data_enter(props);
    pn_data_put_string(props, pn_bytes(SEND_TIME.size, SEND_TIME.start));
    pn_data_put_long(props, stime);
    pn_data_exit(props);

    pn_data_t *body = pn_message_body(message);
    pn_data_clear(body);
    pn_data_put_binary(body, pn_bytes(data->size, data->start));
    pn_data_exit(body);

    //    pn_data_put_int(pn_message_id(message),
    //                    app->sent); /* Set the message_id also */
    if (pn_message_send(message, sender, &app->msgout) < 0) {
        fprintf(stderr, "send error: %s\n",
                pn_error_text(pn_message_error(message)));
        exit_code = 1;
    }
}

static bool send_burst(app_data_t *app, pn_event_t *event) {
//    pn_link_t *sender = pn_event_link(event);
    pn_link_t *sender = app->sender;

    int credits = pn_link_credit(sender);
    if ( credits <= 10 ) {
        return 0;
    }
    /* The peer has given us some credit, now we can send messages */
    int burst = 0;

    struct timespec now;

    clock_gettime(CLOCK_REALTIME, &now);
    time_sprintf(now_buf, now);

    app->total_bursts++;
    app->burst_credit += credits;
    while (pn_link_credit(sender) > 0) {
        if (app->message_count > 0 && app->metrics_sent == app->message_count) {
            break;
        }
        app->amqp_sent++;
        app->metrics_sent += app->num_cd_per_mesg;

        /* Use sent counter as unique delivery tag. */
        pn_delivery(sender, pn_dtag((const char *)&app->metrics_sent,
                                    sizeof(app->metrics_sent)));
        pn_rwbytes_t data;

        gen_mesg(&data, app, now_buf);
        send_message(app, sender, &data);
        if (app->burst_size > 0 && ++burst >= app->burst_size) {
            break;
        }
    }
    
    if (app->sleep_usec)
       usleep(app->sleep_usec);

    return 0;
}

/* Handle all events, delegate to handle_send or handle_receive depending on
   link mode. Return true to continue, false to exit
*/
static bool handle(app_data_t *app, pn_event_t *event) {
    switch (pn_event_type(event)) {
        case PN_LINK_FLOW: {
            pn_link_t *sender = pn_event_link(event);
            if (app->verbose > 1) {
                printf("PN_LINK_FLOW %d\n", pn_link_credit(sender));
            }
            send_burst(app, event);
            break;
        }

        case PN_LINK_REMOTE_OPEN: {
            if (app->verbose > 1) {
                printf("PN_LINK_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_link_t *l = pn_event_link(event);
            pn_terminus_t *t = pn_link_target(l);
            pn_terminus_set_address(t, app->amqp_address);
            pn_link_open(l);
        } break;

        case PN_DELIVERY: {
            if (app->verbose > 2) {
                printf("send PN_DELIVERY %s\n", app->container_id);
            }
            /* We received acknowledgement from the peer that a message was
             * delivered. */
            pn_delivery_t *d = pn_event_delivery(event);

            if (pn_delivery_remote_state(d) == PN_ACCEPTED) {
                pn_delivery_settle(d);
                app->acknowledged += app->num_cd_per_mesg;
                if (app->acknowledged == app->message_count) {
                    printf("%ld messages metrics_sent and acknowledged\n",
                           app->acknowledged);
                    close_all(pn_event_connection(event), app);
                }
            }
        } break;

        case PN_LISTENER_OPEN: {
            char port[256]; /* Get the listening port */
            pn_netaddr_host_port(pn_listener_addr(pn_event_listener(event)),
                                 NULL, 0, port, sizeof(port));
            if (app->verbose > 0) {
                printf("listening on %s\n", port);
            }
            fflush(stdout);
            break;
        }
        case PN_LISTENER_ACCEPT:
            pn_listener_accept2(pn_event_listener(event), NULL, NULL);
            break;

        case PN_CONNECTION_INIT:
            if (app->verbose > 1) {
                printf("PN_CONNECTION_INIT %s\n", app->container_id);
            }
            pn_connection_t *c = pn_event_connection(event);
            pn_connection_set_container(c, app->container_id);
            //pn_connection_open(c);

            pn_session_t *s = pn_session(c);
            pn_session_open(s);
            {
                char link_name[30];
                rand_str(link_name,16,"sa-gen-");
                pn_link_t *sender = pn_sender(s, link_name);
                app->sender = sender;
                pn_terminus_set_address(pn_link_target(sender), app->amqp_address);
                pn_link_set_snd_settle_mode(sender, PN_SND_UNSETTLED);
                pn_link_set_rcv_settle_mode(sender, PN_RCV_FIRST);
                pn_link_open(sender);
            }
            break;

        case PN_CONNECTION_WAKE:
            if (app->verbose > 1) {
                printf("PN_CONNECTION_WAKE %s\n", app->container_id);
            }
            break;

        case PN_CONNECTION_BOUND: {
            if (app->verbose > 1) {
                printf("PN_CONNECTION_BOUND %s\n", app->container_id);
            }
            /* Turn off security */
            pn_transport_t *t = pn_event_transport(event);
            pn_transport_require_auth(t, false);
            pn_sasl_allowed_mechs(pn_sasl(t), "ANONYMOUS");
            //pn_sasl_set_allow_insecure_mechs(pn_sasl(t), true);

            break;
        }
        case PN_CONNECTION_LOCAL_OPEN: {
            if (app->verbose > 1) {
                printf("PN_CONNECTION_LOCAL_OPEN %s\n", app->container_id);
            }
            break;
        }
        case PN_CONNECTION_REMOTE_OPEN: {
            if (app->verbose > 1) {
                printf("PN_CONNECTION_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_connection_open(
                pn_event_connection(event)); /* Complete the open */
            break;
        }

        case PN_SESSION_LOCAL_OPEN: {
            if (app->verbose > 1) {
                printf("PN_SESSION_LOCAL_OPEN %s\n", app->container_id);
            }
            // pn_connection_t *c = pn_event_connection(event);
            // pn_session_t *s = pn_session(c);
            // pn_link_t *l = pn_receiver(s, "my_receiver");
            // pn_terminus_set_address(pn_link_source(l), app->amqp_address);

            break;
        }
        case PN_SESSION_INIT: {
            if (app->verbose > 1) {
                printf("PN_SESSION_INIT %s\n", app->container_id);
            }
            break;
        }
        case PN_SESSION_REMOTE_OPEN: {
            if (app->verbose > 1) {
                printf("PN_SESSION_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_session_open(pn_event_session(event));
            break;
        }

        case PN_TRANSPORT_CLOSED:
            if (app->verbose > 1) {
                printf("PN_TRANSPORT_CLOSED %s\n", app->container_id);
            }
            check_condition(
                event, pn_transport_condition(pn_event_transport(event)), app);
            break;

        case PN_CONNECTION_REMOTE_CLOSE:
            if (app->verbose > 1) {
                printf("PN_CONNECTION_REMOTE_CLOSE %s\n", app->container_id);
            }
            check_condition(
                event,
                pn_connection_remote_condition(pn_event_connection(event)),
                app);
            pn_connection_close(
                pn_event_connection(event)); /* Return the close */
            break;

        case PN_SESSION_REMOTE_CLOSE:
            if (app->verbose > 1) {
                printf("PN_SESSION_REMOTE_CLOSE %s\n", app->container_id);
            }
            check_condition(
                event, pn_session_remote_condition(pn_event_session(event)),
                app);
            pn_session_close(pn_event_session(event)); /* Return the close */
            pn_session_free(pn_event_session(event));
            break;

        case PN_LINK_REMOTE_CLOSE:
        case PN_LINK_REMOTE_DETACH:
            if (app->verbose > 1) {
                printf("PN_LINK_REMOTE_DETACH %s\n", app->container_id);
            }
            check_condition(
                event, pn_link_remote_condition(pn_event_link(event)), app);
            pn_link_close(pn_event_link(event)); /* Return the close */
            pn_link_free(pn_event_link(event));
            break;

        case PN_PROACTOR_TIMEOUT:
            if (app->verbose > 1) {
                printf("PN_PROACTOR_TIMEOUT %s\n", app->container_id);
            }
            /* Wake the sender's connection */
            pn_connection_wake(
                 pn_session_connection(pn_link_session(app->sender)));
            break;

        case PN_PROACTOR_INACTIVE:
            if (app->verbose > 1) {
                printf("PN_PROACTOR_INACTIVE %s\n", app->container_id);
            }
            return false;
            break;

        default: {
            if (app->verbose > 2) {
                printf("Unhandled eventtype: %s\n", pn_event_type_name(pn_event_type(event)));
            }
            break;
        }
    }
    return exit_code == 0;
}

void run(app_data_t *app) {
    /* Loop and handle events */
    if (app->verbose) {
        printf("%s: %s(..) start...\n", __FILE__, __func__);
    }

    start_time = clock();

    do {
        pn_event_batch_t *events = pn_proactor_wait(app->proactor);
        pn_event_t *e;
        for (e = pn_event_batch_next(events); e;
             e = pn_event_batch_next(events)) {
            if (!handle(app, e)) {
                return;
            }
            batch_count++;
        }
        pn_proactor_done(app->proactor, events);
    } while (true);
}

double amqp_snd_clock() {
    time_t stop_time = clock();

    return (double)(stop_time - start_time) / CLOCKS_PER_SEC;
}

void amqp_snd_th_cleanup(void *app_ptr) {
    app_data_t *app = (app_data_t *)app_ptr;

    if (app) {
        app->amqp_snd_th_running = 0;
    }

    fprintf(stderr, "Exit AMQP SND thread...\n");
}

void *amqp_snd_th(void *app_ptr) {
    pthread_cleanup_push(amqp_snd_th_cleanup, app_ptr);

    app_data_t *app = (app_data_t *)app_ptr;

    char addr[PN_MAX_ADDR];
    pn_proactor_addr(addr, sizeof(addr), app->host, app->port);

    /* Create the proactor and connect */
    app->proactor = pn_proactor();

    pn_proactor_connect2(app->proactor, NULL, NULL, addr);

    run(app);

    pn_proactor_free(app->proactor);

    pthread_cleanup_pop(1);

    return NULL;
}
