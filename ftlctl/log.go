package main

import (
	"fmt"
	"github.com/m-asama/ftl"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"os"
	"sort"
	"time"
)

func getAccessLog(node net.IP, date string) []string {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", node, ftl.GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return nil
	}
	defer conn.Close()

	client := ftl.NewFTLClient(conn)

	req := &ftl.GetAccessLogRequest{}
	req.Date = date
	rep, err := client.GetAccessLog(context.Background(), req)
	if err != nil {
		fmt.Println("client.GetAccessLog failed")
		return nil
	}
	return rep.Lines
}

func doLog() {
	nodeIds := getNodeIds(os.Args[1])
	t := time.Now().UTC()
	date := fmt.Sprintf("%4d%02d%02d", t.Year(), int(t.Month()), t.Day())
	if len(os.Args) > 3 {
		date = os.Args[3]
	}
	logs := make([]string, 0)
	for _, nodeId := range nodeIds {
		lines := getAccessLog(ftl.NodeId2Addr4(ftl.NodeId(nodeId)), date)
		logs = append(logs, lines...)
	}
	sort.Strings(logs)
	for _, line := range logs {
		fmt.Println(line)
	}
}
