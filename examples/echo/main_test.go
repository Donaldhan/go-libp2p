package main

import (
	"context"
	"log"
	"testing"

	"github.com/libp2p/go-libp2p/examples/testutils"
)

func TestMain(t *testing.T) {
	var h testutils.LogHarness
	h.Expect("listening for connections")
	h.Expect("sender opening stream")
	h.Expect("sender saying hello")
	h.Expect("listener received new stream")
	h.Expect("read: Hello, world!")
	h.Expect(`read reply: "Hello, world!\n"`)

	h.Run(t, func() {
		// Create a context that will stop the hosts when the tests end
		ctx, cancel := context.WithCancel(context.Background())
		// 退出时，cancel ，stop host
		defer cancel()

		// Get a tcp port for the listener
		lport, err := testutils.FindFreePort(t, "", 5)
		if err != nil {
			log.Println(err)
			return
		}

		// Get a tcp port for the sender
		sport, err := testutils.FindFreePort(t, "", 5)
		if err != nil {
			log.Println(err)
			return
		}

		// Make listener
		lh, err := makeBasicHost(lport, true, 1)
		if err != nil {
			log.Println(err)
			return
		}
		//监听
		startListener(ctx, lh, lport, true)

		// Make sender 获取监听服务端监听地址
		listenAddr := getHostAddress(lh)
		sh, err := makeBasicHost(sport, true, 2)
		if err != nil {
			log.Println(err)
			return
		}
		//client dial拨号
		runSender(ctx, sh, listenAddr)
	})
}
