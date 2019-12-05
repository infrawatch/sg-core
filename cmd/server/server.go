package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"

	"github.com/atyronesmith/sa-benchmark/pkg/udpserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] [ip_address]\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(20)
		runtime.SetMutexProfileFraction(20)
	}

	// parse command line option
	port := flag.Int("port", 0, "Port to use, otherwise OS will choose")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	pport := flag.Int("pport", 8081, "Prometheus scrape port.")

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

	var ip net.IP

	if len(args) == 1 {
		ip = net.ParseIP(args[0])
		if ip == nil {
			fmt.Fprintf(os.Stderr, "Invalid target IP addres %s...", ip)
			usage()
			os.Exit(1)
		}
	} else if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Invalid number of arguments...")
		usage()
		os.Exit(1)
	} else {
		ip = net.ParseIP("127.0.0.1")
	}

	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(*pport), promhttp.Handler())
		if err != nil {
			fmt.Printf("http server failed!...")
			fmt.Printf("%+v\n", err)
		}
	}()

	ctx := context.Background()
	err := udpserver.Listen(ctx, ip.String()+":"+strconv.Itoa(*port))

	if err != nil {
		fmt.Printf("Error occurred")
	}
}
