

CC=gcc
CFLAGS=-Wall -O3
LIBS=-lqpid-proton 
LDFLAGS=
INCLUDES=

SRC = $(wildcard *.c)
OBJ = $(SRC:.c=.o)

BRIDGE = bridge

all: $(BRIDGE)

$(BRIDGE): $(OBJ)
	$(CC) $(CFLAGS) $(INCLUDES) -o $(BRIDGE) $(OBJ) $(LDFLAGS) $(LIBS)

.c.o:
	$(CC) $(CFLAGS) $(INCLUDES) -c $<  -o $@

.PHONY: clean
clean:
	rm -f *.o $(BRIDGE)

depend: $(SRCS)
	makedepend $(INCLUDES) $^