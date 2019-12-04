Build: gcc -L< directory with libqpid-proton.so > -Wall -o bridge bridge.c -lqpid-proton
Example Usage: ./bridge 127.0.0.1 5672 sg 0 127.0.0.1 5673