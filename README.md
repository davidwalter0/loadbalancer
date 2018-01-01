---
Little load balancer

- connect to kubernetes cluster
- watch services
  - when the Type=LoadBalancer 
    - load endpoints for the service name/namespace
    - create a forward service listening on loadbalancer IP + port
    - accept new connections
    - create a "pipe" bidirectional copy to endpoint|from source
  - when the key is deleted or the type is changed from loadbalancer
    delete the forward service
