/*

Copyright 2018-2025 David Walter.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package mgr

import (
	"github.com/davidwalter0/loadbalancer/tracer"
)

// Equal compares two ManagedListener objects
func (lhs *ManagedListener) Equal(rhs *ManagedListener) bool {
	defer trace.Tracer.ScopedTrace()()
	return lhs.Key == rhs.Key &&
		lhs.Source == rhs.Source &&
		lhs.Sink == rhs.Sink &&
		lhs.Name == rhs.Name &&
		lhs.Namespace == rhs.Namespace &&
		lhs.Mode == rhs.Mode &&
		lhs.Debug == rhs.Debug &&
		lhs.Ports.Equal(rhs.Ports) &&
		lhs.IPs.Equal(rhs.IPs)
}

// Copy points w/o erasing EndPoints
func (lhs *ManagedListener) Copy(rhs *ManagedListener) *ManagedListener {
	lhs.Key = rhs.Key
	lhs.Source = rhs.Source
	lhs.Sink = rhs.Sink
	lhs.Name = rhs.Name
	lhs.Namespace = rhs.Namespace
	lhs.Debug = rhs.Debug
	lhs.Mode = rhs.Mode
	lhs.Active = rhs.Active
	lhs.Ports = make(Ports, len(rhs.Ports))
	copy(lhs.Ports, rhs.Ports)
	lhs.IPs = make(IPs, len(rhs.IPs))
	copy(lhs.IPs, rhs.IPs)
	return lhs
}
