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
	"strconv"

	"github.com/infrawatch/sg2/pkg/collectd"
	"github.com/infrawatch/sg2/pkg/udpclient"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [-c count] ip_address port \n", os.Args[0])
	fmt.Fprintf(os.Stderr, "options:\n")
	flag.PrintDefaults()
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

	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Invalid number of arguments...")
		usage()
		os.Exit(1)
	}

	destIp := args[0]
	netIP := net.ParseIP(destIp)
	if netIP == nil {
		fmt.Fprintf(os.Stderr, "Invalid target IP addres %s...", destIp)
		usage()
		os.Exit(1)
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid target port %s...", args[1])
		usage()
		os.Exit(1)
	}

	metric := collectd.GenCPUMetric(10, "Goblin", *metricPerMsg)

	ctx := context.Background()

	err = udpclient.SendMetrics(ctx, netIP.String()+":"+strconv.Itoa(port), *msgCount, *hostsNum, metric)
	if err != nil {
		fmt.Printf("Error occurred")
	}
}
