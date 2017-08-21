package ftl

import (
	"bufio"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	HTTP_PORT                 = 8080
	RETURN_CACHE_TIMEOUT      = 60 * 10
	MAX_CACHE_FILE_SIZE       = 8 * 1024 * 1024 * 1024
	FORWARDED_EXPIRE          = 5 * 1000000000
	OBSERVATION_CACHE_MAX_AGE = 60
)

const (
	etagStr          = "etag"
	contentTypeStr   = "content-type"
	contentLengthStr = "content-length"
	lastModifiedStr  = "last-modified"
)

var (
	etagKey          = textproto.CanonicalMIMEHeaderKey(etagStr)
	contentTypeKey   = textproto.CanonicalMIMEHeaderKey(contentTypeStr)
	contentLengthKey = textproto.CanonicalMIMEHeaderKey(contentLengthStr)
	lastModifiedKey  = textproto.CanonicalMIMEHeaderKey(lastModifiedStr)
)

var dataDirLock sync.Mutex

func cacheFileDir(host string) string {
	return fmt.Sprintf("%s/%s", dataDir, host)
}

func cacheFilePathBase(host, uri string) string {
	hash := sha1.New()
	io.WriteString(hash, uri)
	uriHash := fmt.Sprintf("%x", hash.Sum(nil))
	return fmt.Sprintf("%s/%s/%s", dataDir, host, uriHash)
}

type FtlCacheHeader struct {
	Etag          string
	ContentType   string
	ContentLength int
	LastModified  int64
}

func (lhs *FtlCacheHeader) equal(rhs *FtlCacheHeader) bool {
	if lhs.Etag != rhs.Etag {
		return false
	}
	if lhs.ContentType != rhs.ContentType {
		return false
	}
	if lhs.ContentLength != rhs.ContentLength {
		return false
	}
	if lhs.LastModified != rhs.LastModified {
		return false
	}
	return true
}

func (ftlCacheHeader *FtlCacheHeader) parseHeader(headerPath string) error {
	var err error

	if ftlCacheHeader == nil {
		logError("parseHeader: \"", headerPath, "\": ftlCacheHeader nil.")
		return errors.New("ftlCacheHeader nil.")
	}

	headerFp, err := os.Open(headerPath)
	if err != nil {
		logError("parseHeader: \"", headerPath, "\": file not found.")
		return errors.New("file not found.")
	}

	headerReader := bufio.NewReaderSize(headerFp, 4096)
	r := regexp.MustCompile(`^([^:]+)\s*:\s*(.*)$`)
	for {
		line, _, err := headerReader.ReadLine()
		if err != nil {
			break
		}
		kv := r.FindStringSubmatch(string(line))
		if len(kv) != 3 {
			continue
		}
		if textproto.CanonicalMIMEHeaderKey(kv[1]) == etagKey {
			ftlCacheHeader.Etag = kv[2]
		}
		if textproto.CanonicalMIMEHeaderKey(kv[1]) == contentTypeKey {
			ftlCacheHeader.ContentType = kv[2]
		}
		if textproto.CanonicalMIMEHeaderKey(kv[1]) == contentLengthKey {
			l, err := strconv.Atoi(kv[2])
			if err == nil {
				ftlCacheHeader.ContentLength = l
			}
		}
		if textproto.CanonicalMIMEHeaderKey(kv[1]) == lastModifiedKey {
			t, err := http.ParseTime(kv[2])
			if err == nil {
				ftlCacheHeader.LastModified = int64(t.Unix())
			}
		}
	}
	return nil
}

func (ftlCacheHeader *FtlCacheHeader) parseResponse(resp *http.Response) error {
	if ftlCacheHeader == nil {
		logError("parseResponse: ftlCacheHeader nil.")
		return errors.New("ftlCacheHeader nil.")
	}
	etag := resp.Header.Get(etagStr)
	if etag != "" {
		ftlCacheHeader.Etag = etag
	}
	contentType := resp.Header.Get(contentTypeStr)
	if contentType != "" {
		ftlCacheHeader.ContentType = contentType
	}
	contentLength := resp.Header.Get(contentLengthStr)
	if contentLength != "" {
		l, err := strconv.Atoi(contentLength)
		if err == nil {
			ftlCacheHeader.ContentLength = l
		}
	}
	lastModified := resp.Header.Get(lastModifiedStr)
	if lastModified != "" {
		t, err := http.ParseTime(lastModified)
		if err == nil {
			ftlCacheHeader.LastModified = t.Unix()
		}
	}
	return nil
}

func (ftlCacheHeader *FtlCacheHeader) dumpHeader(headerPath string) error {
	if ftlCacheHeader == nil {
		logError("dumpHeader: \"", headerPath, "\": ftlCacheHeader nil.")
		return errors.New("ftlCacheHeader nil.")
	}
	headerFp, err := os.Create(headerPath)
	if err != nil {
		logError("dumpHeader: \"", headerPath, "\": create failed.")
		return errors.New("create failed.")
	}
	headerWriter := bufio.NewWriterSize(headerFp, 4096)
	if ftlCacheHeader.Etag != "" {
		headerWriter.WriteString(fmt.Sprintf("%s: %s\n", etagKey, ftlCacheHeader.Etag))
	}
	headerWriter.WriteString(fmt.Sprintf("%s: %s\n", contentTypeKey, ftlCacheHeader.ContentType))
	headerWriter.WriteString(fmt.Sprintf("%s: %d\n", contentLengthKey, ftlCacheHeader.ContentLength))
	if ftlCacheHeader.LastModified != 0 {
		lastModifiedTmp := time.Unix(ftlCacheHeader.LastModified, 0).UTC().Format(http.TimeFormat)
		headerWriter.WriteString(fmt.Sprintf("%s: %s\n", lastModifiedKey, lastModifiedTmp))
	}
	headerWriter.Flush()
	headerFp.Close()
	return nil
}

func deleteOldCache(host, uri, etag, contentType, contentLength, lastModified string) {
	//logError("deleteOldCache: begin")
	//defer logError("deleteOldCache: end")

	base := cacheFilePathBase(host, uri)
	header := fmt.Sprintf("%s.h", base)
	body := fmt.Sprintf("%s.b", base)

	var err error

	dataDirLock.Lock()
	defer dataDirLock.Unlock()

	ch := &FtlCacheHeader{}

	err = ch.parseHeader(header)
	if err != nil {
		logError("deleteOldCache: parseHeader(\"", header, "\") error.")
		return
	}
	cacheOld := false
	if ch.Etag != etag {
		cacheOld = true
	}
	if ch.ContentType != contentType {
		cacheOld = true
	}
	l, err := strconv.Atoi(contentLength)
	if err != nil || ch.ContentLength != l {
		cacheOld = true
	}
	t, err := http.ParseTime(lastModified)
	if err != nil || ch.LastModified != t.Unix() {
		cacheOld = true
	}
	if cacheOld {
		os.Remove(header)
		os.Remove(body)
	}
}

func ftlHeadHandler(writer http.ResponseWriter, req *http.Request) int {
	//logError("ftlHeadHandler: begin")
	//defer logError("ftlHeadHandler: end")

	var err error

	host := originDomainName(req.Host)
	uri := req.RequestURI

	resp, err := http.Head(fmt.Sprintf("http://%s%s", host, uri))
	if err != nil {
		logError("ftlHeadHandler: http.Head:", host, uri, "failed.")
		writer.WriteHeader(502)
		return 502
	}

	etag := resp.Header.Get(etagStr)
	if etag != "" {
		writer.Header().Set(etagStr, etag)
	}
	contentType := resp.Header.Get(contentTypeStr)
	if contentType != "" {
		writer.Header().Set(contentTypeStr, contentType)
	}
	contentLength := resp.Header.Get(contentLengthStr)
	if contentLength != "" {
		writer.Header().Set(contentLengthStr, contentLength)
	}
	lastModified := resp.Header.Get(lastModifiedStr)
	if lastModified != "" {
		writer.Header().Set(lastModifiedStr, lastModified)
	}
	writer.Header().Set("connection", "close")
	deleteOldCache(host, uri, etag, contentType, contentLength, lastModified)
	writer.WriteHeader(resp.StatusCode)
	return resp.StatusCode
}

func getOriginAndWriteCache(host, uri string, status chan int) {
	//logError("getOriginAndWriteCache: begin")
	//defer logError("getOriginAndWriteCache: end")

	base := cacheFilePathBase(host, uri)
	header := fmt.Sprintf("%s.h", base)
	body := fmt.Sprintf("%s.b", base)

	resp, err := http.Get(fmt.Sprintf("http://%s%s", host, uri))
	if err != nil {
		logError("getOriginAndWriteCache: http.Get:", host, uri, "failed.")
		status <- 502
		return
	}
	defer resp.Body.Close()

	ch := &FtlCacheHeader{}

	err = ch.parseResponse(resp)
	if err != nil {
		logError("getOriginAndWriteCache: ch.parseResponse:", host, uri)
		status <- 502
		return
	}

	if ch.ContentLength <= 0 || ch.ContentLength > MAX_CACHE_FILE_SIZE {
		logError("getOriginAndWriteCache: ch.ContentLength:", fmt.Sprint(ch.ContentLength), host, uri)
		status <- 502
		return
	}

	dataDirLock.Lock()
	os.Remove(header)
	os.Remove(body)
	err = ch.dumpHeader(header)
	if err != nil {
		logError("getOriginAndWriteCache: ch.dumpHeader:", header, host, uri)
		dataDirLock.Unlock()
		status <- 502
		return
	}
	fp, err := os.OpenFile(body, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logError("getOriginAndWriteCache: os.OpenFile:", body, host, uri)
		os.Remove(header)
		dataDirLock.Unlock()
		status <- 502
		return
	}
	defer fp.Close()
	dataDirLock.Unlock()

	status <- resp.StatusCode

	buf := make([]byte, 8192)
	for {
		n, err := resp.Body.Read(buf)
		fp.Write(buf[0:n])
		if err != nil {
			break
		}
	}

	return
}

func returnCache(writer http.ResponseWriter, bodyFp *os.File, ch *FtlCacheHeader, domType DomainNameType) (uint64, uint64) {
	//logError("returnCache: begin")
	//defer logError("returnCache: end")

	if ch.Etag != "" {
		writer.Header().Set(etagStr, ch.Etag)
	}
	writer.Header().Set(contentTypeStr, ch.ContentType)
	writer.Header().Set(contentLengthStr, fmt.Sprintf("%d", ch.ContentLength))
	if ch.LastModified != 0 {
		lastModifiedTmp := time.Unix(ch.LastModified, 0).UTC().Format(http.TimeFormat)
		writer.Header().Set(lastModifiedStr, lastModifiedTmp)
	}
	writer.Header().Set("connection", "close")
	if domType == FTL_OBSERVATION {
		writer.Header().Set("cache-control", fmt.Sprintf("max-age=%d", OBSERVATION_CACHE_MAX_AGE))
	}

	buf := make([]byte, 8192)
	wsum := 0
	rerrcount := 0
	werrcount := 0
	//logError("returnCache: ch.ContentLength = ", fmt.Sprintf("%d", ch.ContentLength))
	before := time.Now()
	writer.WriteHeader(http.StatusOK)
	for wsum != ch.ContentLength {
		nr, rerr := bodyFp.Read(buf)
		//logError("returnCache: Read: ", fmt.Sprintf("%d", nr))
		if rerr != nil {
			rerrcount++
			if rerrcount > RETURN_CACHE_TIMEOUT {
				logError("returnCache: rerrcount > RETURN_CACHE_TIMEOUT")
				return 0, 0
			}
			time.Sleep(100 * time.Millisecond)
		} else {
			rerrcount = 0
		}
		nwtmp := 0
		for nwtmp != nr {
			nw, werr := writer.Write(buf[nwtmp:nr])
			//logError("returnCache: Write: ", fmt.Sprintf("%d", nw))
			if werr != nil {
				werrcount++
				if werrcount > RETURN_CACHE_TIMEOUT {
					logError("returnCache: werrcount > RETURN_CACHE_TIMEOUT")
					return 0, 0
				}
				time.Sleep(100 * time.Millisecond)
			} else {
				werrcount = 0
			}
			nwtmp += nw
			wsum += nw
		}
	}
	if flusher, ok := writer.(http.Flusher); ok {
		//logError("returnCache: http.Flusher")
		flusher.Flush()
	}
	after := time.Now()
	return uint64(after.UnixNano()) - uint64(before.UnixNano()), uint64(ch.ContentLength)
}

func updateNetStats(start time.Time, elapsed, bytes uint64) {
	//logError("updateNetStats: begin")
	//defer logError("updateNetStats: end")
	var bestT int64
	var bestNetId NetId
	for tmpT, tmpNetId := range forwarded {
		if tmpT > start.UnixNano() || (start.UnixNano()-tmpT) > FORWARDED_EXPIRE {
			continue
		}
		if tmpT > bestT {
			bestT = tmpT
			bestNetId = tmpNetId
		}
	}
	//logError("updateNetStats: bestT = ", bestT)
	if bestT != 0 {
		net, ok := nets[bestNetId]
		if !ok {
			net = newNet()
			nets[bestNetId] = net
		}
		net.statsOut.updateStats(start, 1, elapsed, bytes)
	}
}

func ftlGetHandler(writer http.ResponseWriter, req *http.Request, start time.Time) (int, bool, uint64, uint64) {
	//logError("ftlGetHandler: begin")
	//defer logError("ftlGetHandler: end")

	var err error
	var chRemote *FtlCacheHeader
	var bodyFp *os.File
	var bodyFi os.FileInfo
	var currSize int64
	var elapsed, bytes uint64

	domType := domainNameType(req.Host)
	host := originDomainName(req.Host)
	uri := req.RequestURI

	resp, err := http.Head(fmt.Sprintf("http://%s%s", host, uri))
	if err != nil {
		logError("ftlGetHandler: http.Head:", host, uri)
		writer.WriteHeader(502)
		return 502, false, 0, 0
	}

	os.Mkdir(cacheFileDir(host), 0755)

	base := cacheFilePathBase(host, uri)
	header := fmt.Sprintf("%s.h", base)
	body := fmt.Sprintf("%s.b", base)

	chLocal := &FtlCacheHeader{}

	dataDirLock.Lock()
	err = chLocal.parseHeader(header)
	if err != nil {
		dataDirLock.Unlock()
		goto miss
	}
	chRemote = &FtlCacheHeader{}
	err = chRemote.parseResponse(resp)
	if err != nil || !chLocal.equal(chRemote) {
		dataDirLock.Unlock()
		goto miss
	}
	bodyFp, err = os.OpenFile(body, os.O_RDONLY, 0644)
	if err != nil {
		logError("ftlGetHandler: os.OpenFile:", body, host, uri)
		writer.WriteHeader(502)
		dataDirLock.Unlock()
		return 502, false, 0, 0
	}
	defer bodyFp.Close()
	bodyFi, err = bodyFp.Stat()
	if err != nil {
		logError("ftlGetHandler: bodyFp.Stat:", body, host, uri)
		writer.WriteHeader(502)
		dataDirLock.Unlock()
		return 502, false, 0, 0
	}
	currSize = bodyFi.Size()
	dataDirLock.Unlock()
	elapsed, bytes = returnCache(writer, bodyFp, chLocal, domType)
	if domType == FTL_OBSERVATION && currSize == int64(chRemote.ContentLength) &&
		elapsed > 0 && bytes > 0 {
		updateNetStats(start, elapsed, bytes)
	}
	return 200, true, bytes, elapsed
miss:

	statusChan := make(chan int)
	go getOriginAndWriteCache(host, uri, statusChan)
	status := <-statusChan
	if status != 200 {
		logError("ftlGetHandler: status != 200", host, uri)
		writer.WriteHeader(502)
		return 502, false, 0, 0
	}

	dataDirLock.Lock()
	err = chLocal.parseHeader(header)
	if err != nil {
		logError("ftlGetHandler: chLocal.parseHeader:", header, host, uri)
		writer.WriteHeader(502)
		dataDirLock.Unlock()
		return 502, false, 0, 0
	}
	bodyFp, err = os.OpenFile(body, os.O_RDONLY, 0644)
	if err != nil {
		logError("ftlGetHandler: os.OpenFile:", body, host, uri)
		writer.WriteHeader(502)
		dataDirLock.Unlock()
		return 502, false, 0, 0
	}
	defer bodyFp.Close()
	dataDirLock.Unlock()
	elapsed, bytes = returnCache(writer, bodyFp, chLocal, domType)
	return 200, false, bytes, elapsed
}

func ftlHttpHandler(writer http.ResponseWriter, req *http.Request) {
	//logError("ftlHttpHandler: begin")
	//defer logError("ftlHttpHandler: end")

	var status int
	var cacheHit bool
	var bytes, elapsed uint64

	t := time.Now()

	domainNameType := domainNameType(req.Host)
	if domainNameType == DISCARD {
		logError("ftlHttpHandler: domainNameType == DISCARD", req.Host)
		writer.WriteHeader(502)
		status = 502
		goto log
	}

	switch req.Method {
	case "GET":
		status, cacheHit, bytes, elapsed = ftlGetHandler(writer, req, t)
	case "HEAD":
		status = ftlHeadHandler(writer, req)
	default:
		writer.WriteHeader(502)
		status = 502
	}
log:
	logAccess(t,
		fmt.Sprint(NodeId2Addr4(myNodeId)),
		req.RemoteAddr,
		req.Method,
		fmt.Sprint(status),
		fmt.Sprint(cacheHit),
		req.Host,
		req.RequestURI,
		fmt.Sprint(bytes),
		fmt.Sprint(elapsed))
	return
}

func HTTPService() {
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", HTTP_PORT),
		Handler:        http.HandlerFunc(ftlHttpHandler),
		ReadTimeout:    6 * time.Hour,
		WriteTimeout:   6 * time.Hour,
		MaxHeaderBytes: 1 << 20,
	}
	server.ListenAndServe()
}
