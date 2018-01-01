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

---
## Example use


If you were running a service named `echo` running on port 8888 

```
# ------------------------- Service ------------------------- #
---
apiVersion: v1
kind: Service
metadata:
  name: echo
  labels:
    app: echo
spec:
  selector:
    app: echo
  ports:
  - port: 8888
    name: echo

```

Then updated it with a definition similar to the following `kubectl
apply -f service.yaml` to update that service, with LLB running
outside the cluster the accessible port will be a *NodePort*. That
NodePort will be the upstream *sink* add a new external port using the kubernetes inserted
NodePort value as the destination

```
# ------------------------- Service ------------------------- #
---
apiVersion: v1
kind: Service
metadata:
  name: echo
  labels:
    app: echo
spec:
  selector:
    app: echo
  ports:
  - port: 8888
    name: echo
  type: LoadBalancer
```
