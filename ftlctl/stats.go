package main

import (
	"fmt"
	"github.com/m-asama/ftl"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"os"
)

func doStats() {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", os.Args[1], ftl.GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := ftl.NewFTLClient(conn)

	request := &ftl.GetStatsRequest{}
	reply, err := client.GetStats(context.Background(), request)
	if err != nil {
		fmt.Println("GetStats failed.")
		return
	}

	for _, nodeContainer := range reply.Nodes {
		for _, netEntry := range nodeContainer.ClientNets {
			fmt.Printf("%-16s %-16s %14.9f %14.9f %14.9f\n",
				ftl.NodeId2Addr4(ftl.NodeId(nodeContainer.NodeId)),
				ftl.NetId2Addr4(ftl.NetId(netEntry.NetId)),
				float64(netEntry.StatsHourDownloadBytes)/float64(netEntry.StatsHourDownloadNanoSec),
				float64(netEntry.StatsDayDownloadBytes)/float64(netEntry.StatsDayDownloadNanoSec),
				float64(netEntry.StatsWeekDownloadBytes)/float64(netEntry.StatsWeekDownloadNanoSec))
		}
	}

}
