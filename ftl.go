package ftl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"math"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type NodeId uint32
type NetId uint32

type Node struct {
	addr4    net.IP
	addr6    net.IP
	statsIn  map[NodeId]*NodeStats
	statsOut *NodeStatsCollect
}

type Net struct {
	statsIn  map[NodeId]*NetStats
	statsOut *NetStatsCollect
}

var (
	myNodeId  NodeId
	myAddr4   net.IP
	myAddr6   net.IP
	dataDir   string
	logDir    string
	nodes     map[NodeId]*Node
	nets      map[NetId]*Net
	nodeTree  map[NodeId][]NodeId
	forwarded map[int64]NetId
	mapLock   sync.RWMutex
)

func newNode(nid NodeId) *Node {
	node := &Node{}
	node.addr4 = NodeId2Addr4(nid)
	node.statsIn = make(map[NodeId]*NodeStats)
	node.statsOut = newNodeStatsCollect()
	return node
}

func addNode(nid NodeId) {
	mapLock.Lock()
	defer mapLock.Unlock()

	n := newNode(nid)
	nodes[nid] = n
	rebuildNodeTree()
}

func removeNode(nid NodeId) {
	mapLock.Lock()
	defer mapLock.Unlock()

	if _, ok := nodes[nid]; ok {
		delete(nodes, nid)
	}
	rebuildNodeTree()
}

func newNet() *Net {
	net := &Net{}
	net.statsIn = make(map[NodeId]*NetStats)
	net.statsOut = newNetStatsCollect()
	return net
}

func addNet(nid NetId) {
	mapLock.Lock()
	defer mapLock.Unlock()

	n := newNet()
	nets[nid] = n
}

func removeNet(nid NetId) {
	mapLock.Lock()
	defer mapLock.Unlock()

	if _, ok := nets[nid]; ok {
		delete(nets, nid)
	}
}

func setMyNodeId(nodeIdStr *string) error {
	if nodeIdStr != nil {
		tmp := net.ParseIP(*nodeIdStr)
		if tmp != nil {
			tmp = tmp.To4()
			if tmp != nil {
				myNodeId = NodeId(binary.BigEndian.Uint32(tmp))
			}
		}
	}
	return nil
}

func setMyAddr4() error {
	myAddr4 = NodeId2Addr4(myNodeId)
	return nil
}

func setMyAddr6(addr6str *string) error {
	if addr6str == nil || *addr6str == "" {
		return nil
	}
	addr6tmp := net.ParseIP(*addr6str)
	if addr6tmp == nil {
		return errors.New("setMyAddr6: net.ParseIP error.")
	}
	if addr6tmp.To4() != nil {
		return errors.New("setMyAddr6: not IPv4 address?")
	}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return errors.New("setMyAddr6: net.InterfaceAddrs() failed.")
	}
	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if !ip.IsGlobalUnicast() {
			continue
		}
		if ip.To4() != nil {
			continue
		}
		if ip.Equal(addr6tmp) {
			myAddr6 = ip
			break
		}
	}
	if myAddr6 == nil {
		return errors.New("setMyAddr6: ipv6Addr invalid.")
	}
	return nil
}

func setDataDir(dataDirTmp *string) error {
	if dataDirTmp == nil || *dataDirTmp == "" {
		return errors.New("setDataDir: dataDir is nil.")
	}
	di, err := os.Stat(*dataDirTmp)
	if err != nil {
		return errors.New("setDataDir: get stat failed.")
	}
	if !di.Mode().IsDir() {
		return errors.New("setDataDir: dataDir is not directory.")
	}
	dataDir = *dataDirTmp
	return nil
}

func setLogDir(logDirTmp *string) error {
	if logDirTmp == nil || *logDirTmp == "" {
		return errors.New("setLogDir: logDir is nil.")
	}
	di, err := os.Stat(*logDirTmp)
	if err != nil {
		return errors.New("setLogDir: get stat failed.")
	}
	if !di.Mode().IsDir() {
		return errors.New("setLogDir: dataDir is not directory.")
	}
	logDir = *logDirTmp
	return nil
}

func Init(hints, myNodeId, myAddr6, dataDir, logDir *string) error {
	if err := setMyNodeId(myNodeId); err != nil {
		return err
	}
	if err := setMyAddr4(); err != nil {
		return err
	}
	if err := setMyAddr6(myAddr6); err != nil {
		return err
	}
	if err := setDataDir(dataDir); err != nil {
		return err
	}
	if err := setLogDir(logDir); err != nil {
		return err
	}

	nodes = make(map[NodeId]*Node)
	nets = make(map[NetId]*Net)
	nodeTree = make(map[NodeId][]NodeId)
	forwarded = make(map[int64]NetId)

	rand.Seed(time.Now().Unix())

	return nil
}

func makeNodeList() (*PushNodeListRequest, uint64) {
	mapLock.RLock()
	defer mapLock.RUnlock()
	nodeArray := make([]*NodeEntry, len(nodes))
	var i uint64 = 0
	for nodeId, node := range nodes {
		nodeEntry := &NodeEntry{}
		nodeEntry.NodeId = uint32(nodeId)
		nodeEntry.fillNodeEntryFromNodeStatsCollect(node.statsOut)
		nodeArray[i] = nodeEntry
		i++
	}
	nodeList := &PushNodeListRequest{}
	nodeList.MyNodeId = uint32(myNodeId)
	nodeList.Nodes = nodeArray
	dummy := NodeEntry{}
	size := uint64(uint64(unsafe.Sizeof(dummy))*i + uint64(unsafe.Sizeof(*nodeList)))
	padlen := 1
	if size < 8192 {
		padlen = 2048 - (int(size) / 4)
	}
	nodeList.Padding = make([]uint32, padlen)
	return nodeList, size
}

type GraphNode struct {
	nodeId NodeId
	node   *Node
	cost   uint64
	prev   *GraphNode
	nexts  []*GraphNode
}

func newGraphNode() *GraphNode {
	graphNode := &GraphNode{}
	graphNode.nexts = make([]*GraphNode, 0, len(nodes))
	return graphNode
}

func nodeCost(n1, n2 *GraphNode) uint64 {
	if n2 == nil {
		return math.MaxUint32
	}
	var stats *NodeStats
	if n1 == nil {
		stats = newNodeStats()
		stats.fillNodeStatsFromNodeStatsCollect(n2.node.statsOut)
	} else {
		var ok bool
		stats, ok = n1.node.statsIn[n2.nodeId]
		if !ok {
			return math.MaxUint32
		}
	}
	if stats.hourCount == 0 {
		return math.MaxUint32
	}
	return stats.hourLatency / uint64(stats.hourCount)
}

func rebuildNodeTree() {
	allNodes := make(map[NodeId]*GraphNode)
	remainingNodes := make(map[NodeId]*GraphNode)
	for nodeId, node := range nodes {
		gn := newGraphNode()
		gn.nodeId = nodeId
		gn.node = node
		gn.cost = nodeCost(nil, gn)
		allNodes[nodeId] = gn
		remainingNodes[nodeId] = gn
	}

	for len(remainingNodes) > 0 {
		var gnMinimumCost *GraphNode
		for _, gnTmp := range remainingNodes {
			if gnMinimumCost == nil || gnTmp.cost < gnMinimumCost.cost {
				gnMinimumCost = gnTmp
			}
		}
		delete(remainingNodes, gnMinimumCost.nodeId)
		for _, gnTmp := range remainingNodes {
			if gnTmp.cost > gnMinimumCost.cost+nodeCost(gnMinimumCost, gnTmp) {
				gnTmp.cost = gnMinimumCost.cost + nodeCost(gnMinimumCost, gnTmp)
				gnTmp.prev = gnMinimumCost
			}
		}
	}

	newNodeTree := make(map[NodeId][]NodeId)
	newNodeTree[myNodeId] = make([]NodeId, 0)
	for _, gn := range allNodes {
		if gn.prev == nil {
			newNodeTree[myNodeId] = append(newNodeTree[myNodeId], gn.nodeId)
			continue
		}
		prev := gn.prev
		prev.nexts = append(prev.nexts, gn)
	}
	for _, gn := range allNodes {
		newNodeTree[gn.nodeId] = make([]NodeId, len(gn.nexts))
		for i := 0; i < len(gn.nexts); i++ {
			newNodeTree[gn.nodeId][i] = gn.nexts[i].nodeId
		}
	}

	nodeTree = newNodeTree
}

func nodeReady(node *Node, netId NetId, observation bool) bool {
	addr4 := myAddr4
	if node != nil {
		addr4 = node.addr4
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", addr4, GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return false
	}
	defer conn.Close()

	client := NewFTLClient(conn)

	query := &QueryReadyRequest{uint32(myNodeId), uint32(netId), observation}
	ready, err := client.QueryReady(context.Background(), query)
	if err != nil || !ready.Ready {
		return false
	}

	return true
}

func findRandomNode(netId NetId, observation bool) *Node {
	var node *Node
	retryCount := 0
	nodeArray := make([]*Node, len(nodes)+1)
	i := 1
	for _, n := range nodes {
		nodeArray[i] = n
		i++
	}
retry:
	if retryCount > 3 {
		return nil
	}
	node = nodeArray[rand.Intn(len(nodeArray))]
	if !nodeReady(node, netId, true) {
		retryCount++
		goto retry
	}
	return node
}

func findUnknownNode(netId NetId) *Node {
	net, ok := nets[netId]
	if !ok {
		return findRandomNode(netId, true)
	}
	var bestNode *Node
	bestNodeStats := &NetStats{}
	bestNodeStats.fillNetStatsFromNetStatsCollect(net.statsOut)
	retryCount := 0
retry:
	if retryCount > 3 {
		return nil
	}
	for nodeId, stats := range net.statsIn {
		tmp, ok := nodes[nodeId]
		if !ok {
			continue
		}
		if stats.hourCount < bestNodeStats.hourCount {
			bestNode = tmp
			bestNodeStats = stats
		}
	}
	if !nodeReady(bestNode, netId, true) {
		retryCount++
		goto retry
	}
	return bestNode
}

func findBestNode(netId NetId) *Node {
	net, ok := nets[netId]
	if !ok {
		return findRandomNode(netId, false)
	}
	var bestNode *Node
	bestNodeStats := &NetStats{}
	bestNodeStats.fillNetStatsFromNetStatsCollect(net.statsOut)
	retryCount := 0
retry:
	if retryCount > 3 {
		return nil
	}
	for nodeId, stats := range net.statsIn {
		tmp, ok := nodes[nodeId]
		if !ok {
			continue
		}
		if stats.isBetterThan(bestNodeStats) {
			bestNode = tmp
			bestNodeStats = stats
		}
	}
	if !nodeReady(bestNode, netId, false) {
		retryCount++
		goto retry
	}
	return bestNode
}

func HelloLongTimeNoSee() {
	var n *Node
	var nid NodeId
	for k, v := range nodes {
		if n == nil || v.statsOut.lastUpdate.Before(n.statsOut.lastUpdate) {
			n = v
			nid = k
		}
	}
	if n == nil {
		logError("HelloLongTimeNoSee: no nodes")
		return
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", n.addr4, GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return
	}

	var before, after int64

	client := NewFTLClient(conn)

	request := &HelloRequest{uint32(myNodeId), time.Now().Unix()}
	before = time.Now().UnixNano()
	reply, err := client.Hello(context.Background(), request)
	after = time.Now().UnixNano()
	if err != nil {
		removeNode(nid)
		logError("HelloLongTimeNoSee: Hello failed.", fmt.Sprint(reply))
		conn.Close()
		return
	}
	latency := uint64(after - before)
	n.addr6 = net.ParseIP(reply.Addr6)

	nodeList, uploadBytes := makeNodeList()
	before = time.Now().UnixNano()
	result, err := client.PushNodeList(context.Background(), nodeList)
	conn.Close()
	after = time.Now().UnixNano()
	if err != nil {
		logError("HelloLongTimeNoSee: Push Node List failed.", fmt.Sprint(result))
		return
	}
	uploadNanoSec := uint64(after - before)

	now := time.Now()
	n.statsOut.updateStats(now, 1, uploadNanoSec, uploadBytes, latency)
	//n.lastUpdate = now
}

func sendNetStatsToNexthop(nexthop NodeId, request *BroadcastNetListRequest) {
	//logError("sendNetStatsToNexthop: begin", request.MyNodeId, uint32(nexthop))
	//defer logError("sendNetStatsToNexthop: end", request.MyNodeId, uint32(nexthop))

	n, ok := nodes[nexthop]
	if !ok {
		return
	}

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", n.addr4, GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := NewFTLClient(conn)

	reply, err := client.BroadcastNetList(context.Background(), request)
	_ = reply
	_ = err
}

func BroadcastNetStats() {
	//logError("BroadcastNetStats: begin")
	//defer logError("BroadcastNetStats: end")

	rebuildNodeTree()

	nexthopIds, ok := nodeTree[myNodeId]
	if !ok {
		logError("BroadcastNetStats: ok != nodeTree[myNodeId]")
		return
	}

	var i int

	destinations := make([]*Destination, len(nodeTree))
	i = 0
	for dst, nexts := range nodeTree {
		d := &Destination{}
		d.NodeId = uint32(dst)
		d.NexthopIds = make([]uint32, len(nexts))
		for j := 0; j < len(nexts); j++ {
			d.NexthopIds[j] = uint32(nexts[j])
		}
		destinations[i] = d
		i++
	}

	netEntries := make([]*NetEntry, len(nets))
	i = 0
	for netId, net := range nets {
		n := &NetEntry{}
		n.NetId = uint32(netId)
		n.fillNetEntryFromNetStatsCollect(net.statsOut)
		netEntries[i] = n
		i++
	}

	request := &BroadcastNetListRequest{}
	request.MyNodeId = uint32(myNodeId)
	request.Destinations = destinations
	request.Nets = netEntries

	for _, nexthopId := range nexthopIds {
		//logError("BroadcastNetStats:", nexthopId)
		_, ok := nodes[nexthopId]
		if !ok {
			//logError("BroadcastNetStats: continue")
			continue
		}
		sendNetStatsToNexthop(nexthopId, request)
	}
}

func connect(hint string) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", hint, GRPC_PORT), opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := NewFTLClient(conn)

	request := &HelloRequest{uint32(myNodeId), time.Now().Unix()}
	_, err = client.Hello(context.Background(), request)
	if err != nil {
		logError("connect: Hello failed.")
		return
	}

	void := &PullNodeIdListRequest{}
	nodeIdList, err := client.PullNodeIdList(context.Background(), void)
	if err != nil {
		logError("connect: PullNodeList failed.")
		return
	}

	for _, nid := range nodeIdList.NodeIds {
		if NodeId(nid) == myNodeId {
			continue
		}
		if _, ok := nodes[NodeId(nid)]; !ok {
			addNode(NodeId(nid))
		}
	}
}

func Connect(hints *string) {
	if hints == nil || *hints == "" {
		return
	}
	hs := strings.Split(*hints, ",")
	for _, hint := range hs {
		connect(hint)
	}
}

func GCService() {
	for {
		now := time.Now().UnixNano()
		dels := make(map[int64]NetId)
		mapLock.Lock()
		for k, v := range forwarded {
			if k < now-1000000000*60 {
				dels[k] = v
			}
		}
		for k, _ := range dels {
			delete(forwarded, k)
		}
		mapLock.Unlock()
		time.Sleep(60 * time.Second)
	}
}

func Dump() {
	mapLock.RLock()
	defer mapLock.RUnlock()
	logError("")
	logError("myNodeId:  ", fmt.Sprint(myNodeId))
	logError("myAddr4:   ", fmt.Sprint(myAddr4))
	logError("myAddr6:   ", fmt.Sprint(myAddr6))
	logError("dataDir: ", dataDir)
	logError("nodes:")
	for k, v := range nodes {
		logError("   ", fmt.Sprint(k))
		logError("        addr4:               ", fmt.Sprint(v.addr4))
		logError("        addr6:               ", fmt.Sprint(v.addr6))
		logError("        statsIn:")
		for k2, v2 := range v.statsIn {
			logError("           ", fmt.Sprint(k2))
			logError("                lastUpdate: ", fmt.Sprint(v2.lastUpdate))
			logError(fmt.Sprintf("                Count:         %15d %15d %15d",
				v2.hourCount, v2.dayCount, v2.weekCount))
			logError(fmt.Sprintf("                UploadNanoSec: %15d %15d %15d",
				v2.hourUploadNanoSec, v2.dayUploadNanoSec, v2.weekUploadNanoSec))
			logError(fmt.Sprintf("                UploadBytes:   %15d %15d %15d",
				v2.hourUploadBytes, v2.dayUploadBytes, v2.weekUploadBytes))
			logError(fmt.Sprintf("                Latency:       %15d %15d %15d",
				v2.hourLatency, v2.dayLatency, v2.weekLatency))
		}
		logError("        statsOut:")
		logError("            lastUpdate: ", fmt.Sprint(v.statsOut.lastUpdate))
		logError(fmt.Sprintf("            Count:         %15d %15d %15d",
			v.statsOut.getHourCount(),
			v.statsOut.getDayCount(),
			v.statsOut.getWeekCount()))
		logError(fmt.Sprintf("            UploadNanoSec: %15d %15d %15d",
			v.statsOut.getHourUploadNanoSec(),
			v.statsOut.getDayUploadNanoSec(),
			v.statsOut.getWeekUploadNanoSec()))
		logError(fmt.Sprintf("            UploadBytes:   %15d %15d %15d",
			v.statsOut.getHourUploadBytes(),
			v.statsOut.getDayUploadBytes(),
			v.statsOut.getWeekUploadBytes()))
		logError(fmt.Sprintf("            Latency:       %15d %15d %15d",
			v.statsOut.getHourLatency(),
			v.statsOut.getDayLatency(),
			v.statsOut.getWeekLatency()))
	}
	logError("nets:")
	for k1, v1 := range nets {
		logError("   ", fmt.Sprint(k1))
		logError("        statsIn:")
		for k2, v2 := range v1.statsIn {
			logError("           ", fmt.Sprint(k2))
			logError("                lastUpdate: ", fmt.Sprint(v2.lastUpdate))
			logError(fmt.Sprintf("                Count:           %15d %15d %15d",
				v2.hourCount, v2.dayCount, v2.weekCount))
			logError(fmt.Sprintf("                DownloadNanoSec: %15d %15d %15d",
				v2.hourDownloadNanoSec, v2.dayDownloadNanoSec, v2.weekDownloadNanoSec))
			logError(fmt.Sprintf("                DownloadBytes:   %15d %15d %15d",
				v2.hourDownloadBytes, v2.dayDownloadBytes, v2.weekDownloadBytes))
		}
		logError("        statsOut:")
		logError("            lastUpdate: ", fmt.Sprint(v1.statsOut.lastUpdate))
		logError(fmt.Sprintf("            Count:           %15d %15d %15d",
			v1.statsOut.getHourCount(),
			v1.statsOut.getDayCount(),
			v1.statsOut.getWeekCount()))
		logError(fmt.Sprintf("            DownloadNanoSec: %15d %15d %15d",
			v1.statsOut.getHourDownloadNanoSec(),
			v1.statsOut.getDayDownloadNanoSec(),
			v1.statsOut.getWeekDownloadNanoSec()))
		logError(fmt.Sprintf("            DownloadBytes:   %15d %15d %15d",
			v1.statsOut.getHourDownloadBytes(),
			v1.statsOut.getDayDownloadBytes(),
			v1.statsOut.getWeekDownloadBytes()))
	}
	logError("nodeTree:")
	for k1, v1 := range nodeTree {
		logError("   ", fmt.Sprint(k1))
		for _, v2 := range v1 {
			logError("       ", fmt.Sprint(v2))
		}
	}
	logError("forwarded:")
	for k1, v1 := range forwarded {
		logError("   ", fmt.Sprint(k1, ":", v1))
	}
}
