package main

import (
	"flag"
	"fmt"
	"github.com/m-asama/ftl"
	"time"
)

var (
	hints     = flag.String("hints", "", "Hints to connect FTL network (Ex: ftl.example.com)")
	myNodeId  = flag.String("myNodeId", "", "My node ID(IPv4 address) (Ex: 203.0.113.123)")
	myAddr6   = flag.String("myAddr6", "", "My IPv6 address (Ex: 2001:db8:1:2::3)")
	myDataDir = flag.String("myDataDir", "", "Store data directory (Ex: /home/ftl/data)")
	logDir    = flag.String("logDir", "", "Store log directory (Ex: /home/ftl/log)")
)

func helloLongTimeNoSee() {
	for {
		ftl.HelloLongTimeNoSee()
		time.Sleep(10 * time.Second)
	}
}

func broadcastNetStats() {
	for {
		ftl.BroadcastNetStats()
		time.Sleep(20 * time.Second)
	}
}

func main() {
	flag.Parse()

	err := ftl.Init(hints, myNodeId, myAddr6, myDataDir, logDir)
	if err != nil {
		fmt.Println("ftl.Init:", err)
		return
	}

	go ftl.DNSService()
	go ftl.HTTPService()
	go ftl.RPCService()
	go ftl.GCService()

	ftl.Connect(hints)

	go helloLongTimeNoSee()
	go broadcastNetStats()

	for {
		ftl.Dump()
		time.Sleep(60 * time.Second)
	}
}
