package esi

import "sync/atomic"

// counter is an atomic tally shared between test goroutines and the assertions
// that read it.
type counter struct{ n atomic.Int64 }

func (c *counter) inc()     { c.n.Add(1) }
func (c *counter) get() int { return int(c.n.Load()) }
