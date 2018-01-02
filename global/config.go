package global

import (
	"github.com/davidwalter0/llb/share"
)

var envCfg = &share.ServerCfg{}

func init() {
	// Load the configuration
	envCfg.Read()
}

// Cfg exposes common configuration item
func Cfg() *share.ServerCfg {
	return envCfg
}
