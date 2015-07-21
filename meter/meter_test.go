// Copyright (c) 2014 Datacratic. All rights reserved.

package meter

import (
	"github.com/RAttab/goset"

	"testing"
)

func CheckValues(t *testing.T, title string, values map[string]float64, exp map[string]float64) {
	if exp == nil {
		if values != nil {
			t.Errorf("FAIL(%s): expected 'nil' got '%v'", title, values)
		}
		return

	} else if values == nil {
		t.Errorf("FAIL(%s): expected '%v' got 'nil'", title, exp)
		return
	}

	a := set.NewString()
	b := set.NewString()

	for key := range values {
		a.Put(key)
	}

	for key := range exp {
		b.Put(key)
	}

	if diff := a.Difference(b); len(diff) > 0 {
		t.Errorf("FAIL(%s): extra keys %s", title, diff)
	}

	if diff := b.Difference(a); len(diff) > 0 {
		t.Errorf("FAIL(%s): missing keys %s", title, diff)
	}

	for key, value := range values {
		if exp[key] != value {
			t.Errorf("FAIL(%s): value mismatch for key '%s' -> %f != %f", title, key, value, exp[key])
		}
	}
}
