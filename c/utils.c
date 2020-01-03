#include "utils.h"
#include <stdio.h>

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

char *time_sprintf(char *buf, struct timespec t1) {
	double pct = (t1.tv_sec * 1000000000.0) + t1.tv_nsec / 1000000000.0;

	sprintf(buf, "%f", pct);

	return buf;
}