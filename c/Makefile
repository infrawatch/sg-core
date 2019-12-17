

CC=gcc
CFLAGS=-Wall -O3
LIBS=-lqpid-proton -lpthread
LDFLAGS=
INCLUDES=amqp_rcv_th.h  bridge.h  rb.h  socket_snd_th.h

SRC = $(wildcard *.c)
OBJ = $(SRC:.c=.o)

BRIDGE = bridge

all: $(BRIDGE)

$(BRIDGE): $(OBJ) $(INCLUDES)
	$(CC) $(CFLAGS) $(INCLUDES) -o $(BRIDGE) $(OBJ) $(LDFLAGS) $(LIBS)

.c.o:
	$(CC) $(CFLAGS) -c $<  -o $@

.PHONY: clean
clean:
	rm -f *.o $(BRIDGE)

depend: $(SRCS)
	makedepend $(INCLUDES) $^