package ftl

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	LOG_FORMAT = "2006-01-02 15:04:05.000000000 -0700 MST"
)

var logLock sync.Mutex

func getYmdStr(t time.Time) string {
	return fmt.Sprintf("%4d%02d%02d", t.Year(), int(t.Month()), t.Day())
}

func logAccess(t time.Time, strs ...string) {
	logLock.Lock()
	defer logLock.Unlock()

	accessLogPath := fmt.Sprintf("%s/access-%s.log", logDir, getYmdStr(t.UTC()))
	accessLog, err := os.OpenFile(accessLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer accessLog.Close()

	fmt.Fprintln(accessLog,
		t.Format(LOG_FORMAT),
		strs[0],
		strs[1],
		strs[2],
		strs[3],
		strs[4],
		fmt.Sprintf("\"%s\"", strs[5]),
		fmt.Sprintf("\"%s\"", strs[6]),
		strs[7],
		strs[8])
}

func logError(strs ...string) {
	logLock.Lock()
	defer logLock.Unlock()

	errorLogPath := fmt.Sprintf("%s/error-%s.log", logDir, getYmdStr(time.Now().UTC()))
	errorLog, err := os.OpenFile(errorLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer errorLog.Close()

	fmt.Fprintln(errorLog, time.Now().Format(LOG_FORMAT), strs)
}
