// Copyright (c) 2014 Datacratic. All rights reserved.

package meter_test

import (
	"github.com/RAttab/gometer/meter"

	"fmt"
	"sort"
	"time"
)

// Meters will usually be associated with a service or a component
type MyComponent struct {

	// Meters are crawled via the meter.Load function where the name of the
	// fields will be used as the name of the key for the meter.
	metrics struct {
		Gauge     *meter.Gauge
		State     *meter.State
		Counter   *meter.Counter
		Histogram *meter.Histogram

		// Nested structs are also supported and will lead to nested meters.
		Multi struct {
			Counter   *meter.MultiCounter
			Histogram *meter.MultiHistogram
			Gauge     *meter.MultiGauge
		}
	}
}

func (component *MyComponent) Init() {

	// Initializing and registering the various meters is accomplished via the
	// meter.Load function which will crawl the given object to fill and register
	// the various meters.
	meter.Load(&component.metrics, "myComponent")
}

func (component *MyComponent) Exec() {
	// The Counter is used to count the number of events that occured within a
	// second. Pretty straightforward.
	component.metrics.Counter.Hit()
	component.metrics.Counter.Count(10)

	// A MultiCounter can be used to seperate your counts into buckets
	// determined at runtime. Here we're indicating that we've seen one success
	// and one error.
	component.metrics.Multi.Counter.Hit("success")
	component.metrics.Multi.Counter.Hit("error")

	// A Gauge meter will output a given value until the value is
	// changed. Useful to record events that only happen occasionally and are
	// therefor not a good fit for histograms.
	component.metrics.Gauge.Change(5)

	// A State is similar to a gauge except that it will output the value 1 for
	// a given value.
	component.metrics.State.Change("happy")

	// Histograms are used to output the distribution of events that occured
	// within a second.
	for i := 0; i < 100; i++ {
		component.metrics.Histogram.Record(float64(i))
	}
}

func ExampleMeter() {

	// It's also to monitor various process metrics via the ProcessStats
	// function (disabled to keep the example's output simple).
	// meter.ProcessStats("")

	// Before we start polling we need to decide where our metrics will go. We
	// can either use one of the builtin handlers (eg. CarbonHandler,
	// RESTHandler, etc.)  or, for the sake of the example, we can create our
	// own which will log to a channel.
	resultC := make(chan map[string]float64)
	handler := func(values map[string]float64) { resultC <- values }
	meter.Handle(meter.HandlerFunc(handler))

	// Meter polling must be initiated via the Poll function.
	meter.Poll("myProcess", 1*time.Second)

	// Finally, let's instantiate our component and start logging some metrics.
	var component MyComponent
	component.Init()
	component.Exec()

	// Finally, we'll finish off the test by reading the value and printing them
	// out.
	SortAndPrint(<-resultC)

	// Output:
	// myProcess.myComponent.Counter: 11.000000
	// myProcess.myComponent.Gauge: 5.000000
	// myProcess.myComponent.Histogram.avg: 49.500000
	// myProcess.myComponent.Histogram.count: 100.000000
	// myProcess.myComponent.Histogram.max: 99.000000
	// myProcess.myComponent.Histogram.min: 0.000000
	// myProcess.myComponent.Histogram.p50: 50.000000
	// myProcess.myComponent.Histogram.p90: 90.000000
	// myProcess.myComponent.Histogram.p99: 99.000000
	// myProcess.myComponent.Multi.Counter.error: 1.000000
	// myProcess.myComponent.Multi.Counter.success: 1.000000
	// myProcess.myComponent.State.happy: 1.000000
}

// SortAndPrint prints the map in a deterministic manner such that we can
// reliably check the output of our example. This is strictly boilerplate for
// the purpose of the example and is not required in actual code.
func SortAndPrint(values map[string]float64) {
	var keys []string

	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf("%s: %f\n", key, values[key])
	}

}
