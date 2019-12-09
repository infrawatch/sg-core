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
#include <stdio.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>

#include <stdio.h>
#include <stdlib.h>

#include <time.h>

typedef struct app_data_t {
    const char *host, *port;
    const char *amqp_address;
    const char *container_id;
    int message_count;

    pn_proactor_t *proactor;
    pn_listener_t *listener;
    pn_rwbytes_t msgin, msgout; /* Buffers for incoming/outgoing messages */

    /* Sender values */
    int sent;
    int acknowledged;
    pn_link_t *sender;

    /* Receiver values */
    int received;
} app_data_t;

static const int BATCH = 1000; /* Batch size for unlimited receive */

static int exit_code = 0;

static int send_sock = -1;
static char *peer_host;
static char *peer_port;
static struct addrinfo *peer_addrinfo = 0;

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

static void decode_message(pn_rwbytes_t data) {
    pn_message_t *m = pn_message();
    int err = pn_message_decode(m, data.start, data.size);
    if (!err) {
        /* Print the decoded message */
        pn_string_t *s = pn_string(NULL);
        pn_inspect(pn_message_body(m), s);
/*        printf("%s\n", pn_string_get(s));*/

        int send_flags = MSG_DONTWAIT;

        // Get the string version of the message, and strip the leading
        // b" and trailing " from it
        const char *msg = pn_string_get(s);
        int msg_len = strlen(msg) - 3;
        char *msg2;

        if (msg_len > 2) {
            msg2 = (char *)calloc(msg_len, sizeof(char));
        } else {
            msg2 = strdup(msg);
        }

        memcpy(msg2, &msg[2], strlen(msg) - 1);
        msg2[strlen(msg2) - 1] = '\0';

        int sent_bytes =
            sendto(send_sock, msg2, strlen(msg2), send_flags,
                   peer_addrinfo->ai_addr, peer_addrinfo->ai_addrlen);

        char addrstr[100];
        void *ptr;
/*
        ptr = &((struct sockaddr_in *)peer_addrinfo->ai_addr)->sin_addr;
        inet_ntop(peer_addrinfo->ai_family, ptr, addrstr, 100);

        printf("Message (%d bytes) forwarded to %s:%d: %s\n", sent_bytes,
               addrstr,
               ntohs((((struct sockaddr_in *)((struct sockaddr *)
                                                  peer_addrinfo->ai_addr))
                          ->sin_port)),
               msg2);

        free(msg2);

        if (sent_bytes < 0) {
            fprintf(stderr, "socket send error: %d\n", errno);
            perror("Error");
            exit_code = 1;
        }
*/
        fflush(stdout);
        pn_free(s);
        pn_message_free(m);
        free(data.start);
    } else {
        fprintf(stderr, "decode error: %s\n",
                pn_error_text(pn_message_error(m)));
        exit_code = 1;
    }
}

/* This function handles events when we are acting as the receiver */
static void handle_receive(app_data_t *app, pn_event_t *event) {
/*    printf("handle_receive %s\n", app->container_id);*/

    switch (pn_event_type(event)) {
        case PN_LINK_INIT: {
            printf("PN_LINK_INIT %s\n", app->container_id);
        } break;
        case PN_LINK_LOCAL_OPEN: {
            printf("PN_LINK_LOCAL_OPEN %s\n", app->container_id);
        } break;
        case PN_LINK_REMOTE_OPEN: {
            printf("PN_LINK_REMOTE_OPEN %s\n", app->container_id);
        } break;

        case PN_DELIVERY: { /* Incoming message data */
            pn_delivery_t *d = pn_event_delivery(event);
            if (pn_delivery_readable(d)) {
                pn_link_t *l = pn_delivery_link(d);
                size_t size = pn_delivery_pending(d);
                pn_rwbytes_t *m =
                    &app->msgin; /* Append data to incoming message buffer */
                ssize_t recv;
                m->size += size;
                m->start = (char *)realloc(m->start, m->size);
                recv = pn_link_recv(l, m->start, m->size);
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
                    decode_message(*m);
                    *m = pn_rwbytes_null;
                    pn_delivery_update(d, PN_ACCEPTED);
                    pn_delivery_settle(d); /* settle and free d */
                    if (app->message_count == 0) {
                        /* receive forever - see if more credit is needed */
                        if (pn_link_credit(l) < BATCH / 2) {
                            pn_link_flow(l, BATCH - pn_link_credit(l));
                        }
                    } else if (++app->received >= app->message_count) {
                        printf("%d messages received\n", app->received);
                        close_all(pn_event_connection(event), app);
                    }
                }
            }
            break;
        }
        default:
            break;
    }
}

/* This function handles events when we are acting as the sender */
static void handle_send(app_data_t *app, pn_event_t *event) {
    switch (pn_event_type(event)) {
        case PN_LINK_REMOTE_OPEN: {
            printf("send PN_LINK_REMOTE_OPEN %s\n", app->container_id);

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
static bool handle(app_data_t *app, pn_event_t *event) {
    pn_event_type_t eType = pn_event_type(event);
    switch (pn_event_type(event)) {
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
            printf("PN_CONNECTION_INIT %s\n", app->container_id);
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
                pn_link_flow(l, app->message_count ? app->message_count : BATCH);
            }
            break;

        case PN_CONNECTION_BOUND: {
            printf("PN_CONNECTION_BOUND %s\n", app->container_id);
            /* Turn off security */
            pn_transport_t *t = pn_event_transport(event);
            pn_transport_require_auth(t, false);
            pn_sasl_allowed_mechs(pn_sasl(t), "ANONYMOUS");
            break;
        }
        case PN_CONNECTION_LOCAL_OPEN: {
            printf("PN_CONNECTION_LOCAL_OPEN %s\n", app->container_id);
            break;
        }
        case PN_CONNECTION_REMOTE_OPEN: {
            printf("PN_CONNECTION_REMOTE_OPEN %s\n", app->container_id);

            pn_connection_open(
                pn_event_connection(event)); /* Complete the open */
            break;
        }

        case PN_SESSION_LOCAL_OPEN: {
            printf("PN_SESSION_LOCAL_OPEN %s\n", app->container_id);
            pn_connection_t *c = pn_event_connection(event);

            pn_session_t *s = pn_session(c);
            pn_link_t *l = pn_receiver(s, "my_receiver");
            pn_terminus_set_address(pn_link_source(l), app->amqp_address);

            break;
        }
        case PN_SESSION_INIT: {
            printf("PN_SESSION_INIT %s\n", app->container_id);
            break;
        }
        case PN_SESSION_REMOTE_OPEN: {
            printf("PN_SESSION_REMOTE_OPEN %s\n", app->container_id);
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
            pn_link_t *l = pn_event_link(event);
            if (l) { /* Only delegate link-related events */
                if (pn_link_is_sender(l)) {
                    handle_send(app, event);
                } else {
                    handle_receive(app, event);
                }
            }
        }
    }
    return exit_code == 0;
}

void run(app_data_t *app) {
    /* Loop and handle events */
    do {
        pn_event_batch_t *events = pn_proactor_wait(app->proactor);
        pn_event_t *e;
        for (e = pn_event_batch_next(events); e;
             e = pn_event_batch_next(events)) {
            if (!handle(app, e)) {
                return;
            }
        }
        pn_proactor_done(app->proactor, events);
    } while (true);
}

static int prepare_send_socket() {
    struct addrinfo hints = {
        .ai_family = AF_UNSPEC,
        .ai_socktype = SOCK_DGRAM,
        .ai_protocol = 0,
        .ai_flags = AI_ADDRCONFIG,
    };

    int err = getaddrinfo(peer_host, peer_port, &hints, &peer_addrinfo);

    if (err != 0) {
        fprintf(
            stderr,
            "prepare_send_socket: getaddrinfo returned non-zero value: %d\n",
            errno);
        perror("Error");
        freeaddrinfo(peer_addrinfo);
        return -1;
    }

    send_sock = socket(peer_addrinfo->ai_family, peer_addrinfo->ai_socktype,
                       peer_addrinfo->ai_protocol);
    if (send_sock == -1) {
        fprintf(stderr, "prepare_send_socket: socket returned -1\n");
        perror("Error");
        freeaddrinfo(peer_addrinfo);
        return -1;
    }

    // freeaddrinfo(res);

    fprintf(stdout, "Socket %d opened\n", send_sock);

    return 0;
}

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
    struct app_data_t app = {0};
    char addr[PN_MAX_ADDR];
    char cid_buf[100];
    int opt;
    int standalone = 0;

    srand(time(0));

    sprintf(cid_buf, "sa-%x", rand() % 1024);

    app.container_id = cid_buf; /* Should be unique */

    app.host = "127.0.0.1";
    app.port = "5672";
    app.amqp_address = "collectd/telemetry";
    app.message_count = 0;

    while ((opt = getopt(argc, argv, "i:a:c:sh")) != -1) {
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
                standalone = 1;
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
    peer_host = strdup(argv[optind++]);
    peer_port = strdup(argv[optind++]);

    // Create the send socket
    if (prepare_send_socket() == -1) {
        fprintf(stderr, "Failed to create socket -- exiting!");
        return 1;
    }

    /* Create the proactor and connect */
    app.proactor = pn_proactor();
    if (standalone) {
        app.listener = pn_listener();
    }
    pn_proactor_addr(addr, sizeof(addr), app.host, app.port);
    fprintf(stdout, "Connecting to host: %s\n", addr);
    if (standalone) {
        pn_proactor_listen(app.proactor, app.listener, addr, 16);
    } else {
        /* Initialize Sasl transport */
        pn_transport_t *pnt = pn_transport();
        pn_sasl_set_allow_insecure_mechs(pn_sasl(pnt), true);
        pn_proactor_connect2(app.proactor, NULL, NULL, addr);
    }
    run(&app);
    pn_proactor_free(app.proactor);
    free(app.msgout.start);
    free(app.msgin.start);

    if (send_sock != -1) {
        close(send_sock);
        fprintf(stdout, "Socket closed\n");
    }

    free(peer_host);
    free(peer_port);
    freeaddrinfo(peer_addrinfo);

    return exit_code;
}