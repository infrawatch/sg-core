package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/infrawatch/sg2/pkg/collectd"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-c count] unix_address \n", os.Args[0])
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
}

func sendMetrics(ctx context.Context, address string, count int, hostsCount int, mesg []byte) (err error) {
	// Resolve the UDP address so that we can make use of DialUDP
	// with an actual IP and port instead of a name (in case a
	// hostname is specified).
	raddr, err := net.ResolveUnixAddr("unixgram", address)
	if err != nil {
		return
	}

	// // Although we're not in a connection-oriented transport,
	// // the act of `dialing` is analogous to the act of performing
	// // a `connect(2)` syscall for a socket of type SOCK_DGRAM:
	// // - it forces the underlying socket to only read and write
	// //   to and from a specific remote address.
	conn, err := net.DialUnix("unixgram", nil, raddr)
	if err != nil {
		return
	}

	// Closes the underlying file descriptor associated with the,
	// socket so that it no longer refers to any file.
	defer conn.Close()

	doneChan := make(chan error, 1)
	sent := 0
	bytesSent := 0
	var start time.Time
	var end time.Time

	go func(conn *net.UnixConn, mesg []byte) {

		start = time.Now()

		for i := 0; i < count; i++ {
			_, err := conn.Write(mesg)
			if err != nil {
				doneChan <- err
				return
			}
			sent++
			bytesSent += len(mesg)
		}
		end = time.Now()
		doneChan <- nil
	}(conn, mesg)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("cancelled")
			err = ctx.Err()
			goto done
		case err = <-doneChan:
			fmt.Printf("Sent...%d\n", sent)
			goto done
		default:
			time.Sleep(time.Second * 1)
			fmt.Printf("Sending...%d\n", sent)
			if sent >= count {
				goto done
			}
		}
	}
done:
	diff := end.Sub(start)
	fmt.Printf("Send %d messages in %.8v, %.4f Mbps\n", sent, diff, float64(bytesSent)/diff.Seconds()*8.0/1000000.0)

	return
}

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(20)
		runtime.SetMutexProfileFraction(20)
	}
	// parse command line option
	hostsNum := flag.Int("hosts", 1, "Number of hosts to simulate")
	metricPerMsg := flag.Int("mpm", 1, "Number of metrics per messsage")
	msgCount := flag.Int("count", 1000000, "Number of metrics to send")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Usage = usage
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	args := flag.Args()

	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Invalid number of arguments...")
		usage()
		os.Exit(1)
	}

	addr := args[0]

	metric := collectd.GenCPUMetric(10, "Goblin", *metricPerMsg)

	ctx := context.Background()

	err := sendMetrics(ctx, addr, *msgCount, *hostsNum, metric)
	if err != nil {
		fmt.Printf("Error occurred: %s\n", err)
	}
}
