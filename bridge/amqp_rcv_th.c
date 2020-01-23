#define _GNU_SOURCE
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

#include <assert.h>
#include <pthread.h>
#include <stdio.h>
#include <time.h>

#include "bridge.h"

#define LISTEN_BACKLOG 16

static int exit_code = 0;

static time_t start_time;
int batch_count = 0;

/* Close the connection and the listener so so we will get a
 * PN_PROACTOR_INACTIVE event and exit, once all outstanding events
 * are processed.
 */
static void close_all(pn_connection_t *c, app_data_t *app) {
    if (c) pn_connection_close(c);
    if (app->listener) pn_listener_close(app->listener);
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

/* This function handles events when we are acting as the receiver */
static void handle_receive(app_data_t *app, pn_event_t *event, int *batch_done) {
    /*    printf("handle_receive %s\n", app->container_id);*/

    pn_delivery_t *d = pn_event_delivery(event);
    if (pn_delivery_readable(d)) {
        pn_link_t *l = pn_delivery_link(d);
        size_t size = pn_delivery_pending(d);

        pn_rwbytes_t *m = rb_get_head(app->rbin); /* Append data to incoming message buffer */
        assert(m);
        ssize_t recv;
        // First time through m->size = 0 for a partial message...
        size_t oldsize = m->size;
        m->size += size;
        recv = pn_link_recv(l, m->start + oldsize, size);
        if (recv == PN_ABORTED) {
            printf("Message aborted\n");
            fflush(stdout);
            m->size = 0;                         /* Forget the data we accumulated */
            pn_delivery_settle(d);               /* Free the delivery so we can
                                              receive the next message */
            pn_link_flow(l, 1);                  /* Replace credit for aborted message */
        } else if (recv < 0 && recv != PN_EOS) { /* Unexpected error */
            pn_condition_format(pn_link_condition(l), "broker",
                                "PN_DELIVERY error: %s", pn_code(recv));
            pn_link_close(l);                 /* Unexpected error, close the link */
        } else if (!pn_delivery_partial(d)) { /* Message is complete */
            // Place in the ring buffer HERE
            rb_put(app->rbin);

            app->amqp_received++;

            pn_delivery_update(d, PN_ACCEPTED);
            pn_delivery_settle(d); /* settle and free d */

            int inuse = rb_inuse_size(app->rbin);
            if (inuse > app->max_q_depth)
                app->max_q_depth = inuse;

            int link_credit = pn_link_credit(l);
            //pn_link_flow(l, rb_size(app->rbin) - link_credit);
             if (link_credit < rb_free_size(app->rbin)) {
                 *batch_done = link_credit;
                 pn_link_flow(l, rb_free_size(app->rbin) - link_credit);
             }
            if ((app->message_count > 0) && (app->sock_sent >= app->message_count)) {
                close_all(pn_event_connection(event), app);

                exit_code = 1;
            }
        } else {
            app->amqp_partial++;
        }
    }
}

/* Handle all events, delegate to handle_send or handle_receive depending on
   link mode. Return true to continue, false to exit
*/
static bool handle(app_data_t *app, pn_event_t *event, int *batch_done) {
    switch (pn_event_type(event)) {
        case PN_DELIVERY: {
            pn_link_t *l = pn_event_link(event);
            if (l) { /* Only delegate link-related events */
                    handle_receive(app, event, batch_done);
            }
            break;
        }

        case PN_LISTENER_OPEN: {
            char port[256]; /* Get the listening port */
            pn_netaddr_host_port(pn_listener_addr(pn_event_listener(event)),
                                 NULL, 0, port, sizeof(port));
            printf("listening on %s\n", port);
            fflush(stdout);
            break;
        }
        case PN_LISTENER_ACCEPT:
            pn_listener_accept2(pn_event_listener(event), NULL, NULL);
            break;

        case PN_CONNECTION_INIT:
            if (app->verbose) {
                printf("PN_CONNECTION_INIT %s\n", app->container_id);
            }
            pn_connection_t *c = pn_event_connection(event);
            pn_connection_set_container(c, app->container_id);
            pn_connection_open(c);
            pn_session_t *s = pn_session(c);
            pn_session_open(s);
            {
                pn_link_t *l = pn_receiver(s, "sa_receiver");
                pn_terminus_set_address(pn_link_source(l), app->amqp_address);
                pn_link_open(l);
                /* cannot receive without granting credit: */
                pn_link_flow(l, RING_BUFFER_COUNT);
            }
            break;

        case PN_CONNECTION_BOUND: {
            if (app->verbose) {
                printf("PN_CONNECTION_BOUND %s\n", app->container_id);
            }
            /* Turn off security */
            pn_transport_t *t = pn_event_transport(event);
            pn_transport_require_auth(t, false);
            pn_sasl_allowed_mechs(pn_sasl(t), "ANONYMOUS");
            break;
        }
        case PN_CONNECTION_LOCAL_OPEN: {
            if (app->verbose) {
                printf("PN_CONNECTION_LOCAL_OPEN %s\n", app->container_id);
            }
            break;
        }
        case PN_CONNECTION_REMOTE_OPEN: {
            if (app->verbose) {
                printf("PN_CONNECTION_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_connection_open(
                pn_event_connection(event)); /* Complete the open */
            break;
        }

        case PN_SESSION_LOCAL_OPEN: {
            if (app->verbose) {
                printf("PN_SESSION_LOCAL_OPEN %s\n", app->container_id);
            }
            pn_connection_t *c = pn_event_connection(event);
            pn_session_t *s = pn_session(c);
            pn_link_t *l = pn_receiver(s, "my_receiver");
            pn_terminus_set_address(pn_link_source(l), app->amqp_address);

            break;
        }
        case PN_SESSION_INIT: {
            if (app->verbose) {
                printf("PN_SESSION_INIT %s\n", app->container_id);
            }
            break;
        }
        case PN_SESSION_REMOTE_OPEN: {
            if (app->verbose) {
                printf("PN_SESSION_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_session_open(pn_event_session(event));
            break;
        }

        case PN_TRANSPORT_CLOSED:
            check_condition(
                event, pn_transport_condition(pn_event_transport(event)), app);
            break;

        case PN_CONNECTION_REMOTE_CLOSE:
            check_condition(
                event,
                pn_connection_remote_condition(pn_event_connection(event)),
                app);
            pn_connection_close(
                pn_event_connection(event)); /* Return the close */
            break;

        case PN_SESSION_REMOTE_CLOSE:
            check_condition(
                event, pn_session_remote_condition(pn_event_session(event)),
                app);
            pn_session_close(pn_event_session(event)); /* Return the close */
            pn_session_free(pn_event_session(event));
            break;

        case PN_LINK_REMOTE_CLOSE:
        case PN_LINK_REMOTE_DETACH:
            check_condition(
                event, pn_link_remote_condition(pn_event_link(event)), app);
            pn_link_close(pn_event_link(event)); /* Return the close */
            pn_link_free(pn_event_link(event));
            break;

        case PN_PROACTOR_TIMEOUT:
            break;

        case PN_LISTENER_CLOSE:
            app->listener = NULL; /* Listener is closed */
            check_condition(
                event, pn_listener_condition(pn_event_listener(event)), app);
            break;

        case PN_PROACTOR_INACTIVE:
            return false;
            break;

        default: {
            break;
        }
    }
    return exit_code == 0;
}

void run(app_data_t *app) {
    /* Loop and handle events */
    int batch_done = 0;

    printf("%s: %s start...\n", __FILE__, __func__);

    start_time = clock();

    do {
        batch_done = 0;
        pn_event_batch_t *events = pn_proactor_wait(app->proactor);
        pn_event_t *e;
        for (e = pn_event_batch_next(events); e;
             e = pn_event_batch_next(events)) {
            if (!handle(app, e, &batch_done)) {
                return;
            }
            if (batch_done) {
                break;
            }
            batch_count++;
        }
        pn_proactor_done(app->proactor, events);
    } while (true);
}

double amqp_rcv_clock() {
    time_t stop_time = clock();

    return (double)(stop_time - start_time) / CLOCKS_PER_SEC;
}

void amqp_rcv_th_cleanup(void *app_ptr) {
    app_data_t *app = (app_data_t *)app_ptr;

    if (app) {
        app->amqp_rcv_th_running = 0;
    }

    fprintf(stderr, "Exit AMQP RCV thread...\n");
}

void *amqp_rcv_th(void *app_ptr) {
    pthread_cleanup_push(amqp_rcv_th_cleanup, app_ptr);

    app_data_t *app = (app_data_t *)app_ptr;

    char addr[PN_MAX_ADDR];

    /* Create the proactor and connect */
    app->proactor = pn_proactor();
    if (app->standalone) {
        app->listener = pn_listener();
    }
    pn_proactor_addr(addr, sizeof(addr), app->host, app->port);
    fprintf(stdout, "Connecting to amqp addr: %s\n", addr);
    if (app->standalone) {
        pn_proactor_listen(app->proactor, app->listener, addr, LISTEN_BACKLOG);
    } else {
        /* Initialize Sasl transport */
        pn_transport_t *pnt = pn_transport();
        pn_sasl_set_allow_insecure_mechs(pn_sasl(pnt), true);
        pn_proactor_connect2(app->proactor, NULL, NULL, addr);
    }

    run(app);

    pn_proactor_free(app->proactor);

    pthread_cleanup_pop(1);

    return NULL;
}
