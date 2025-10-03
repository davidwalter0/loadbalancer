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
