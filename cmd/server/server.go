package main

import (
	"bufio"
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

	"github.com/atyronesmith/sa-benchmark/pkg/sharedserver"
	"github.com/atyronesmith/sa-benchmark/pkg/udpserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetBlockProfileRate(20)
		runtime.SetMutexProfileFraction(20)
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [options] udp [ip_address]\n", os.Args[0])
		flag.PrintDefaults()
	}

	netCommand := flag.NewFlagSet("net", flag.ExitOnError)
	sharedCommand := flag.NewFlagSet("shared", flag.ExitOnError)

	// Add Flags for net command
	// parse command line option
	port := netCommand.Int("port", 0, "Port to use, otherwise OS will choose")
	var cpuprofile = netCommand.String("cpuprofile", "", "write cpu profile to file")
	pport := netCommand.Int("pport", 8081, "Prometheus scrape port.")
	capture := netCommand.Bool("capture", false, "Catpure json output.")

	// Add Flags for shared command
	socketPath := sharedCommand.String("path", "/tmp/sg", "Path/file for the shared memeory socket")

	// Verify that a subcommand has been provided
	// os.Arg[0] is the main command
	// os.Arg[1] will be the subcommand
	if len(os.Args) < 2 {
		fmt.Println("net or shared subcommand is required")
		os.Exit(1)
	}

	// Switch on the subcommand
	// Parse the flags for appropriate FlagSet
	// FlagSet.Parse() requires a set of arguments to parse as input
	// os.Args[2:] will be all arguments starting after the subcommand at os.Args[1]
	switch os.Args[1] {
	case "net":
		err := netCommand.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
	case "shared":
		err := sharedCommand.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}

	var w *bufio.Writer
	var err error

	if *capture {
		var fo *os.File
		// open output file
		fo, err = os.Create("cd-capture.txt")
		if err != nil {
			panic(err)
		}
		// close fo on exit and check for its returned error
		defer func() {
			if err := fo.Close(); err != nil {
				panic(err)
			}
		}()
		// make a write buffer
		w = bufio.NewWriter(fo)
	}

	go func() {
		err := http.ListenAndServe(":"+strconv.Itoa(*pport), promhttp.Handler())
		if err != nil {
			fmt.Printf("http server failed!...")
			fmt.Printf("%+v\n", err)
		}
	}()

	ctx := context.Background()

	if netCommand.Parsed() {
		var ip net.IP

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
		netArgs := netCommand.Args()

		if len(netArgs) == 1 {
			ip = net.ParseIP(netArgs[0])
			if ip == nil {
				fmt.Fprintf(os.Stderr, "Invalid target IP addres %s...", ip)
				flag.Usage()
				os.Exit(1)
			}
		} else if len(netArgs) > 1 {
			fmt.Fprintln(os.Stderr, "Invalid number of arguments...")
			flag.Usage()
			os.Exit(1)
		} else {
			ip = net.ParseIP("127.0.0.1")
		}

		err = udpserver.Listen(ctx, ip.String()+":"+strconv.Itoa(*port), w)
		if err != nil {
			fmt.Printf("Error occurred")
		}
	} else if sharedCommand.Parsed() {
		err = sharedserver.Listen(ctx, *socketPath, w)
		if err != nil {
			fmt.Printf("Error occurred")
		}
	}

	if *capture {
		if err = w.Flush(); err != nil {
			panic(err)
		}
	}
}
