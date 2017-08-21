package main

import (
	"fmt"
	"github.com/m-asama/ftl"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"os"
)

func usage() {
	fmt.Println("usage:")
	fmt.Println("  ftlctl hint stats")
	fmt.Println("  ftlctl hint log")
	fmt.Println("Ex) $ ftlctl 203.0.113.11 stats")
}

func getNodeIds(hint string) []uint32 {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", hint, ftl.GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return nil
	}
	defer conn.Close()

	client := ftl.NewFTLClient(conn)

	void := &ftl.PullNodeIdListRequest{}
	nodeIdList, err := client.PullNodeIdList(context.Background(), void)
	if err != nil {
		fmt.Println("PullNodeList failed.")
		return nil
	}

	nids := make([]uint32, len(nodeIdList.NodeIds))
	for i, nid := range nodeIdList.NodeIds {
		nids[i] = nid
	}

	return nids
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("args required")
		fmt.Println(os.Args)
		usage()
		return
	}

	switch os.Args[2] {
	case "stats":
		doStats()
	case "log":
		doLog()
	default:
		usage()
	}
	return
}
