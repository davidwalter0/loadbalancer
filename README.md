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

- Disallow reuse of service specified loadbalancerIPs
  - when adding an ip address, it will be used only by one load
    balancers service
  - Each service must choose a unique address
  - if using the default address multiple services may share the
    default linkdevice address but port collision management is up to
    the author of the service specification

---
## Example use


Run loadbalancer with superuser permissions so that loadbalancer can modify routes and
use privileged ports.

```
sudo bin/loadbalancer --kubeconfig cluster/auth/kubeconfig --linkdevice eth0
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
apply  -f  service.yaml` to  update  that  service, with  LOADBALANCER  running
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
*List*

List services and their type

```
printf "$(kubectl get svc --all-namespaces --output=go-template --template='{{range .items}}{{.metadata.namespace}}/{{.metadata.name}}:{{.spec.type}} LB:{{ .spec.loadBalancerIP }} ExternalIPs{{.spec.externalIPs}}\n{{end}}')"
```

Service addresses for load balancers

```
printf "$(kubectl get svc --all-namespaces --output=go-template --template='{{range .items}}{{if eq .spec.type "LoadBalancer"}}{{.metadata.namespace}}/{{.metadata.name}}:{{.spec.type}} LB:{{ .spec.loadBalancerIP }} ExternalIPs{{.spec.externalIPs}}\n{{end}}{{end}}')"
```

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

- Unique IP assignment fails
  - if 2 services attempt to use the same address log the second as an
    error and ignore it.
  - current this causes a panic
  - disallow empty/missing loadBalancerIP or reuse of default bridge
    ip address

---

*TODO*

### Features / behaviour

Moved to complete and testing

- [x] Load active devices (use --linkdevice to specify the active
  device)
- [x] Load active primary ip address per device
  - must specify the device on the command line --linkdevice
- [x]  set default ip address per device
- [x] Check for new load balancer request's ip match to a device
  default subnet and add if not found
- [x] Catch/recover from errors associated with missing IP, illegal
  IP/CIDR, address in use and report accordingly
  - check valid ip address ignore if invalid
- [x] Get endpoint node list by service
  - marry nodes to nodeports as service endpoints for out of cluster
- [x] Create endpoint watcher similar to service watch
  - out of cluster use node watcher
- [x] All namespaces through one load balancer
- [x] Update service ExternalIPs with the ip address of the load balancer
- [x] Add signal handler to cleanup ExternalIPs on shutown sigint,
  sigterm
- [x] Run in a managed Kubernetes managed deployment pod inside cluster
- [x] IP address endpoint assignment by collecting node names from
  kubernetes cluster
  - [x] Complete
- [x] Test InCluster endpoint activity
  - [ ] In progress


--- 

*Possible Future Work*

- [ ] research netlink network route/device watcher for both insertion
  of physical hardware or default address change
- [ ] allow multiple ports per service to be forwarded


---

loadbalancer/examples/yaml:

Ensure that the loadBalancerIP addresses that you use are in the
subnet of the device specified for your subnet and not reserved, or if
using a home router, outside the range the router will offer to
devices on the network

Many of the simple examples are based on the echo service

```
kubectl apply -f examples/manifests/echodaemonset.yaml
```

-  kubernetes-dashboard-lb.yaml
-  kubernetes-dashboard.yaml
-  service-lb-new-addr.yaml
   - load balancer with a specified address loadBalancerIP= 
   - 
-  service-lb.yaml
   - load balancer without a specified address
-  service.yaml

If you run these in a locally configured VM with a bridged interface
the dynamically allocated ip addresses are visible to the external
network while isolating network changes from the host machine in the
VM.

- Running in a managed Kubernetes deployment pod inside the cluster
  - Manage ip addresses on linkdevice
  - Add address to and remove address from the linkdevice and use the
    address specified in the service's loadBalancerIP field as the
    service's externalIP
  - Example files: enable cluster role and configure deployment
    - kubectl -f examples/yaml/loadbalancerdeployment.yaml -f examples/yaml/loadbalancerclusterrole.yaml
    - loadbalancerdeployment.yaml
    - loadbalancerclusterrole.yaml 
  - Run inside a manually configured bridge in virtualbox or a
    bridged interface with vagrant
    - `https://www.vagrantup.com/docs/networking/public_network.html`
    - in Vagrant you can select the interface to use as the bridge and
      add the bridge when provisioning the VM
      - config.vm.network :public_network, :public_network => "wlan0"
      - config.vm.network :public_network, :public_network => "eth0"
      - answer the prompt with the bridge interface number
  - Run in cluster with host network privilege inside a kubernetes
    managed pod and a bridge interface specified as --linkdevice
    - label the node `node-role.kubernetes.io/load-balancer="primary"`
    - run a deployment or a replication set with a replica count of one
      replicas: 1
    - use the bridge interface device to apply the changes
    - configure permissions if the cluster has enabled
    - loadbalancer configures ips on the bridged interface supplied on the
      commandline


---
Configuring manifests and nodes for scheduling affinity / anti affinity

(bootkube ... multi-mode filesystem configuration reference)

Modify calico.yaml and kube-proxy.yaml in cluster/manifests

```
      tolerations:
        # Allow the pod to run on master nodes
        - key: node-role.kubernetes.io/master
          effect: NoSchedule
        # Allow the pod to run on loadbalancer nodes
        - key: node-role.kubernetes.io/loadbalancer
          effect: NoSchedule
```

Force scheduling load balancer only on
node-role.kubernetes.io/loadbalancer labeled node and allow scheduling
with toleration

```
      tolerations:
        - key: node-role.kubernetes.io/loadbalancer
          operator: Exists
          effect: NoSchedule
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/loadbalancer
                operator: Exists

```


Taint the load balancer node to repel (give scheduling anti affinity to all but those pods with manifests)

Label the node for scheduling affinity, taint for general anti affinity



```
    .
    .
    .
          --node-labels=node-role.kubernetes.io/loadbalancer=primary \
          --register-with-taints=node-role.kubernetes.io/loadbalancer=:NoSchedule \
    .
    .

```
