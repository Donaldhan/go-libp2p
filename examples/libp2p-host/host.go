package main

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/p2p/net/connmgr"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
)

func main() {
	run()
}

func run() {
	// The context governs the lifetime of the libp2p node.
	// Cancelling it will stop the the host.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// To construct a simple host with all the default settings, just use `New`
	// 构架一个简单的libp2p host
	h, err := libp2p.New()
	if err != nil {
		panic(err)
	}
	//延迟关闭server
	defer h.Close()

	log.Printf("Hello World, my hosts ID is %s\n", h.ID())

	// Now, normally you do not just want a simple host, you want
	// that is fully configured to best support your p2p application.
	// Let's create a second host setting some more options.

	// Set your own keypair 生成公私钥对
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		panic(err)
	}

	var idht *dht.IpfsDHT

	connmgr, err := connmgr.NewConnManager(
		100, // Lowwater
		400, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		panic(err)
	}
	h2, err := libp2p.New(
		// Use the keypair we generated 使用公私钥身份
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",      // regular tcp connections
			"/ip4/0.0.0.0/udp/9000/quic", // a UDP endpoint for the QUIC transport
		),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support noise connections
		libp2p.Security(noise.ID, noise.New),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr),
		// Attempt to open ports using uPNP for NATed hosts. 尝试使用打开uPNP打开 NAT host端口
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts 使用DHT发现其他peer
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h)
			return idht, err
		}),
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		// 启动延迟广播自己
		libp2p.EnableAutoRelay(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		// 开启NAT服务
		libp2p.EnableNATService(),
	)
	if err != nil {
		panic(err)
	}
	//延迟关闭Client
	defer h2.Close()

	// The last step to get fully up and running would be to connect to
	// bootstrap peers (or any other peers). We leave this commented as
	// this is an example and the peer will die as soon as it finishes, so
	// it is unnecessary to put strain on the network.

	/*
		// This connects to public bootstrappers
		for _, addr := range dht.DefaultBootstrapPeers {
			pi, _ := peer.AddrInfoFromP2pAddr(addr)
			// We ignore errors as some bootstrap peers may be down
			// and that is fine.
			h2.Connect(ctx, *pi)
		}
	*/
	log.Printf("Hello World, my second hosts ID is %s\n", h2.ID())
}
