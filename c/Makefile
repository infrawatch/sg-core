all: bridge

CC=gcc
CFLAGS=-Wall
LIBS=-lqpid-proton 
INCLUDES=

BRIDGE = bridge

$(BRIDGE): $(BRIDGE).c
	$(CC) $(CFLAGS) -o $(BRIDGE) $(BRIDGE).c $(LIBS)