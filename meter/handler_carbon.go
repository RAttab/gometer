// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/datacratic/goklog/klog"

	"bufio"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	CarbonDialTimeout  = 1 * time.Second
	CarbonMaxConnDelay = 1 * time.Minute
)

type msgConn struct {
	URL  string
	Conn net.Conn
}

type CarbonHandler struct {
	URLs []string

	initialize sync.Once

	conns   map[string]net.Conn
	connC   chan msgConn
	valuesC chan map[string]float64
}

func (carbon *CarbonHandler) Init() {
	carbon.initialize.Do(carbon.init)
}

func (carbon *CarbonHandler) HandleMeters(values map[string]float64) {
	carbon.Init()
	carbon.valuesC <- values
}

func (carbon *CarbonHandler) init() {
	if len(carbon.URLs) == 0 {
		klog.KFatal("meter.carbon.init.error", "no URL configured")
	}

	carbon.conns = make(map[string]net.Conn)
	for _, URL := range carbon.URLs {
		carbon.connect(URL)
	}

	carbon.connC = make(chan msgConn)
	carbon.valuesC = make(chan map[string]float64)

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
		klog.KPrintf("meter.carbon.send.debug", "%s %f %d", key, value, ts)

		if _, err = fmt.Fprintf(writer, "%s %f %d\n", key, value, ts); err != nil {
			return
		}
	}

	err = writer.Flush()
	return
}