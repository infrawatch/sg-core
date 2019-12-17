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

static const int BATCH = 100; /* Batch size for unlimited receive */
static const int MIN_CREDIT = 20;

#define LISTEN_BACKLOG 16

static int exit_code = 0;

static time_t start_time;

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

/* Create a message with a map { "sequence" : number } encode it and return the
 * encoded buffer. */
static void send_message(app_data_t *app, pn_link_t *sender) {
    /* Construct a message with the map { "sequence": app.sent } */
    pn_message_t *message = pn_message();
    pn_data_t *body = pn_message_body(message);
    pn_data_put_int(pn_message_id(message),
                    app->sent); /* Set the message_id also */
    pn_data_put_map(body);
    pn_data_enter(body);
    pn_data_put_string(body, pn_bytes(sizeof("sequence") - 1, "sequence"));
    pn_data_put_int(body, app->sent); /* The sequence number */
    pn_data_exit(body);
    if (pn_message_send(message, sender, &app->msgout) < 0) {
        fprintf(stderr, "send error: %s\n",
                pn_error_text(pn_message_error(message)));
        exit_code = 1;
    }
    pn_message_free(message);
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
            //decode_message(*m);

            pn_delivery_update(d, PN_ACCEPTED);
            pn_delivery_settle(d); /* settle and free d */

            int qd = pn_link_queued(l);
            if (qd > app->max_amqp_queue_depth)
                app->max_amqp_queue_depth = qd;

            int link_credit = pn_link_credit(l);
            //pn_link_flow(l, rb_size(app->rbin) - link_credit);
             if (link_credit < 100) {
                 *batch_done = link_credit;
                 pn_link_flow(l, rb_free_size(app->rbin));
             }
            if ((app->message_count > 0) && (app->received >= app->message_count)) {
                close_all(pn_event_connection(event), app);

                exit_code = 1;
            }
        } else {
            printf("partial\n");
        }
    }
}

/* This function handles events when we are acting as the sender */
static void handle_send(app_data_t *app, pn_event_t *event) {
    switch (pn_event_type(event)) {
        case PN_LINK_REMOTE_OPEN: {
            if (app->verbose) {
                printf("send PN_LINK_REMOTE_OPEN %s\n", app->container_id);
            }
            pn_link_t *l = pn_event_link(event);
            pn_terminus_set_address(pn_link_target(l), app->amqp_address);
            pn_link_open(l);
        } break;

        case PN_LINK_FLOW: {
            /* The peer has given us some credit, now we can send messages */
            pn_link_t *sender = pn_event_link(event);
            while (pn_link_credit(sender) > 0 &&
                   app->sent < app->message_count) {
                ++app->sent;
                /* Use sent counter as unique delivery tag. */
                pn_delivery(sender, pn_dtag((const char *)&app->sent,
                                            sizeof(app->sent)));
                send_message(app, sender);
            }
            break;
        }

        case PN_DELIVERY: {
            /* We received acknowledgement from the peer that a message was
             * delivered. */
            pn_delivery_t *d = pn_event_delivery(event);
            if (pn_delivery_remote_state(d) == PN_ACCEPTED) {
                if (++app->acknowledged == app->message_count) {
                    printf("%d messages sent and acknowledged\n",
                           app->acknowledged);
                    close_all(pn_event_connection(event), app);
                }
            }
        } break;

        default:
            break;
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
                if (pn_link_is_sender(l)) {
                    handle_send(app, event);
                } else {
                    handle_receive(app, event, batch_done);
                }
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
                pn_link_flow(l, BATCH);
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
            /* Wake the sender's connection */
            pn_connection_wake(
                pn_session_connection(pn_link_session(app->sender)));
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

void time_diff(struct timespec t1, struct timespec t2, struct timespec *diff) {
    if(t2.tv_nsec < t1.tv_nsec)
	{
		/* If nanoseconds in t1 are larger than nanoseconds in t2, it
		   means that something like the following happened:
		   t1.tv_sec = 1000    t1.tv_nsec = 100000
		   t2.tv_sec = 1001    t2.tv_nsec = 10
		   In this case, less than a second has passed but subtracting
		   the tv_sec parts will indicate that 1 second has passed. To
		   fix this problem, we subtract 1 second from the elapsed
		   tv_sec and add one second to the elapsed tv_nsec. See
		   below:
		*/
		diff->tv_sec  += t2.tv_sec  - t1.tv_sec  - 1;
		diff->tv_nsec += t2.tv_nsec - t1.tv_nsec + 1000000000;
	}
	else
	{
		diff->tv_sec  += t2.tv_sec  - t1.tv_sec;
		diff->tv_nsec += t2.tv_nsec - t1.tv_nsec;
	}

}
void run(app_data_t *app) {
    /* Loop and handle events */
    int batch_done = 0;

    printf("%s: %s start...\n", __FILE__, __func__);

    
    start_time = clock();

    do {
        int batch_count =0;
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
