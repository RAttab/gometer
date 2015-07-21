// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/goklog/klog"

	"encoding/json"
	"log"
)

// KlogHandler is used to print the received meter values to klog.
type KlogHandler struct{}

// HandleMeters sends the given values to klog.
func (KlogHandler) HandleMeters(values map[string]float64) {
	body, err := json.Marshal(values)
	if err != nil {
		log.Panicf("unable to marshal meter values: %s", err)
	}

	klog.KPrint("meter.klog.info", string(body))
}
