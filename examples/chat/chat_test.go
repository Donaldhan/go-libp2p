package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"

	"github.com/libp2p/go-libp2p/examples/testutils"
)

func TestMain(t *testing.T) {
	var h testutils.LogHarness
	h.Expect("Waiting for incoming connection")
	h.Expect("Established connection to destination")
	h.Expect("Got a new stream!")

	h.Run(t, func() {
		// Create a context that will stop the hosts when the tests end
		//创建带取消功能的host上下文
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() //退出关闭host

		port1, err := testutils.FindFreePort(t, "", 5)
		if err != nil {
			log.Println(err)
			return
		}

		port2, err := testutils.FindFreePort(t, "", 5)
		if err != nil {
			log.Println(err)
			return
		}
		//创建host
		h1, err := makeHost(port1, rand.Reader)
		if err != nil {
			log.Println(err)
			return
		}
		//开启server
		go startPeer(ctx, h1, func(network.Stream) {
			log.Println("Got a new stream!")
			cancel() // end the test
		})
		//目的地址
		dest := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/p2p/%s", port1, h1.ID().Pretty())

		h2, err := makeHost(port2, rand.Reader)
		if err != nil {
			log.Println(err)
			return
		}

		go func() {
			//开启客户端peer，并连接server
			rw, err := startPeerAndConnect(ctx, h2, dest)
			if err != nil {
				log.Println(err)
				return
			}

			rw.WriteString("test message")
			rw.Flush()
		}()

		<-ctx.Done()
	})
}
