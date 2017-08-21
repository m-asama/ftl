package ftl

import (
	"encoding/binary"
	"net"
)

func NodeId2Addr4(nid NodeId) net.IP {
	tmp := make([]byte, 4)
	tmp[0] = byte(uint32(nid) >> 24)
	tmp[1] = byte(uint32(nid) >> 16)
	tmp[2] = byte(uint32(nid) >> 8)
	tmp[3] = byte(uint32(nid))
	return net.IPv4(tmp[0], tmp[1], tmp[2], tmp[3])
}

func NetId2Addr4(nid NetId) net.IP {
	tmp := make([]byte, 4)
	tmp[0] = byte(uint32(nid) >> 24)
	tmp[1] = byte(uint32(nid) >> 16)
	tmp[2] = byte(uint32(nid) >> 8)
	tmp[3] = byte(uint32(nid))
	return net.IPv4(tmp[0], tmp[1], tmp[2], tmp[3])
}

func IP2NetId(ip net.IP) NetId {
	var netId NetId
	if tmp := ip.To4(); tmp != nil {
		netId = NetId(binary.BigEndian.Uint32(tmp) >> 8)
	} else {
		tmp = ip[0:4]
		netId = NetId(binary.BigEndian.Uint32(tmp))
	}
	return netId
}
