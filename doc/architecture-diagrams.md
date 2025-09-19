# Architecture Diagrams

This document contains architecture diagrams for the Kubernetes External LoadBalancer.

## System Components

```mermaid
classDiagram
    class Mgr {
        +Listeners map[string]*ManagedListener
        +NodeWatcher *QueueMgr
        +ServiceWatcher *QueueMgr
        +EndpointWatcher *QueueMgr
        +Run()
        +NodeWatch()
        +ServiceWatch()
        +EndpointWatch()
        +Listen(Service)
        +Shutdown()
    }
    
    class ManagedListener {
        +Definition pipe.Definition
        +Listener net.Listener
        +Pipes map[*pipe.Pipe]bool
        +Service *v1.Service
        +Endpoints *v1.Endpoints
        +CIDR *ipmgr.CIDR
        +Open()
        +Listening()
        +SetExternalIP()
        +RemoveExternalIP()
        +Close()
    }
    
    class Pipe {
        +SourceConn net.Conn
        +SinkConn net.Conn
        +Connect()
        +Close()
    }
    
    class IPManager {
        +AddAddr(IPNet, LinkDevice)
        +RemoveAddr(IPNet, LinkDevice)
    }
    
    class QueueMgr {
        +QueueItems chan QueueItem
        +Run(threadiness, sleepSeconds)
    }
    
    class Definition {
        +Key string
        +Source string
        +Sink string
        +Name string
        +Namespace string
    }
    
    class CIDR {
        +IP string
        +Bits string
        +LinkDevice string
        +String()
    }
    
    Mgr "1" -- "many" ManagedListener : manages
    ManagedListener "1" -- "many" Pipe : manages
    ManagedListener "1" -- "1" Definition : has
    ManagedListener "1" -- "1" CIDR : has
    Mgr "1" -- "3" QueueMgr : has
    IPManager -- CIDR : uses
```

## Operational Flow

```mermaid
sequenceDiagram
    participant KubeAPI as Kubernetes API
    participant Mgr as Manager
    participant ML as ManagedListener
    participant IP as IPManager
    participant Client
    participant Backend as Service Endpoint
    
    KubeAPI->>Mgr: Service Create Event
    Mgr->>ML: Create ManagedListener
    ML->>IP: Add External IP
    ML->>ML: Create TCP Listener
    ML->>KubeAPI: Update Service externalIPs
    
    Client->>ML: TCP Connection
    ML->>Backend: Forward Connection
    Backend->>ML: Response
    ML->>Client: Forward Response
    
    KubeAPI->>Mgr: Endpoint Update Event
    Mgr->>ML: Update Endpoints
    
    KubeAPI->>Mgr: Service Delete Event
    Mgr->>ML: Close ManagedListener
    ML->>ML: Close Connections
    ML->>IP: Remove External IP
    ML->>KubeAPI: Clear Service externalIPs
```

## Network Architecture

```mermaid
graph LR
    Client((Client)) -- TCP --> ExtIP[External IP on Network Interface]
    ExtIP -- Forward --> LB[LoadBalancer]
    LB -- Round Robin --> EP1[Endpoint 1]
    LB -- Round Robin --> EP2[Endpoint 2]
    LB -- Round Robin --> EP3[Endpoint 3]
    
    subgraph Kubernetes Cluster
        LB
        EP1
        EP2
        EP3
    end
```

## Component Interactions

```mermaid
graph TD
    A[main.go] --> B[Mgr]
    B --> C[ServiceWatcher]
    B --> D[EndpointWatcher]
    B --> E[NodeWatcher]
    
    C --> F[ManagedListener Creation]
    D --> G[Endpoint Updates]
    
    F --> H[Listener Opening]
    F --> I[IP Assignment]
    
    H --> J[Connection Handling]
    J --> K[Pipe Creation]
    
    K --> L[Data Transfer]
```

## Data Flow

```mermaid
flowchart LR
    A[Client Request] --> B{LoadBalancer}
    B --> C[Round Robin Selection]
    C --> D[Create Pipe]
    D --> E[Forward to Endpoint]
    E --> F[Endpoint Processing]
    F --> G[Response]
    G --> D
    D --> A
```

## Initialization Sequence

```mermaid
sequenceDiagram
    participant Main
    participant Kubeconfig
    participant Mgr
    participant IPMgr
    participant Watchers
    
    Main->>Kubeconfig: NewClientset()
    Kubeconfig-->>Main: clientset
    Main->>Mgr: NewMgr(config, clientset)
    Mgr->>IPMgr: Initialize IP Management
    Mgr->>Watchers: Create Watchers
    Main->>Mgr: Run()
    Mgr->>Watchers: Start Watching
    Watchers-->>Mgr: Service Event
    Mgr->>Mgr: Handle Event
```

## State Transitions

```mermaid
stateDiagram-v2
    [*] --> Running
    Running --> [*]: Shutdown
    
    state Running {
        [*] --> Watching
        Watching --> CreatingListener: Service Event
        CreatingListener --> ManagingPipes: Listener Created
        ManagingPipes --> UpdatingEndpoints: Endpoint Event
        UpdatingEndpoints --> ManagingPipes
        ManagingPipes --> ClosingListener: Service Deleted
        ClosingListener --> Watching
    }
```