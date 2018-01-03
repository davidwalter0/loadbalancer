package trace

import (
	"github.com/davidwalter0/go-tracer"
	"github.com/davidwalter0/llb/global"
)

var Tracer = tracer.New()
var Detail = false
var Enabled = global.Cfg().Debug

func init() {
	Tracer.Detailed(Detail).Enable(Enabled)
}
