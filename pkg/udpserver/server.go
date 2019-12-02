package udpserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/atyronesmith/sa-benchmark/pkg/collectd"
)

const maxBufferSize = 1024

func Listen(ctx context.Context, address string) (err error) {

	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return
	}

	myAddr := pc.LocalAddr()
	fmt.Printf("Listening on %s\n", myAddr)

	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, maxBufferSize)

	count := 0

	go func() {
		cd := new(collectd.Collectd)

		for {
			n, _, err := pc.ReadFrom(buffer)
			if err != nil || n < 1 {
				doneChan <- err
				return
			}
			metric, err := cd.ParseInputByte(buffer)
			if err != nil {
				fmt.Printf("Error parsing JSON!\n")
				doneChan <- err
			}
			if (*metric)[0].Interval < 0.0 {
				doneChan <- err
			}
			count++
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("cancelled")
			err = ctx.Err()
			goto done
		case err = <-doneChan:
			goto done
		default:
			time.Sleep(time.Second * 5)
			fmt.Printf("Count %d\n", count)
		}
	}
done:
	return err
}
