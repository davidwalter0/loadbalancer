---
Manage routes / addresses for external ip addresses

- add the option to add ip addresses from the load balancer service
  definition
  - add if not present
  - maintain a map of addresses
  - remove when the last load balancer using the address is removed
