// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/goklog/klog"

	"fmt"
	"io/ioutil"
	"runtime"
	"syscall"
	"time"
)

type process struct {
	Boot    *Counter
	Running *Gauge

	Load *Gauge

	Golang struct {
		Threads    *Gauge
		Goroutines *Gauge

		GcRuns      *Counter
		GcPauseTime *Gauge

		AllocationRate *Gauge

		HeapAlloc    *Gauge
		HeapSys      *Gauge
		HeapIdle     *Gauge
		HeapInuse    *Gauge
		HeapReleased *Gauge
		HeapObjects  *Gauge
	}

	Processor struct {
		UserTime   *Gauge
		SystemTime *Gauge

		ContextSwitchesV *Gauge
		ContextSwitchesI *Gauge
	}

	Memory struct {
		Resident *Gauge
		Virtual  *Gauge
		Shared   *Gauge

		MinorFaults *Gauge
		MajorFaults *Gauge
		Swaps       *Gauge
	}

	Block struct {
		InOps  *Gauge
		OutOps *Gauge
	}

	lastRusage   syscall.Rusage
	lastMemStats runtime.MemStats
}

// ProcessStats registers various process metrics from the OS and the go runtime
// under the given prefix.
func ProcessStats(prefix string) {
	meter := &process{}
	Load(meter, Join(prefix, "process"))

	go func() {
		meter.Boot.Hit()
		meter.Running.Change(1)
		meter.lastRusage = meter.rusage()
		runtime.ReadMemStats(&meter.lastMemStats)

		tickC := time.Tick(1 * time.Second)
		for {
			<-tickC
			meter.sampleGolang()
			meter.sampleRusage()
			meter.sampleStatm()
		}
	}()
}

func (meter *process) rusage() (result syscall.Rusage) {
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &result); err != nil {
		klog.KFatalf("meter.process.rusage.error", err.Error())
	}
	return
}

func (meter *process) sampleGolang() {
	meter.Golang.Goroutines.Change(float64(runtime.NumGoroutine()))
	meter.Golang.Threads.Change(float64(runtime.GOMAXPROCS(0)))

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	gcRuns := uint64(mem.NumGC - meter.lastMemStats.NumGC)
	meter.Golang.GcRuns.Count(gcRuns)

	gcPause := uint64(mem.PauseTotalNs - meter.lastMemStats.PauseTotalNs)
	if gcRuns > 0 {
		gcPause /= gcRuns
	}
	meter.Golang.GcPauseTime.Change(float64(gcPause) / float64(time.Second))

	meter.Golang.AllocationRate.Change(float64(mem.TotalAlloc - meter.lastMemStats.TotalAlloc))

	meter.Golang.HeapAlloc.Change(float64(mem.HeapAlloc))
	meter.Golang.HeapSys.Change(float64(mem.HeapSys))
	meter.Golang.HeapIdle.Change(float64(mem.HeapIdle))
	meter.Golang.HeapInuse.Change(float64(mem.HeapInuse))
	meter.Golang.HeapReleased.Change(float64(mem.HeapReleased))
	meter.Golang.HeapObjects.Change(float64(mem.HeapObjects))

	meter.lastMemStats = mem
}

func (meter *process) sampleRusage() {
	rusage := meter.rusage()

	utime := time.Duration(rusage.Utime.Nano() - meter.lastRusage.Utime.Nano())
	meter.Processor.UserTime.ChangeDuration(utime)

	stime := time.Duration(rusage.Stime.Nano() - meter.lastRusage.Stime.Nano())
	meter.Processor.SystemTime.ChangeDuration(stime)

	// Overestimates because the goruntime creates two background threads which
	// are not accounted by GOMAXPROCS. Better then nothing though.
	threads := runtime.GOMAXPROCS(0)
	meter.Load.Change(float64(utime+stime) / float64(time.Duration(threads)*time.Second))

	meter.Processor.ContextSwitchesI.Change(float64(rusage.Nvcsw - meter.lastRusage.Nvcsw))
	meter.Processor.ContextSwitchesV.Change(float64(rusage.Nivcsw - meter.lastRusage.Nivcsw))

	meter.Memory.MinorFaults.Change(float64(rusage.Minflt - meter.lastRusage.Minflt))
	meter.Memory.MajorFaults.Change(float64(rusage.Majflt - meter.lastRusage.Majflt))
	meter.Memory.Swaps.Change(float64(rusage.Nswap))

	meter.Block.InOps.Change(float64(rusage.Inblock - meter.lastRusage.Inblock))
	meter.Block.OutOps.Change(float64(rusage.Oublock - meter.lastRusage.Oublock))

	meter.lastRusage = rusage
}

func (meter *process) sampleStatm() {
	body, err := ioutil.ReadFile("/proc/self/statm")
	if err != nil {
		klog.KFatalf("meter.process.statm.error", err.Error())
	}

	var virt, rss, shared uint64
	if _, err := fmt.Sscanf(string(body), "%d %d %d", &virt, &rss, &shared); err != nil {
		klog.KFatalf("meter.process.statm.parse.error", err.Error())
	}

	meter.Memory.Resident.Change(float64(rss * 1000))
	meter.Memory.Virtual.Change(float64(virt * 1000))
	meter.Memory.Shared.Change(float64(shared * 1000))
}
