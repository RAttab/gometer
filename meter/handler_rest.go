// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/gorest/rest"

	"strings"
	"sync"
)

// DefaultPathREST is used as the value of PathPrefix in RESTHandler if no
// values are provided.
const DefaultPathREST = "/debug/meter"

// RESTHandler provides a REST interface for the values polled from the meters.
type RESTHandler struct {

	// PathPrefix is the HTTP base path where the REST interface will be
	// registered at.
	PathPrefix string

	mutex sync.Mutex
	last  map[string]float64
}

// NewRESTHandler creates a new REST interface and registers it with the default
// gorest Mux.
func NewRESTHandler(path string) *RESTHandler {
	handler := &RESTHandler{PathPrefix: path}
	rest.AddService(handler)
	return handler
}

// RESTRoutes returns the list of REST routes.
func (handler *RESTHandler) RESTRoutes() rest.Routes {
	prefix := handler.PathPrefix
	if prefix == "" {
		prefix = DefaultPathREST + "/keys"
	}

	return []*rest.Route{
		rest.NewRoute(prefix, "GET", handler.Get),
		rest.NewRoute(prefix+"/prefix/:substr", "GET", handler.GetPrefix),
		rest.NewRoute(prefix+"/substr/:substr", "GET", handler.GetSubstr),
		rest.NewRoute(prefix+"/pattern/:pattern", "GET", handler.GetPattern),
	}
}

// HandleMeters records the aggregated metrics to be used by the REST interface.
func (handler *RESTHandler) HandleMeters(meters map[string]float64) {
	handler.mutex.Lock()

	handler.last = meters

	handler.mutex.Unlock()
}

// Get returns the last seen set of metrics.
func (handler *RESTHandler) Get() map[string]float64 {
	handler.mutex.Lock()

	result := handler.last

	handler.mutex.Unlock()

	return result
}

// GetPrefix returns the set of metrics filtered to have the given prefix.
func (handler *RESTHandler) GetPrefix(prefix string) map[string]float64 {
	return handler.get(func(key string) bool {
		return strings.HasPrefix(key, prefix)
	})
}

// GetSubstr returns the set of metrics filtered to contain the given substring.
func (handler *RESTHandler) GetSubstr(substr string) map[string]float64 {
	return handler.get(func(key string) bool {
		return strings.Index(key, substr) >= 0
	})
}

// GetPattern returns the set of metrics filtered to contain the given pattern.
func (handler *RESTHandler) GetPattern(pattern string) map[string]float64 {
	p := NewPattern(pattern)
	return handler.get(func(key string) bool {
		_, ok := p.Match(key)
		return ok
	})
}

func (handler *RESTHandler) get(filter func(string) bool) map[string]float64 {
	handler.mutex.Lock()

	result := make(map[string]float64)

	for key, value := range handler.last {
		if filter(key) {
			result[key] = value
		}
	}

	handler.mutex.Unlock()

	return result
}
