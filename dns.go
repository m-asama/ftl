package ftl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"regexp"
)

const (
	DNS_PORT = 5353
	DNS_TTL  = 20
)

type DNSQ struct {
	DomainName string
	Type       uint16
	Class      uint16
}

type DNSRR struct {
	DomainName string
	Type       uint16
	Class      uint16
	TTL        uint32
	RDLength   uint16
	RData      []byte
}

type DNSPacket struct {
	ID      uint16
	QR      bool
	OPCODE  uint8
	AA      bool
	TC      bool
	RD      bool
	RA      bool
	Z       bool
	AD      bool
	CD      bool
	RCODE   uint8
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
	QD      []*DNSQ
	AN      []*DNSRR
	NS      []*DNSRR
	AR      []*DNSRR
}

type DomainNameType int

const (
	FTL_CACHE = iota
	FTL_OBSERVATION
	DISCARD
)

func domainNameType(domainName string) DomainNameType {
	isFTLCache := regexp.MustCompile(`\.ftl-cache\.`)
	isFTLObservation := regexp.MustCompile(`\.ftl-observation\.`)
	switch {
	case isFTLCache.MatchString(domainName):
		return FTL_CACHE
	case isFTLObservation.MatchString(domainName):
		return FTL_OBSERVATION
	}
	return DISCARD
}

func originDomainName(domainName string) string {
	switch domainNameType(domainName) {
	case FTL_CACHE:
		re := regexp.MustCompile(`\.ftl-cache\.`)
		domainName = re.ReplaceAllLiteralString(domainName, ".")
	case FTL_OBSERVATION:
		re := regexp.MustCompile(`\.ftl-observation\.`)
		domainName = re.ReplaceAllLiteralString(domainName, ".")
	}
	return domainName
}

func parseLabel(data []byte, size int, i int) (*string, int, error) {
	//logError("parseLabel: begin")
	//defer logError("parseLabel: end")
	oldi := i
	comp := false
	if size < i+2 {
		logError("parseLabel: label parse error(1):",
			fmt.Sprint("data =", data, "size =", size, "i =", i))
		return nil, i, errors.New("parse error")
	}
	if (data[i] & 0xc0) == 0xc0 {
		i = int(binary.BigEndian.Uint16(data[i:i+2]) & 0x3fff)
		comp = true
	}
	label := ""
	fin := false
	for size >= i+1 {
		if data[i] == 0 {
			i++
			fin = true
			break
		}
		labellen := int(data[i])
		i++
		if size < i+labellen {
			logError("parseLabel: label parse error(2)",
				fmt.Sprint("data =", data, "size =", size, "i =", i))
			return nil, i, errors.New("parse error")
		}
		if label != "" {
			label += "."
		}
		label += string(data[i : i+labellen])
		i += labellen
	}
	if !fin {
		logError("parseLabel: label parse error(3)",
			fmt.Sprint("data =", data, "size =", size, "i =", i))
		return nil, i, errors.New("parse error")
	}
	if comp {
		i = oldi + 2
	}
	return &label, i, nil
}

func parseDNSPacket(data []byte, size int) (*DNSPacket, error) {
	//logError("parseDNSPacket: begin")
	//defer logError("parseDNSPacket: end")
	if size < 12 {
		logError("parseDNSPacket: parse error(1)",
			fmt.Sprint("data =", data, "size =", size))
		return nil, errors.New("parse error(1)")
	}
	p := &DNSPacket{}
	p.ID = binary.BigEndian.Uint16(data[0:2])
	p.QR = (data[2] & 0x80) == 0x80
	p.OPCODE = (data[2] >> 3) & 0x0f
	p.AA = (data[2] & 0x04) == 0x04
	p.TC = (data[2] & 0x02) == 0x02
	p.RD = (data[2] & 0x01) == 0x01
	p.RA = (data[3] & 0x80) == 0x80
	p.Z = ((data[3] >> 4) & 0x07) == 0x07
	p.RCODE = data[3] & 0x0f
	p.QDCOUNT = binary.BigEndian.Uint16(data[4:6])
	p.ANCOUNT = binary.BigEndian.Uint16(data[6:8])
	p.NSCOUNT = binary.BigEndian.Uint16(data[8:10])
	p.ARCOUNT = binary.BigEndian.Uint16(data[10:12])
	p.QD = make([]*DNSQ, p.QDCOUNT)
	p.AN = make([]*DNSRR, p.ANCOUNT)
	p.NS = make([]*DNSRR, p.NSCOUNT)
	p.AR = make([]*DNSRR, p.ARCOUNT)
	i := 12
	var label *string
	var err error
	for qdi := 0; qdi < int(p.QDCOUNT); qdi++ {
		label, i, err = parseLabel(data, size, i)
		if err != nil {
			logError("parseDNSPacket: parse error(2)", fmt.Sprint(err))
			return nil, errors.New("parse error(2)")
		}
		qd := &DNSQ{}
		qd.DomainName = *label
		qd.Type = binary.BigEndian.Uint16(data[i : i+2])
		qd.Class = binary.BigEndian.Uint16(data[i+2 : i+4])
		p.QD[qdi] = qd
		i += 4
	}
	for ani := 0; ani < int(p.ANCOUNT); ani++ {
		label, i, err = parseLabel(data, size, i)
		if err != nil {
			logError("parseDNSPacket: parse error(3)", fmt.Sprint(err))
			return nil, errors.New("parse error(3)")
		}
		an := &DNSRR{}
		an.DomainName = *label
		an.Type = binary.BigEndian.Uint16(data[i : i+2])
		an.Class = binary.BigEndian.Uint16(data[i+2 : i+4])
		an.TTL = binary.BigEndian.Uint32(data[i+4 : i+8])
		an.RDLength = binary.BigEndian.Uint16(data[i+8 : i+10])
		an.RData = make([]byte, int(an.RDLength))
		an.RData = data[i+10 : i+10+int(an.RDLength)]
		p.AN[ani] = an
		i += 10 + int(an.RDLength)
	}
	for nsi := 0; nsi < int(p.NSCOUNT); nsi++ {
		label, i, err = parseLabel(data, size, i)
		if err != nil {
			logError("parseDNSPacket: parse error(4)", fmt.Sprint(err))
			return nil, errors.New("parse error(4)")
		}
		ns := &DNSRR{}
		ns.DomainName = *label
		ns.Type = binary.BigEndian.Uint16(data[i : i+2])
		ns.Class = binary.BigEndian.Uint16(data[i+2 : i+4])
		ns.TTL = binary.BigEndian.Uint32(data[i+4 : i+8])
		ns.RDLength = binary.BigEndian.Uint16(data[i+8 : i+10])
		ns.RData = data[i+10 : i+10+int(ns.RDLength)]
		p.NS[nsi] = ns
		i += 10 + int(ns.RDLength)
	}
	for ari := 0; ari < int(p.ARCOUNT); ari++ {
		label, i, err = parseLabel(data, size, i)
		if err != nil {
			logError("parseDNSPacket: parse error(5)", fmt.Sprint(err))
			return nil, errors.New("parse error(5)")
		}
		ar := &DNSRR{}
		ar.DomainName = *label
		ar.Type = binary.BigEndian.Uint16(data[i : i+2])
		ar.Class = binary.BigEndian.Uint16(data[i+2 : i+4])
		ar.TTL = binary.BigEndian.Uint32(data[i+4 : i+8])
		ar.RDLength = binary.BigEndian.Uint16(data[i+8 : i+10])
		ar.RData = data[i+10 : i+10+int(ar.RDLength)]
		p.AR[ari] = ar
		i += 10 + int(ar.RDLength)
	}
	if i != size {
		logError("parseDNSPacket: parse error(6)")
		logError("parseDNSPacket: data =", fmt.Sprint(data), "size =", fmt.Sprint(size))
		logError("parseDNSPacket:", p.dump())
		return nil, errors.New("parse error(6)")
	}
	return p, nil
}

func (p *DNSPacket) dump() string {
	s := fmt.Sprintf("\n")
	s += fmt.Sprintf("ID     %04x\n", p.ID)
	s += fmt.Sprintf("QR     %5t\n", p.QR)
	s += fmt.Sprintf("OPCODE %02x\n", p.OPCODE)
	s += fmt.Sprintf("AA     %5t\n", p.AA)
	s += fmt.Sprintf("TC     %5t\n", p.TC)
	s += fmt.Sprintf("RD     %5t\n", p.RD)
	s += fmt.Sprintf("RA     %5t\n", p.RA)
	s += fmt.Sprintf("Z      %5t\n", p.Z)
	s += fmt.Sprintf("AD     %5t\n", p.AD)
	s += fmt.Sprintf("CD     %5t\n", p.CD)
	s += fmt.Sprintf("RCODE  %02x\n", p.RCODE)
	s += fmt.Sprintf("QDCOUNT %2d\n", p.QDCOUNT)
	s += fmt.Sprintf("ANCOUNT %2d\n", p.ANCOUNT)
	s += fmt.Sprintf("NSCOUNT %2d\n", p.NSCOUNT)
	s += fmt.Sprintf("ARCOUNT %2d\n", p.ARCOUNT)
	for i := 0; i < len(p.QD); i++ {
		s += fmt.Sprintf("    QD[%d]:\n", i)
		s += fmt.Sprintf("        DomainName  %s\n", p.QD[i].DomainName)
		s += fmt.Sprintf("        Type        0x%04x\n", p.QD[i].Type)
		s += fmt.Sprintf("        Class       0x%04x\n", p.QD[i].Class)
	}
	for i := 0; i < len(p.AN); i++ {
		s += fmt.Sprintf("    AN[%d]:\n", i)
		s += fmt.Sprintf("        DomainName  %s\n", p.AN[i].DomainName)
		s += fmt.Sprintf("        Type        0x%04x\n", p.AN[i].Type)
		s += fmt.Sprintf("        Class       0x%04x\n", p.AN[i].Class)
		s += fmt.Sprintf("        TTL         %8d\n", p.AN[i].TTL)
		s += fmt.Sprintf("        RDLength    %8d\n", p.AN[i].RDLength)
		s += fmt.Sprintf("        RData       %x\n", p.AN[i].RData)
	}
	for i := 0; i < len(p.NS); i++ {
		s += fmt.Sprintf("    NS[%d]:\n", i)
		s += fmt.Sprintf("        DomainName  %s\n", p.NS[i].DomainName)
		s += fmt.Sprintf("        Type        0x%04x\n", p.NS[i].Type)
		s += fmt.Sprintf("        Class       0x%04x\n", p.NS[i].Class)
		s += fmt.Sprintf("        TTL         %8d\n", p.NS[i].TTL)
		s += fmt.Sprintf("        RDLength    %8d\n", p.NS[i].RDLength)
		s += fmt.Sprintf("        RData       %x\n", p.NS[i].RData)
	}
	for i := 0; i < len(p.AR); i++ {
		s += fmt.Sprintf("    AR[%d]:\n", i)
		s += fmt.Sprintf("        DomainName  %s\n", p.AR[i].DomainName)
		s += fmt.Sprintf("        Type        0x%04x\n", p.AR[i].Type)
		s += fmt.Sprintf("        Class       0x%04x\n", p.AR[i].Class)
		s += fmt.Sprintf("        TTL         %8d\n", p.AR[i].TTL)
		s += fmt.Sprintf("        RDLength    %8d\n", p.AR[i].RDLength)
		s += fmt.Sprintf("        RData       %x\n", p.AR[i].RData)
	}
	return s
}

func fillLabel(data []byte, labelstr string) {
	var i, t int
	label := []byte(labelstr)
	for i < len(label) {
		if label[i] == '.' {
			data[i-t] = byte(t)
			t = -1
		}
		data[i+1] = label[i]
		t++
		i++
	}
	data[i-t] = byte(t)
}

func (p *DNSPacket) encode() ([]byte, int, error) {
	labels := make(map[string]int)
	size := 12
	for i := 0; i < len(p.QD); i++ {
		_, ok := labels[p.QD[i].DomainName]
		if p.QD[i].DomainName == "" {
			size += 1
		} else if !ok {
			labels[p.QD[i].DomainName] = size
			size += len(p.QD[i].DomainName) + 2
		} else {
			size += 2
		}
		size += 4
	}
	for i := 0; i < len(p.AN); i++ {
		_, ok := labels[p.AN[i].DomainName]
		if p.AN[i].DomainName == "" {
			size += 1
		} else if !ok {
			labels[p.AN[i].DomainName] = size
			size += len(p.AN[i].DomainName) + 2
		} else {
			size += 2
		}
		size += 10 + int(p.AN[i].RDLength)
	}
	for i := 0; i < len(p.NS); i++ {
		_, ok := labels[p.NS[i].DomainName]
		if p.NS[i].DomainName == "" {
			size += 1
		} else if !ok {
			labels[p.NS[i].DomainName] = size
			size += len(p.NS[i].DomainName) + 2
		} else {
			size += 2
		}
		size += 10 + int(p.NS[i].RDLength)
	}
	for i := 0; i < len(p.AR); i++ {
		_, ok := labels[p.AR[i].DomainName]
		if p.AR[i].DomainName == "" {
			size += 1
		} else if !ok {
			labels[p.AR[i].DomainName] = size
			size += len(p.AR[i].DomainName) + 2
		} else {
			size += 2
		}
		size += 10 + int(p.AR[i].RDLength)
	}
	data := make([]byte, size)
	binary.BigEndian.PutUint16(data[0:2], p.ID)
	data[2] = 0
	if p.QR {
		data[2] |= 0x80
	}
	data[2] |= (p.OPCODE << 3) & 0x78
	if p.AA {
		data[2] |= 0x04
	}
	if p.TC {
		data[2] |= 0x02
	}
	if p.RD {
		data[2] |= 0x01
	}
	if p.RA {
		data[3] |= 0x80
	}
	if p.Z {
		data[3] |= 0x40
	}
	if p.AD {
		data[3] |= 0x20
	}
	if p.CD {
		data[3] |= 0x10
	}
	data[3] |= p.RCODE & 0x0f
	binary.BigEndian.PutUint16(data[4:6], p.QDCOUNT)
	binary.BigEndian.PutUint16(data[6:8], p.ANCOUNT)
	binary.BigEndian.PutUint16(data[8:10], p.NSCOUNT)
	binary.BigEndian.PutUint16(data[10:12], p.ARCOUNT)
	i := 12
	for qdi := 0; qdi < len(p.QD); qdi++ {
		labelindex, ok := labels[p.QD[qdi].DomainName]
		if p.QD[qdi].DomainName == "" {
			data[i] = 0
			i += 1
		} else if ok && i != labelindex {
			binary.BigEndian.PutUint16(data[i:i+2], uint16(labelindex))
			data[i] |= 0xc0
			i += 2
		} else {
			fillLabel(data[i:i+len(p.QD[qdi].DomainName)+2], p.QD[qdi].DomainName)
			i += len(p.QD[qdi].DomainName) + 2
		}
		binary.BigEndian.PutUint16(data[i:i+2], p.QD[qdi].Type)
		binary.BigEndian.PutUint16(data[i+2:i+4], p.QD[qdi].Class)
		i += 4
	}
	for ani := 0; ani < len(p.AN); ani++ {
		labelindex, ok := labels[p.AN[ani].DomainName]
		if p.AN[ani].DomainName == "" {
			data[i] = 0
			i += 1
		} else if ok && i != labelindex {
			binary.BigEndian.PutUint16(data[i:i+2], uint16(labelindex))
			data[i] |= 0xc0
			i += 2
		} else {
			fillLabel(data[i:i+len(p.AN[ani].DomainName)+2], p.AN[ani].DomainName)
			i += len(p.AN[ani].DomainName) + 2
		}
		binary.BigEndian.PutUint16(data[i:i+2], p.AN[ani].Type)
		binary.BigEndian.PutUint16(data[i+2:i+4], p.AN[ani].Class)
		binary.BigEndian.PutUint32(data[i+4:i+8], p.AN[ani].TTL)
		binary.BigEndian.PutUint16(data[i+8:i+10], p.AN[ani].RDLength)
		for j := 0; j < int(p.AN[ani].RDLength); j++ {
			data[i+10+j] = p.AN[ani].RData[j]
		}
		i += 10 + int(p.AN[ani].RDLength)
	}
	for nsi := 0; nsi < len(p.NS); nsi++ {
		labelindex, ok := labels[p.NS[nsi].DomainName]
		if p.NS[nsi].DomainName == "" {
			data[i] = 0
			i += 1
		} else if ok && i != labelindex {
			binary.BigEndian.PutUint16(data[i:i+2], uint16(labelindex))
			data[i] |= 0xc0
			i += 2
		} else {
			fillLabel(data[i:i+len(p.NS[nsi].DomainName)+2], p.NS[nsi].DomainName)
			i += len(p.NS[nsi].DomainName) + 2
		}
		binary.BigEndian.PutUint16(data[i:i+2], p.NS[nsi].Type)
		binary.BigEndian.PutUint16(data[i+2:i+4], p.NS[nsi].Class)
		binary.BigEndian.PutUint32(data[i+4:i+8], p.NS[nsi].TTL)
		binary.BigEndian.PutUint16(data[i+8:i+10], p.NS[nsi].RDLength)
		for j := 0; j < int(p.NS[nsi].RDLength); j++ {
			data[i+10+j] = p.NS[nsi].RData[j]
		}
		i += 10 + int(p.NS[nsi].RDLength)
	}
	for ari := 0; ari < len(p.AR); ari++ {
		labelindex, ok := labels[p.AR[ari].DomainName]
		if p.AR[ari].DomainName == "" {
			data[i] = 0
			i += 1
		} else if ok && i != labelindex {
			binary.BigEndian.PutUint16(data[i:i+2], uint16(labelindex))
			data[i] |= 0xc0
			i += 2
		} else {
			fillLabel(data[i:i+len(p.AR[ari].DomainName)+2], p.AR[ari].DomainName)
			i += len(p.AR[ari].DomainName) + 2
		}
		binary.BigEndian.PutUint16(data[i:i+2], p.AR[ari].Type)
		binary.BigEndian.PutUint16(data[i+2:i+4], p.AR[ari].Class)
		binary.BigEndian.PutUint32(data[i+4:i+8], p.AR[ari].TTL)
		binary.BigEndian.PutUint16(data[i+8:i+10], p.AR[ari].RDLength)
		for j := 0; j < int(p.AR[ari].RDLength); j++ {
			data[i+10+j] = p.AR[ari].RData[j]
		}
		i += 10 + int(p.AR[ari].RDLength)
	}
	return data, size, nil
}

func (p *DNSPacket) clone() *DNSPacket {
	c := &DNSPacket{}
	c.ID = p.ID
	c.QR = p.QR
	c.OPCODE = p.OPCODE
	c.AA = p.AA
	c.TC = p.TC
	c.RD = p.RD
	c.RA = p.RA
	c.Z = p.Z
	c.AD = p.AD
	c.CD = p.CD
	c.RCODE = p.RCODE
	c.QDCOUNT = p.QDCOUNT
	c.ANCOUNT = p.ANCOUNT
	c.NSCOUNT = p.NSCOUNT
	c.ARCOUNT = p.ARCOUNT
	c.QD = make([]*DNSQ, len(p.QD))
	for i := 0; i < len(p.QD); i++ {
		c.QD[i] = &DNSQ{}
		c.QD[i].DomainName = p.QD[i].DomainName
		c.QD[i].Type = p.QD[i].Type
		c.QD[i].Class = p.QD[i].Class
	}
	c.AN = make([]*DNSRR, len(p.AN))
	for i := 0; i < len(p.AN); i++ {
		c.AN[i] = &DNSRR{}
		c.AN[i].DomainName = p.AN[i].DomainName
		c.AN[i].Type = p.AN[i].Type
		c.AN[i].Class = p.AN[i].Class
		c.AN[i].TTL = p.AN[i].TTL
		c.AN[i].RDLength = p.AN[i].RDLength
		c.AN[i].RData = p.AN[i].RData
	}
	c.NS = make([]*DNSRR, len(p.NS))
	for i := 0; i < len(p.NS); i++ {
		c.NS[i] = &DNSRR{}
		c.NS[i].DomainName = p.NS[i].DomainName
		c.NS[i].Type = p.NS[i].Type
		c.NS[i].Class = p.NS[i].Class
		c.NS[i].TTL = p.NS[i].TTL
		c.NS[i].RDLength = p.NS[i].RDLength
		c.NS[i].RData = p.NS[i].RData
	}
	c.AR = make([]*DNSRR, len(p.AR))
	for i := 0; i < len(p.AR); i++ {
		c.AR[i] = &DNSRR{}
		c.AR[i].DomainName = p.AR[i].DomainName
		c.AR[i].Type = p.AR[i].Type
		c.AR[i].Class = p.AR[i].Class
		c.AR[i].TTL = p.AR[i].TTL
		c.AR[i].RDLength = p.AR[i].RDLength
		c.AR[i].RData = p.AR[i].RData
	}
	return c
}

func makeRData(from net.IP, domainName string) []byte {
	netId := IP2NetId(from)
	tmpAddr := myAddr4
	var tmpNode *Node
	switch domainNameType(domainName) {
	case FTL_CACHE:
		tmpNode = findBestNode(netId)
	case FTL_OBSERVATION:
		tmpNode = findUnknownNode(netId)
	}
	if tmpNode != nil {
		tmpAddr = tmpNode.addr4
	} else {
		tmpAddr = myAddr4
	}
	tmpAddr = tmpAddr.To4()
	rdata := []byte(tmpAddr)
	return rdata
}

func makeDNSResponse(query []byte, size int, from net.IP) ([]byte, error) {
	//logError("makeDNSResponse: begin")
	//defer logError("makeDNSResponse: end")
	q, err := parseDNSPacket(query, size)
	if err != nil {
		logError("makeDNSResponse: parseDNSPacket failed(1):", fmt.Sprint(err))
		return nil, errors.New("parseDNSPacket failed(1)")
	}
	r := q.clone()
	r.ARCOUNT = 0
	r.AR = make([]*DNSRR, 0)
	r.QR = true
	if len(r.QD) > 0 && r.QD[0].Type == 1 && domainNameType(q.QD[0].DomainName) != DISCARD {
		r.ANCOUNT = 1
		r.AN = make([]*DNSRR, 1)
		r.AN[0] = &DNSRR{}
		r.AN[0].DomainName = r.QD[0].DomainName
		r.AN[0].Type = r.QD[0].Type
		r.AN[0].Class = r.QD[0].Class
		r.AN[0].TTL = DNS_TTL
		r.AN[0].RDLength = 4
		r.AN[0].RData = makeRData(from, q.QD[0].DomainName)
	} else {
		r.RCODE = 3
	}
	response, st, err := r.encode()
	if err != nil {
		logError("makeDNSResponse: encode error: ", fmt.Sprint(err))
		return nil, errors.New("encode error")
	}
	_, err = parseDNSPacket(response, st)
	if err != nil {
		logError("makeDNSResponse: parseDNSPacket failed(2):", fmt.Sprint(err))
		logError("makeDNSResponse: ", fmt.Sprint(query[0:size]), fmt.Sprint(size))
		logError("makeDNSResponse: ", q.dump())
		return nil, errors.New("parseDNSPacket failed(2):")
	}
	return response, nil
}

func handleDNSQuery(conn *net.UDPConn) {
	//logError("handleDNSQuery: begin")
	//defer logError("handleDNSQuery: end")
	in := make([]byte, 2048)

	inSize, client, err := conn.ReadFromUDP(in)
	if err != nil {
		logError("handleDNSQuery: ReadFromUDP failed:", fmt.Sprint(err))
		return
	}

	out, err := makeDNSResponse(in, inSize, client.IP)
	if err != nil {
		logError("handleDNSQuery: makeDNSResponse failed:", fmt.Sprint(err))
		return
	}

	_, err = conn.WriteToUDP(out, client)
	if err != nil {
		logError("handleDNSQuery: WriteToUDP failed:", fmt.Sprint(err))
		return
	}
}

func DNSService() {
	server, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DNS_PORT))
	if err != nil {
		logError("DNSService: ResolveUDPAddr failed:", fmt.Sprint(err))
		return
	}

	conn, err := net.ListenUDP("udp", server)
	if err != nil {
		logError("DNSService: ListenUDP failed:", fmt.Sprint(err))
		return
	}
	defer conn.Close()

	for {
		handleDNSQuery(conn)
	}
}
