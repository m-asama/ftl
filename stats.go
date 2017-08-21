package ftl

import (
	"time"
)

type NodeStatsCollect struct {
	minuteCount         [60]uint32
	minuteUploadNanoSec [60]uint64
	minuteUploadBytes   [60]uint64
	minuteLatency       [60]uint64
	hourCount           [24]uint32
	hourUploadNanoSec   [24]uint64
	hourUploadBytes     [24]uint64
	hourLatency         [24]uint64
	dayCount            [7]uint32
	dayUploadNanoSec    [7]uint64
	dayUploadBytes      [7]uint64
	dayLatency          [7]uint64
	lastUpdate          time.Time
}

type NodeStats struct {
	hourCount         uint32
	hourUploadNanoSec uint64
	hourUploadBytes   uint64
	hourLatency       uint64
	dayCount          uint32
	dayUploadNanoSec  uint64
	dayUploadBytes    uint64
	dayLatency        uint64
	weekCount         uint32
	weekUploadNanoSec uint64
	weekUploadBytes   uint64
	weekLatency       uint64
	lastUpdate        time.Time
}

type NetStatsCollect struct {
	minuteCount           [60]uint32
	minuteDownloadNanoSec [60]uint64
	minuteDownloadBytes   [60]uint64
	hourCount             [24]uint32
	hourDownloadNanoSec   [24]uint64
	hourDownloadBytes     [24]uint64
	dayCount              [7]uint32
	dayDownloadNanoSec    [7]uint64
	dayDownloadBytes      [7]uint64
	lastUpdate            time.Time
}

type NetStats struct {
	hourCount           uint32
	hourDownloadNanoSec uint64
	hourDownloadBytes   uint64
	dayCount            uint32
	dayDownloadNanoSec  uint64
	dayDownloadBytes    uint64
	weekCount           uint32
	weekDownloadNanoSec uint64
	weekDownloadBytes   uint64
	lastUpdate          time.Time
}

func (s *NodeStatsCollect) getHourCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.minuteCount); i++ {
		tmp += s.minuteCount[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getHourUploadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.minuteUploadNanoSec); i++ {
		tmp += s.minuteUploadNanoSec[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getHourUploadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.minuteUploadBytes); i++ {
		tmp += s.minuteUploadBytes[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getHourLatency() uint64 {
	var tmp uint64
	for i := 0; i < len(s.minuteLatency); i++ {
		tmp += s.minuteLatency[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getDayCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.hourCount); i++ {
		tmp += s.hourCount[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getDayUploadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.hourUploadNanoSec); i++ {
		tmp += s.hourUploadNanoSec[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getDayUploadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.hourUploadBytes); i++ {
		tmp += s.hourUploadBytes[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getDayLatency() uint64 {
	var tmp uint64
	for i := 0; i < len(s.hourLatency); i++ {
		tmp += s.hourLatency[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getWeekCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.dayCount); i++ {
		tmp += s.dayCount[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getWeekUploadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.dayUploadNanoSec); i++ {
		tmp += s.dayUploadNanoSec[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getWeekUploadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.dayUploadBytes); i++ {
		tmp += s.dayUploadBytes[i]
	}
	return tmp
}

func (s *NodeStatsCollect) getWeekLatency() uint64 {
	var tmp uint64
	for i := 0; i < len(s.dayLatency); i++ {
		tmp += s.dayLatency[i]
	}
	return tmp
}

func (s *NetStatsCollect) getHourCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.minuteCount); i++ {
		tmp += s.minuteCount[i]
	}
	return tmp
}

func (s *NetStatsCollect) getHourDownloadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.minuteDownloadNanoSec); i++ {
		tmp += s.minuteDownloadNanoSec[i]
	}
	return tmp
}

func (s *NetStatsCollect) getHourDownloadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.minuteDownloadBytes); i++ {
		tmp += s.minuteDownloadBytes[i]
	}
	return tmp
}

func (s *NetStatsCollect) getDayCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.hourCount); i++ {
		tmp += s.hourCount[i]
	}
	return tmp
}

func (s *NetStatsCollect) getDayDownloadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.hourDownloadNanoSec); i++ {
		tmp += s.hourDownloadNanoSec[i]
	}
	return tmp
}

func (s *NetStatsCollect) getDayDownloadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.hourDownloadBytes); i++ {
		tmp += s.hourDownloadBytes[i]
	}
	return tmp
}

func (s *NetStatsCollect) getWeekCount() uint32 {
	var tmp uint32
	for i := 0; i < len(s.dayCount); i++ {
		tmp += s.dayCount[i]
	}
	return tmp
}

func (s *NetStatsCollect) getWeekDownloadNanoSec() uint64 {
	var tmp uint64
	for i := 0; i < len(s.dayDownloadNanoSec); i++ {
		tmp += s.dayDownloadNanoSec[i]
	}
	return tmp
}

func (s *NetStatsCollect) getWeekDownloadBytes() uint64 {
	var tmp uint64
	for i := 0; i < len(s.dayDownloadBytes); i++ {
		tmp += s.dayDownloadBytes[i]
	}
	return tmp
}

func newNodeStatsCollect() *NodeStatsCollect {
	stats := &NodeStatsCollect{}
	stats.lastUpdate = time.Now()
	return stats
}

func newNetStatsCollect() *NetStatsCollect {
	stats := &NetStatsCollect{}
	stats.lastUpdate = time.Now()
	return stats
}

func newNodeStats() *NodeStats {
	stats := &NodeStats{}
	stats.lastUpdate = time.Now()
	return stats
}

func newNetStats() *NetStats {
	stats := &NetStats{}
	stats.lastUpdate = time.Now()
	return stats
}

func minuteIndex(t time.Time) int {
	return int(t.Unix() % (60 * 60) / (60))
}

func hourIndex(t time.Time) int {
	return int(t.Unix() % (60 * 60 * 24) / (60 * 60))
}

func dayIndex(t time.Time) int {
	return int(t.Unix() % (60 * 60 * 24 * 7) / (60 * 60 * 24))
}

func (sc *NodeStatsCollect) updateStats(t time.Time, count uint32, uploadNanoSec, uploadBytes, latency uint64) {
	//
	mitmp := minuteIndex(sc.lastUpdate)
	if mitmp > minuteIndex(t) {
		mitmp -= 60
	}
	for mitmp < minuteIndex(t) {
		i := (mitmp + 60 + 1) % 60
		sc.minuteCount[i] = 0
		sc.minuteUploadNanoSec[i] = 0
		sc.minuteUploadBytes[i] = 0
		sc.minuteLatency[i] = 0
		mitmp += 1
	}
	mitmp = mitmp % 60
	sc.minuteCount[mitmp] += count
	sc.minuteUploadNanoSec[mitmp] += uploadNanoSec
	sc.minuteUploadBytes[mitmp] += uploadBytes
	sc.minuteLatency[mitmp] += latency
	//
	hitmp := hourIndex(sc.lastUpdate)
	if hitmp > hourIndex(t) {
		hitmp -= 24
	}
	for hitmp < hourIndex(t) {
		i := (hitmp + 24 + 1) % 24
		sc.hourCount[i] = 0
		sc.hourUploadNanoSec[i] = 0
		sc.hourUploadBytes[i] = 0
		sc.hourLatency[i] = 0
		hitmp += 1
	}
	hitmp = hitmp % 24
	sc.hourCount[hitmp] = sc.getHourCount()
	sc.hourUploadNanoSec[hitmp] = sc.getHourUploadNanoSec()
	sc.hourUploadBytes[hitmp] = sc.getHourUploadBytes()
	sc.hourLatency[hitmp] = sc.getHourLatency()
	//
	ditmp := dayIndex(sc.lastUpdate)
	if ditmp > dayIndex(t) {
		ditmp -= 7
	}
	for ditmp < dayIndex(t) {
		i := (ditmp + 7 + 1) % 7
		sc.dayCount[i] = 0
		sc.dayUploadNanoSec[i] = 0
		sc.dayUploadBytes[i] = 0
		sc.dayLatency[i] = 0
		ditmp += 1
	}
	ditmp = ditmp % 7
	sc.dayCount[ditmp] = sc.getDayCount()
	sc.dayUploadNanoSec[ditmp] = sc.getDayUploadNanoSec()
	sc.dayUploadBytes[ditmp] = sc.getDayUploadBytes()
	sc.dayLatency[ditmp] = sc.getDayLatency()
	//
	sc.lastUpdate = t
}

func (sc *NetStatsCollect) updateStats(t time.Time, count uint32, downloadNanoSec, downloadBytes uint64) {
	//
	mitmp := minuteIndex(sc.lastUpdate)
	if mitmp > minuteIndex(t) {
		mitmp -= 60
	}
	for mitmp < minuteIndex(t) {
		i := (mitmp + 60 + 1) % 60
		sc.minuteCount[i] = 0
		sc.minuteDownloadNanoSec[i] = 0
		sc.minuteDownloadBytes[i] = 0
		mitmp += 1
	}
	mitmp = mitmp % 60
	sc.minuteCount[mitmp] += count
	sc.minuteDownloadNanoSec[mitmp] += downloadNanoSec
	sc.minuteDownloadBytes[mitmp] += downloadBytes
	//
	hitmp := hourIndex(sc.lastUpdate)
	if hitmp > hourIndex(t) {
		hitmp -= 24
	}
	for hitmp < hourIndex(t) {
		i := (hitmp + 24 + 1) % 24
		sc.hourCount[i] = 0
		sc.hourDownloadNanoSec[i] = 0
		sc.hourDownloadBytes[i] = 0
		hitmp += 1
	}
	hitmp = hitmp % 24
	sc.hourCount[hitmp] = sc.getHourCount()
	sc.hourDownloadNanoSec[hitmp] = sc.getHourDownloadNanoSec()
	sc.hourDownloadBytes[hitmp] = sc.getHourDownloadBytes()
	//
	ditmp := dayIndex(sc.lastUpdate)
	if ditmp > dayIndex(t) {
		ditmp -= 7
	}
	for ditmp < dayIndex(t) {
		i := (ditmp + 7 + 1) % 7
		sc.dayCount[i] = 0
		sc.dayDownloadNanoSec[i] = 0
		sc.dayDownloadBytes[i] = 0
		ditmp += 1
	}
	ditmp = ditmp % 7
	sc.dayCount[ditmp] = sc.getDayCount()
	sc.dayDownloadNanoSec[ditmp] = sc.getDayDownloadNanoSec()
	sc.dayDownloadBytes[ditmp] = sc.getDayDownloadBytes()
	//
	sc.lastUpdate = t
}

func (nodeEntry *NodeEntry) fillNodeEntryFromNodeStatsCollect(stats *NodeStatsCollect) {
	nodeEntry.StatsHourCount = stats.getHourCount()
	nodeEntry.StatsHourUploadNanoSec = stats.getHourUploadNanoSec()
	nodeEntry.StatsHourUploadBytes = stats.getHourUploadBytes()
	nodeEntry.StatsHourLatency = stats.getHourLatency()
	nodeEntry.StatsDayCount = stats.getDayCount()
	nodeEntry.StatsDayUploadNanoSec = stats.getDayUploadNanoSec()
	nodeEntry.StatsDayUploadBytes = stats.getDayUploadBytes()
	nodeEntry.StatsDayLatency = stats.getDayLatency()
	nodeEntry.StatsWeekCount = stats.getWeekCount()
	nodeEntry.StatsWeekUploadNanoSec = stats.getWeekUploadNanoSec()
	nodeEntry.StatsWeekUploadBytes = stats.getWeekUploadBytes()
	nodeEntry.StatsWeekLatency = stats.getWeekLatency()
}

func (netEntry *NetEntry) fillNetEntryFromNetStatsCollect(stats *NetStatsCollect) {
	netEntry.StatsHourCount = stats.getHourCount()
	netEntry.StatsHourDownloadNanoSec = stats.getHourDownloadNanoSec()
	netEntry.StatsHourDownloadBytes = stats.getHourDownloadBytes()
	netEntry.StatsDayCount = stats.getDayCount()
	netEntry.StatsDayDownloadNanoSec = stats.getDayDownloadNanoSec()
	netEntry.StatsDayDownloadBytes = stats.getDayDownloadBytes()
	netEntry.StatsWeekCount = stats.getWeekCount()
	netEntry.StatsWeekDownloadNanoSec = stats.getWeekDownloadNanoSec()
	netEntry.StatsWeekDownloadBytes = stats.getWeekDownloadBytes()
}

func (nodeStats *NodeStats) fillNodeStatsFromNodeStatsCollect(stats *NodeStatsCollect) {
	nodeStats.hourCount = stats.getHourCount()
	nodeStats.hourUploadNanoSec = stats.getHourUploadNanoSec()
	nodeStats.hourUploadBytes = stats.getHourUploadBytes()
	nodeStats.hourLatency = stats.getHourLatency()
	nodeStats.dayCount = stats.getDayCount()
	nodeStats.dayUploadNanoSec = stats.getDayUploadNanoSec()
	nodeStats.dayUploadBytes = stats.getDayUploadBytes()
	nodeStats.dayLatency = stats.getDayLatency()
	nodeStats.weekCount = stats.getWeekCount()
	nodeStats.weekUploadNanoSec = stats.getWeekUploadNanoSec()
	nodeStats.weekUploadBytes = stats.getWeekUploadBytes()
	nodeStats.weekLatency = stats.getWeekLatency()
}

func (netStats *NetStats) fillNetStatsFromNetStatsCollect(stats *NetStatsCollect) {
	netStats.hourCount = stats.getHourCount()
	netStats.hourDownloadNanoSec = stats.getHourDownloadNanoSec()
	netStats.hourDownloadBytes = stats.getHourDownloadBytes()
	netStats.dayCount = stats.getDayCount()
	netStats.dayDownloadNanoSec = stats.getDayDownloadNanoSec()
	netStats.dayDownloadBytes = stats.getDayDownloadBytes()
	netStats.weekCount = stats.getWeekCount()
	netStats.weekDownloadNanoSec = stats.getWeekDownloadNanoSec()
	netStats.weekDownloadBytes = stats.getWeekDownloadBytes()
}

func (nodeStats *NodeStats) fillNodeStatsFromNodeEntry(nodeEntry *NodeEntry) {
	nodeStats.hourCount = nodeEntry.StatsHourCount
	nodeStats.hourUploadNanoSec = nodeEntry.StatsHourUploadNanoSec
	nodeStats.hourUploadBytes = nodeEntry.StatsHourUploadBytes
	nodeStats.hourLatency = nodeEntry.StatsHourLatency
	nodeStats.dayCount = nodeEntry.StatsDayCount
	nodeStats.dayUploadNanoSec = nodeEntry.StatsDayUploadNanoSec
	nodeStats.dayUploadBytes = nodeEntry.StatsDayUploadBytes
	nodeStats.dayLatency = nodeEntry.StatsDayLatency
	nodeStats.weekCount = nodeEntry.StatsWeekCount
	nodeStats.weekUploadNanoSec = nodeEntry.StatsWeekUploadNanoSec
	nodeStats.weekUploadBytes = nodeEntry.StatsWeekUploadBytes
	nodeStats.weekLatency = nodeEntry.StatsWeekLatency
}

func (netStats *NetStats) fillNetStatsFromNetEntry(netEntry *NetEntry) {
	netStats.hourCount = netEntry.StatsHourCount
	netStats.hourDownloadNanoSec = netEntry.StatsHourDownloadNanoSec
	netStats.hourDownloadBytes = netEntry.StatsHourDownloadBytes
	netStats.dayCount = netEntry.StatsDayCount
	netStats.dayDownloadNanoSec = netEntry.StatsDayDownloadNanoSec
	netStats.dayDownloadBytes = netEntry.StatsDayDownloadBytes
	netStats.weekCount = netEntry.StatsWeekCount
	netStats.weekDownloadNanoSec = netEntry.StatsWeekDownloadNanoSec
	netStats.weekDownloadBytes = netEntry.StatsWeekDownloadBytes
}

func (nodeEntry *NodeEntry) fillNodeEntryFromNodeStats(nodeStats *NodeStats) {
	nodeEntry.StatsHourCount = nodeStats.hourCount
	nodeEntry.StatsHourUploadNanoSec = nodeStats.hourUploadNanoSec
	nodeEntry.StatsHourUploadBytes = nodeStats.hourUploadBytes
	nodeEntry.StatsHourLatency = nodeStats.hourLatency
	nodeEntry.StatsDayCount = nodeStats.dayCount
	nodeEntry.StatsDayUploadNanoSec = nodeStats.dayUploadNanoSec
	nodeEntry.StatsDayUploadBytes = nodeStats.dayUploadBytes
	nodeEntry.StatsDayLatency = nodeStats.dayLatency
	nodeEntry.StatsWeekCount = nodeStats.weekCount
	nodeEntry.StatsWeekUploadNanoSec = nodeStats.weekUploadNanoSec
	nodeEntry.StatsWeekUploadBytes = nodeStats.weekUploadBytes
	nodeEntry.StatsWeekLatency = nodeStats.weekLatency
}

func (netEntry *NetEntry) fillNetEntryFromNetStats(netStats *NetStats) {
	netEntry.StatsHourCount = netStats.hourCount
	netEntry.StatsHourDownloadNanoSec = netStats.hourDownloadNanoSec
	netEntry.StatsHourDownloadBytes = netStats.hourDownloadBytes
	netEntry.StatsDayCount = netStats.dayCount
	netEntry.StatsDayDownloadNanoSec = netStats.dayDownloadNanoSec
	netEntry.StatsDayDownloadBytes = netStats.dayDownloadBytes
	netEntry.StatsWeekCount = netStats.weekCount
	netEntry.StatsWeekDownloadNanoSec = netStats.weekDownloadNanoSec
	netEntry.StatsWeekDownloadBytes = netStats.weekDownloadBytes
}

func (l *NodeStats) isBetterThan(r *NodeStats) bool {
	isBetter := false
	if l.hourLatency < r.hourLatency {
		isBetter = true
	}
	return isBetter
}

func (l *NetStats) isBetterThan(r *NetStats) bool {
	isBetter := false
	lBandwidth := float64(l.hourDownloadBytes) / float64(l.hourDownloadNanoSec)
	rBandwidth := float64(r.hourDownloadBytes) / float64(r.hourDownloadNanoSec)
	if lBandwidth > rBandwidth {
		isBetter = true
	}
	return isBetter
}
