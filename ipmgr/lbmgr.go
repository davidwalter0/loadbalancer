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

package ipmgr

import ()

// GetEtherIfaceFromIP returns the link device name for which an
// address would be assigned
func (c *CIDR) GetEtherIfaceFromIP() string {
	for _, link := range LinkNames() {
		if addr := LinkDefaultAddr(link); addr != nil {
			if c.MatchAddr(addr) {
				return link
			}
		}
	}
	return ""
}

// SetEtherIfaceFromIP returns the link device name for which an
// address would be assigned
func (c *CIDR) SetEtherIfaceFromIP() string {
	for _, link := range LinkNames() {
		if addr := LinkDefaultAddr(link); addr != nil {
			if c.MatchAddr(addr) {
				c.LinkDevice = link
				return link
			}
		}
	}
	return ""
}
