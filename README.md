*Use at your own risk Alpha software / pre-release*

---
Little load balancer

- Connect to kubernetes cluster
- Watch services
  - when the Type=LoadBalancer 
    - load endpoints for the service name/namespace
    - create a forward service listening on loadbalancer IP + port
    - accept new connections
    - create a "pipe" bidirectional copy to endpoint|from source
  - when the key is deleted or the type is changed from loadbalancer
    delete the forward service
  - when loadBalancerIP is set e.g. if the ip hasn't been set it will
    be added to the ethernet device specified as the LinkDevice
    e.g. `--linkdevice eth0`
- Watch nodes
  - add or remove nodes from events in the queue
  - use nodes with the label `node-role.kubernetes.io/worker`
  - during node creation or with the label command add 
    --node-labels=node-role.kubernetes.io/worker
  - use the ExternalID from the node spec as the IP endpoint


Manage routes / addresses for external ip addresses

- Add or remove ip addresses from the load balancer service definition
  - add if not present
  - maintain a map of addresses
  - remove when the last load balancer using the address is removed

---
## Example use


Run llb with superuser permissions so that llb can modify routes and
use privileged ports.

```
sudo bin/llb --kubeconfig cluster/auth/kubeconfig --linkdevice eth0
```

Run an echo service on port 8888

```
kubectl apply -f https://raw.githubusercontent.com/davidwalter0/echo/master/daemonset.yaml
```

Then create and modify the services like the following

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

Then update  it with  a definition similar  to the  following `kubectl
apply  -f  service.yaml` to  update  that  service, with  LLB  running
outside  the  cluster the  accessible  port  will  be a  *Port*.  That
NodePort will be the upstream *sink* add a new external port using the
kubernetes inserted NodePort value as the destination

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

Now you can `curl loadbalancerIP:8888` where loadbalancerIP is the
host the loadbalancer is running on.

---

IPs will be added when needed and ports assigned based on the
service port. IPs will be added on the specified LinkDevice (ethernet
device for external routes).  A service description with an IP address
adds the ip to the LinkDevice

```
# ------------------------- Service ------------------------- #
---
apiVersion: v1
kind: Service
metadata:
  name: echo5
  labels:
    app: echo
spec:
  selector:
    app: echo
  ports:
  - port: 8888
    name: echo
  loadBalancerIP: 192.168.0.226
  type: LoadBalancer
```

Now you can `curl loadbalancerIP:8888` where loadbalancerIP is the
host the loadbalancer is running on.

The ip management is similar to 

The ip command `ip addr add ip/bits dev linkdevice` `ip addr add
192.168.0.226/24 dev linkdevice`, but derives the CIDR mask bits from
the existing route information on the specified link device.

The reciprocal removal uses the existing CIDR definition when there
are no more listeners on the ip.

`ip addr add ip/bits dev linkdevice`

---
*Dashboard*

Another example enabling a routable dashboard assuming you've already
created the certificates for the dashboard

```
kubectl create secret generic kubernetes-dashboard-certs --from-file=cluster/tls --namespace=kube-system
kubectl apply -f examples/kubernetes-dashboard.yaml
kubectl apply -f examples/dashboard.yaml
```

---
*BUGS*

For testing the ip addresses of 2 VMs are hard coded. The next
iteration will test and extract the nodes from the cluster based on
node labels

---

*TODO*

- IP address endpoint assignment by collecting node names from
  kubernetes cluster
- Test InCluster endpoint activity  
### TODO

More things not yet completed

- [ ]  load active devices
- [x]  load active primary ip address per device
  - must specify the device on the command line --linkdevice

- [ ]  set default ip address per device
- [ ]  check for new load balancer request's ip match to a device default
    subnet and add if not found
- [ ]  catch/recover from errors associated with missing IP, illegal
    IP/CIDR, address in use and report accordingly
- [ ]  get endpoint node list by service
- [ ]  create endpoint watcher similar to service watch
- [ ]  check ability to run in cluster with host net/privileges
- [ ]  allow per namespace load balancer?
- [ ]  allow multiple ports per service to be forwarded

- [ ] research network route/device changes for both insertion of
    physical hardware or address changes
