// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"sync"
	"testing"
	"time"
)

type TestHandler struct {
	T *testing.T

	initialize sync.Once
	valuesC    chan map[string]float64
}

func (handler *TestHandler) Init() {
	handler.initialize.Do(handler.init)
}

func (handler *TestHandler) init() {
	handler.valuesC = make(chan map[string]float64, 1<<8)
}

func (handler *TestHandler) HandleMeters(values map[string]float64) {
	handler.Init()
	handler.valuesC <- values
}

func (handler *TestHandler) Get() map[string]float64 {
	handler.Init()

	timeoutC := time.After(101 * time.Millisecond)

	select {
	case values := <-handler.valuesC:
		return values

	case <-timeoutC:
		return nil
	}
}

func (handler *TestHandler) Expect(title string, exp map[string]float64) {
	CheckValues(handler.T, title, handler.Get(), exp)
}
