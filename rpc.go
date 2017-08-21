package ftl

import (
	"bufio"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"net"
	"os"
	"time"
)

const (
	GRPC_PORT = 9000
)

type FTLRPCServer struct {
}

func (s *FTLRPCServer) Hello(ctx context.Context, request *HelloRequest) (*HelloReply, error) {
	//logError("HelloServer: request = ", *request)
	_, ok := nodes[NodeId(request.MyNodeId)]
	if !ok {
		nid := request.MyNodeId
		addNode(NodeId(nid))
	}
	myAddr6Str := ""
	if myAddr6 != nil {
		myAddr6Str = myAddr6.String()
	}
	reply := &HelloReply{uint32(myNodeId), time.Now().Unix(), myAddr6Str}
	//logError("HelloServer: reply = ", *reply)
	return reply, nil
}

func (s *FTLRPCServer) PullNodeIdList(ctx context.Context, void *PullNodeIdListRequest) (*PullNodeIdListReply, error) {
	nodeIdList := &PullNodeIdListReply{uint32(myNodeId), nil}
	nodeIds := make([]uint32, len(nodes)+1)
	nodeIds[0] = uint32(myNodeId)
	var i int = 1
	for nid, _ := range nodes {
		nodeIds[i] = uint32(nid)
		i++
	}
	nodeIdList.NodeIds = nodeIds
	return nodeIdList, nil
}

func (s *FTLRPCServer) PushNodeList(ctx context.Context, request *PushNodeListRequest) (*PushNodeListReply, error) {
	remoteNodeId := NodeId(request.MyNodeId)
	remoteNode, ok := nodes[remoteNodeId]
	if !ok {
		addNode(remoteNodeId)
		remoteNode = nodes[remoteNodeId]
	}
	delNodeList := make(map[NodeId]bool)
	for nodeIdTmp, _ := range nodes {
		delNodeList[nodeIdTmp] = true
	}
	for _, adjNode := range request.Nodes {
		adjNodeId := NodeId(adjNode.NodeId)
		adjNodeStats, ok := remoteNode.statsIn[adjNodeId]
		if !ok {
			adjNodeStats = newNodeStats()
			remoteNode.statsIn[adjNodeId] = adjNodeStats
		}
		adjNodeStats.fillNodeStatsFromNodeEntry(adjNode)
		adjNodeStats.lastUpdate = time.Now()
		if _, ok := delNodeList[adjNodeId]; ok {
			delete(delNodeList, adjNodeId)
		}
	}
	for delNodeId, _ := range delNodeList {
		delete(remoteNode.statsIn, delNodeId)
	}
	reply := &PushNodeListReply{uint32(myNodeId), 0}
	//time.Sleep(1 * time.Second)
	return reply, nil
}

func (s *FTLRPCServer) BroadcastNetList(ctx context.Context, request *BroadcastNetListRequest) (*BroadcastNetListReply, error) {
	//logError("BroadcastNetList: begin")
	//defer logError("BroadcastNetList: end")

	result := &BroadcastNetListReply{uint32(myNodeId), 0}

	remoteNodeId := NodeId(request.MyNodeId)
	_, ok := nodes[remoteNodeId]
	if !ok {
		logError("BroadcastNetList: ok != nodes[remoteNodeId]")
		return result, nil
	}

	for _, dst := range request.Destinations {
		if NodeId(dst.NodeId) == myNodeId {
			for _, nh := range dst.NexthopIds {
				sendNetStatsToNexthop(NodeId(nh), request)
			}
		}
	}

	for _, ne := range request.Nets {
		//logError("BroadcastNetList: range request.Nets", string(iii))
		netId := NetId(ne.NetId)
		net, ok := nets[netId]
		if !ok {
			net = newNet()
			nets[netId] = net
		}
		netStats, ok := net.statsIn[remoteNodeId]
		if !ok {
			netStats = newNetStats()
			net.statsIn[remoteNodeId] = netStats
		}
		netStats.fillNetStatsFromNetEntry(ne)
		netStats.lastUpdate = time.Now()
	}

	return result, nil
}

func (s *FTLRPCServer) QueryReady(ctx context.Context, req *QueryReadyRequest) (*QueryReadyReply, error) {
	ready := &QueryReadyReply{uint32(myNodeId), true}
	if req.Observation {
		forwarded[time.Now().UnixNano()] = NetId(req.NetId)
	}
	return ready, nil
}

func (s *FTLRPCServer) GetStats(ctx context.Context, req *GetStatsRequest) (*GetStatsReply, error) {
	reply := &GetStatsReply{}
	nodesMap := make(map[NodeId]*NodeContainer)
	n := &NodeContainer{}
	n.NodeId = uint32(myNodeId)
	n.RemoteNodes = make([]*NodeEntry, 0)
	n.ClientNets = make([]*NetEntry, 0)
	nodesMap[myNodeId] = n
	for nodeId, _ := range nodes {
		n := &NodeContainer{}
		n.NodeId = uint32(nodeId)
		n.RemoteNodes = make([]*NodeEntry, 0)
		n.ClientNets = make([]*NetEntry, 0)
		nodesMap[nodeId] = n
	}
	for nodeId, node := range nodes {
		if n, ok := nodesMap[myNodeId]; ok {
			e := &NodeEntry{}
			e.NodeId = uint32(nodeId)
			e.fillNodeEntryFromNodeStatsCollect(node.statsOut)
			n.RemoteNodes = append(n.RemoteNodes, e)
		}
		for nodeIdTmp, statsTmp := range node.statsIn {
			if n, ok := nodesMap[nodeIdTmp]; ok {
				e := &NodeEntry{}
				e.NodeId = uint32(nodeId)
				e.fillNodeEntryFromNodeStats(statsTmp)
				n.RemoteNodes = append(n.RemoteNodes, e)
			}
		}
	}
	for netId, net := range nets {
		if n, ok := nodesMap[myNodeId]; ok {
			e := &NetEntry{}
			e.NetId = uint32(netId)
			e.fillNetEntryFromNetStatsCollect(net.statsOut)
			n.ClientNets = append(n.ClientNets, e)
		}
		for nodeIdTmp, statsTmp := range net.statsIn {
			if n, ok := nodesMap[nodeIdTmp]; ok {
				e := &NetEntry{}
				e.NetId = uint32(netId)
				e.fillNetEntryFromNetStats(statsTmp)
				n.ClientNets = append(n.ClientNets, e)
			}
		}
	}
	nodeContainers := make([]*NodeContainer, len(nodesMap))
	i := 0
	for _, nodeContainer := range nodesMap {
		nodeContainers[i] = nodeContainer
		i++
	}
	reply.Nodes = nodeContainers
	return reply, nil
}

func (s *FTLRPCServer) GetAccessLog(ctx context.Context, req *GetAccessLogRequest) (*GetAccessLogReply, error) {
	logError("GetAccessLog: begin")
	defer logError("GetAccessLog: end")
	rep := &GetAccessLogReply{}
	accessLogPath := fmt.Sprintf("%s/access-%s.log", logDir, req.Date)
	logError("GetAccessLog: accessLogPath =", accessLogPath)
	accessLogFp, err := os.Open(accessLogPath)
	if err != nil {
		logError("GetAccessLog: os.Open failed", fmt.Sprint(err))
		return rep, nil
	}
	accessLogReader := bufio.NewReaderSize(accessLogFp, 4096)
	lines := make([]string, 0)
	for {
		line, _, err := accessLogReader.ReadLine()
		if err != nil {
			break
		}
		lines = append(lines, string(line))
	}
	rep.Lines = lines
	return rep, nil
}

func RPCService() {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", GRPC_PORT))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	RegisterFTLServer(grpcServer, &FTLRPCServer{})
	grpcServer.Serve(conn)
}
