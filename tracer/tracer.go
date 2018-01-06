package trace

import (
	"github.com/davidwalter0/go-tracer"
)

// Tracer trace object
var Tracer = tracer.New()

func init() {
	Tracer.Enable(false)
}

// Enable turn on debug printing
func Enable() {
	Tracer.Enable(true)
}

// Detailed manage trace output level
func Detailed() {
	Tracer.Enable(true).Detailed(true)
}
