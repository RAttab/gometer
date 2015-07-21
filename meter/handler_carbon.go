// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/goklog/klog"

	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	// CarbonDialTimeout is the timeout value when dialing the remote carbon
	// host.
	CarbonDialTimeout = 1 * time.Second

	// CarbonMaxConnDelay is the maximum value of the exponential backoff scheme
	// when reconecting to a carbon host.
	CarbonMaxConnDelay = 1 * time.Minute
)

// CarbonHandler forwards a set of recorded meter values to multiple carbon
// hosts. Any values received while the connection to the carbon host is
// unavailable are dropped.
type CarbonHandler struct {

	// URLs contains the list of carbon hosts to connect to.
	URLs []string

	initialize sync.Once

	conns   map[string]net.Conn
	connC   chan msgConn
	valuesC chan map[string]float64
}

// NewCarbonHandler instantiates a new CarbonHandler which will log to the given
// URL.
func NewCarbonHandler(URL string) *CarbonHandler {
	return &CarbonHandler{URLs: []string{URL}}
}

// Init can be optionally used to initialize the object. Note that the handler
// will lazily initialize itself as needed.
func (carbon *CarbonHandler) Init() {
	carbon.initialize.Do(carbon.init)
}

// HandleMeters forwards the given values to all the carbon host with a valid
// connection.
func (carbon *CarbonHandler) HandleMeters(values map[string]float64) {
	carbon.Init()
	carbon.valuesC <- values
}

type msgConn struct {
	URL  string
	Conn net.Conn
}

func (carbon *CarbonHandler) init() {
	if len(carbon.URLs) == 0 {
		klog.KFatal("meter.carbon.init.error", "no URL configured")
	}

	carbon.connC = make(chan msgConn)
	carbon.valuesC = make(chan map[string]float64)

	carbon.conns = make(map[string]net.Conn)
	for _, URL := range carbon.URLs {
		carbon.connect(URL)
	}

	go carbon.run()
}

func (carbon *CarbonHandler) run() {
	for {
		select {
		case values := <-carbon.valuesC:
			carbon.send(values)

		case msg := <-carbon.connC:
			carbon.conns[msg.URL] = msg.Conn
		}
	}
}

func (carbon *CarbonHandler) connect(URL string) {

	if conn := carbon.conns[URL]; conn != nil {
		conn.Close()
	}
	carbon.conns[URL] = nil

	go carbon.dial(URL)
}

func (carbon *CarbonHandler) dial(URL string) {
	for attempts := 0; ; attempts++ {
		carbon.sleep(attempts)

		conn, err := net.DialTimeout("tcp", URL, CarbonDialTimeout)
		if err == nil {
			klog.KPrintf("meter.carbon.dial.info", "connected to '%s'", URL)
			carbon.connC <- msgConn{URL, conn}
			return
		}

		klog.KPrintf("meter.carbon.dial.error", "unable to connect to '%s': %s", URL, err)
	}
}

func (carbon *CarbonHandler) sleep(attempts int) {
	if attempts == 0 {
		return
	}

	sleepFor := time.Duration(attempts*2) * time.Second

	if sleepFor < CarbonMaxConnDelay {
		time.Sleep(sleepFor)
	} else {
		time.Sleep(CarbonMaxConnDelay)
	}
}

func (carbon *CarbonHandler) send(values map[string]float64) {
	ts := time.Now().Unix()

	for URL, conn := range carbon.conns {
		if conn == nil {
			continue
		}

		if err := carbon.write(conn, values, ts); err != nil {
			klog.KPrintf("meter.carbon.send.error", "error when sending to '%s': %s", URL, err)
			carbon.connect(URL)
		}
	}
}

func (carbon *CarbonHandler) write(conn net.Conn, values map[string]float64, ts int64) (err error) {
	writer := bufio.NewWriter(conn)

	for key, value := range values {
		if _, err = fmt.Fprintf(writer, "%s %f %d\n", key, value, ts); err != nil {
			return
		}
	}

	err = writer.Flush()
	return
}
