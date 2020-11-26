package udpclient

import (
	"context"
	"fmt"
	"net"
	"time"
)

// SendMetrics ...
func SendMetrics(ctx context.Context, address string, count int, hostsCount int, mesg []byte) (err error) {
	// Resolve the UDP address so that we can make use of DialUDP
	// with an actual IP and port instead of a name (in case a
	// hostname is specified).
	raddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return
	}

	// Although we're not in a connection-oriented transport,
	// the act of `dialing` is analogous to the act of performing
	// a `connect(2)` syscall for a socket of type SOCK_DGRAM:
	// - it forces the underlying socket to only read and write
	//   to and from a specific remote address.
	conn, err := net.DialUDP("udp", nil, raddr)
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

	go func(conn *net.UDPConn, mesg []byte) {

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
