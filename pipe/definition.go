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

package pipe

import (
	"sort"

	"github.com/davidwalter0/loadbalancer/tracer"
)

// EP slice of endpoints
type EP []string

// Equal compare two endpoint arrays for equality
func (ep *EP) Equal(rhs *EP) (rc bool) {
	if ep != nil && rhs != nil && len(*ep) == len(*rhs) {
		sort.Strings(*ep)
		sort.Strings(*rhs)
		for i, v := range *ep {
			if v != (*rhs)[i] {
				return
			}
		}
	} else {
		return
	}
	return true
}

// Definition maps source to sink
type Definition struct {
	Key       string `json:"key"       help:"map key (ns/name k8s services|yaml name for definition)"`
	Source    string `json:"source"    help:"source ingress point host:port"`
	Sink      string `json:"sink"      help:"sink service point   host:port"`
	Name      string `json:"service"   help:"service name"`
	Namespace string `json:"namespace" help:"service namespace"`
	Mode      string `json:"mode"      help:"mode of use for this service"`
	Debug     bool   `json:"debug"     help:"enable debug for this pipe"`
}

// NewFromDefinition create and initialize a Definition
func NewFromDefinition(pipe *Definition, inCluster bool) (p *Definition) {
	if pipe != nil {
		defer trace.Tracer.ScopedTrace()()
		p = &Definition{
			// Name is the key of yaml map
			Key:       pipe.Key,
			Source:    pipe.Source,
			Sink:      pipe.Sink,
			Name:      pipe.Name,
			Namespace: pipe.Namespace,
			Debug:     pipe.Debug,
			Mode:      pipe.Mode,
		}
	}
	return
}

// Definitions from text description in yaml
type Definitions map[string]*Definition

// Equal compares two pipe.Definition objects
func (lhs *Definition) Equal(rhs *Definition) bool {
	defer trace.Tracer.ScopedTrace()()
	return lhs.Key == rhs.Key &&
		lhs.Source == rhs.Source &&
		lhs.Sink == rhs.Sink &&
		lhs.Name == rhs.Name &&
		lhs.Namespace == rhs.Namespace &&
		lhs.Mode == rhs.Mode &&
		lhs.Debug == rhs.Debug
}

// Copy points w/o erasing EndPoints
func (lhs *Definition) Copy(rhs *Definition) *Definition {
	lhs.Key = rhs.Key
	lhs.Source = rhs.Source
	lhs.Sink = rhs.Sink
	lhs.Name = rhs.Name
	lhs.Namespace = rhs.Namespace
	lhs.Mode = rhs.Mode
	lhs.Debug = rhs.Debug
	return lhs
}
