all: bridge

CC=gcc
CFLAGS=-Wall -O1
LIBS=-lqpid-proton 
INCLUDES=

BRIDGE = bridge

$(BRIDGE): $(BRIDGE).c
	$(CC) $(CFLAGS) -o $(BRIDGE) $(BRIDGE).c $(LIBS)

.PHONY: clean

clean:
	rm -f *.o $(BRIDGE)
