package ipmgr

import ()

// initialized by caller
// or for tests    testInitializer(c)

// IP default link IP
var IP string

// Bits default link CIDR bits abbrev
var Bits string

// LinkDevice default link device to use for external ip addresses
var LinkDevice string

// DefaultCIDR default link device to use for external ip addresses
var DefaultCIDR *CIDR

// Debug set by initializer
var Debug bool
