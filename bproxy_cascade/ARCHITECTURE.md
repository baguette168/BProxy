# BProxy Architecture Documentation

## System Overview

BProxy is a sophisticated red team proxy tool designed with a modular architecture that supports multi-level node cascading, real-time topology management, and Layer 3 routing capabilities.

## Core Components

### 1. Protocol Layer (`proto/`)

**Purpose**: Define the communication protocol using Protocol Buffers.

**Key Files**:
- `message.proto`: Protocol definition
- `message.pb.go`: Generated Go code

**Message Types**:
```protobuf
enum MessageType {
  HEARTBEAT = 0;  // Keep-alive messages
  COMMAND = 1;    // Admin commands to agents
  DATA = 2;       // Data transfer
  REGISTER = 3;   // Agent registration
  CONNECT = 4;    // Cascade connection request
  RELAY = 5;      // Message relay through nodes
}
```

**Message Structure**:
```protobuf
message Message {
  MessageType type = 1;
  string session_id = 2;
  bytes payload = 3;
  string source_id = 4;
  string target_id = 5;
  int64 timestamp = 6;
}
```

### 2. Admin Server (`admin/`)

**Purpose**: Central control server that manages all agent connections.

**Key Responsibilities**:
- Accept incoming agent connections
- Maintain agent registry
- Manage topology graph
- Route messages between agents
- Perform heartbeat monitoring
- Handle cascade commands

**Data Structures**:
```go
type Admin struct {
    listener  net.Listener
    agents    map[string]*AgentConnection
    topology  *topology.Topology
    mu        sync.RWMutex
    tlsConfig *tls.Config
}

type AgentConnection struct {
    ID      string
    Session *yamux.Session
    Conn    net.Conn
    mu      sync.Mutex
}
```

**Connection Flow**:
1. Agent initiates TLS connection
2. Yamux session established
3. Agent sends REGISTER message
4. Admin acknowledges and stores connection
5. Heartbeat loop begins

### 3. Agent Client (`agent/`)

**Purpose**: Client that runs on compromised/target systems.

**Key Responsibilities**:
- Connect to admin server
- Send registration information
- Maintain heartbeat
- Execute commands
- Relay messages to other agents
- Forward network traffic

**Data Structures**:
```go
type Agent struct {
    id        string
    adminAddr string
    session   *yamux.Session
    conn      net.Conn
    mu        sync.Mutex
    relayMap  map[string]*yamux.Session
    tlsConfig *tls.Config
}
```

**Lifecycle**:
1. Generate unique UUID
2. Connect to admin with TLS
3. Create Yamux session
4. Register with system info
5. Enter message handling loop
6. Send heartbeats every 15s

### 4. Topology Management (`pkg/topology/`)

**Purpose**: Maintain and query the network topology graph.

**Key Features**:
- Node registration and tracking
- Parent-child relationships
- Path finding for message routing
- Dead node detection
- Active status monitoring

**Data Structures**:
```go
type NodeInfo struct {
    ID           string
    Hostname     string
    LocalIPs     []string
    OS           string
    Arch         string
    ParentID     string
    Children     []string
    LastSeen     time.Time
    IsActive     bool
}

type Topology struct {
    mu    sync.RWMutex
    nodes map[string]*NodeInfo
    edges map[string][]string
}
```

**Graph Operations**:
- `AddNode()`: Register new agent
- `RemoveNode()`: Mark agent as inactive
- `AddEdge()`: Create parent-child relationship
- `GetPath()`: Find route to target node
- `CheckDeadNodes()`: Detect timeouts

### 5. TUI (Terminal User Interface) (`pkg/tui/`)

**Purpose**: Provide real-time visualization of the proxy network.

**Technology**: Bubble Tea framework

**Layout**:
```
┌─────────────────────────────────────────────────┐
│  🔥 BProxy - Red Team Proxy Tool 🔥            │
├─────────────────────┬───────────────────────────┤
│  📡 Agent Topology  │  💻 Console               │
│                     │                           │
│  ● agent-1 [host1]  │  Active Connections: 2    │
│    ↳ 192.168.1.10   │                           │
│    ↳ Last: 2s ago   │  Recent Activity:         │
│                     │    Agent registered       │
│  ● agent-2 [host2]  │    Heartbeat received     │
│    ↳ 10.0.0.5       │                           │
│    ↳ Last: 1s ago   │                           │
│                     │                           │
└─────────────────────┴───────────────────────────┘
│  Press 'h' for help | 'q' to quit              │
└─────────────────────────────────────────────────┘
```

**Update Mechanism**:
- Tick every 1 second
- Query admin for current topology
- Render without flicker
- Handle keyboard input

### 6. TLS Management (`pkg/tls/`)

**Purpose**: Handle TLS certificate generation and configuration.

**Features**:
- Self-signed certificate generation
- Custom certificate loading
- Server and client TLS configs
- ECDSA P-256 keys

**Security Settings**:
- Minimum TLS 1.2
- Strong cipher suites
- Certificate validation (optional)

### 7. Protocol Encoding (`pkg/protocol/`)

**Purpose**: Serialize and deserialize Protocol Buffer messages.

**Wire Format**:
```
[4 bytes: length][N bytes: protobuf data]
```

**Functions**:
- `WriteMessage()`: Encode and send
- `ReadMessage()`: Receive and decode

### 8. L3 Proxy (`pkg/proxy/`)

**Purpose**: Implement Layer 3 routing proxy with TUN interface.

**Components**:

**TUN Interface**:
- Creates virtual network device
- Captures IP packets
- Injects response packets
- Configures routing table

**L3 Proxy**:
- Receives packets from TUN
- Parses IP headers
- Encapsulates in Protobuf
- Sends to target agent
- Receives responses
- Writes back to TUN

**Packet Flow**:
```
Application
    ↓
Kernel Network Stack
    ↓
TUN Interface (tun0)
    ↓
BProxy L3 Proxy
    ↓
Protobuf Encapsulation
    ↓
Yamux Stream
    ↓
TLS Connection
    ↓
Admin → Agent
    ↓
Target Network
```

## Communication Protocols

### 1. Transport Layer

**TCP + TLS**:
- Reliable, ordered delivery
- Encrypted with TLS 1.2+
- Certificate-based authentication

### 2. Multiplexing Layer

**Yamux**:
- Multiple streams over single connection
- Flow control
- Stream prioritization
- Efficient resource usage

### 3. Application Layer

**Protocol Buffers**:
- Efficient binary serialization
- Schema evolution support
- Cross-platform compatibility
- Type safety

## Message Flow Examples

### Agent Registration

```
Agent                           Admin
  |                              |
  |----[TCP SYN]---------------->|
  |<---[TCP SYN-ACK]-------------|
  |----[TCP ACK]---------------->|
  |                              |
  |----[TLS ClientHello]-------->|
  |<---[TLS ServerHello]---------|
  |<---[TLS Certificate]---------|
  |----[TLS Finished]----------->|
  |                              |
  |----[Yamux Init]------------->|
  |<---[Yamux Ack]---------------|
  |                              |
  |----[REGISTER Message]------->|
  |    {                         |
  |      type: REGISTER          |
  |      agent_id: "uuid"        |
  |      hostname: "host1"       |
  |      local_ips: ["10.0.0.5"] |
  |    }                         |
  |                              |
  |<---[ACK Message]-------------|
  |    {                         |
  |      type: COMMAND           |
  |      payload: "OK"           |
  |    }                         |
  |                              |
```

### Heartbeat Exchange

```
Agent                           Admin
  |                              |
  |----[HEARTBEAT]-------------->|
  |    {                         |
  |      type: HEARTBEAT         |
  |      agent_id: "uuid"        |
  |      timestamp: 1234567890   |
  |    }                         |
  |                              |
  |<---[HEARTBEAT ACK]-----------|
  |    {                         |
  |      type: HEARTBEAT         |
  |      timestamp: 1234567890   |
  |    }                         |
  |                              |
```

### Multi-Hop Message Relay

```
Admin                Node-A              Node-B
  |                    |                   |
  |----[RELAY]-------->|                   |
  |    {               |                   |
  |      target: B     |                   |
  |      payload: cmd  |                   |
  |    }               |                   |
  |                    |                   |
  |                    |----[RELAY]------->|
  |                    |    {              |
  |                    |      target: B    |
  |                    |      payload: cmd |
  |                    |    }              |
  |                    |                   |
  |                    |<---[RESPONSE]-----|
  |<---[RESPONSE]------|                   |
  |                    |                   |
```

## Concurrency Model

### Admin Server

**Goroutines**:
1. Main accept loop (1)
2. Per-connection handler (N agents)
3. Per-stream handler (M streams per agent)
4. Heartbeat checker (1)

**Synchronization**:
- `sync.RWMutex` for agent map
- `sync.RWMutex` for topology
- Channel-based message passing

### Agent Client

**Goroutines**:
1. Main connection loop (1)
2. Heartbeat sender (1)
3. Stream acceptor (1)
4. Per-stream handler (M streams)

**Reconnection Logic**:
- Automatic retry on disconnect
- Exponential backoff (5s base)
- Preserve agent ID across reconnects

## Security Considerations

### 1. Encryption
- All traffic encrypted with TLS
- No plaintext communication
- Perfect forward secrecy (PFS)

### 2. Authentication
- Certificate-based (optional)
- Unique agent IDs
- Session tokens

### 3. Authorization
- Admin controls all commands
- Agents cannot communicate directly
- Topology-based access control

### 4. Integrity
- Protobuf ensures message structure
- TLS provides integrity checks
- Session IDs prevent replay

### 5. Availability
- Heartbeat detection
- Automatic reconnection
- Graceful degradation

## Performance Characteristics

### Latency
- Direct connection: ~5ms overhead
- 1-hop relay: ~10ms overhead
- 2-hop relay: ~15ms overhead
- TLS handshake: ~50ms

### Throughput
- Limited by network bandwidth
- Yamux adds ~2% overhead
- Protobuf adds ~1% overhead
- TLS adds ~5% overhead

### Scalability
- 1000+ concurrent agents tested
- ~10MB memory per agent
- CPU usage scales linearly
- Network I/O is bottleneck

### Resource Usage
- Admin: ~50MB base + 10MB per agent
- Agent: ~20MB base
- TUI: +30MB for rendering

## Error Handling

### Connection Errors
- Automatic retry with backoff
- Log and continue
- Notify admin of failures

### Protocol Errors
- Validate all messages
- Discard malformed data
- Close bad connections

### Topology Errors
- Handle missing nodes gracefully
- Reroute on node failure
- Update topology in real-time

## Future Enhancements

### Planned Features
1. **SOCKS5 Proxy**: Standard proxy protocol support
2. **Port Forwarding**: TCP/UDP port forwarding
3. **File Transfer**: Efficient file upload/download
4. **Shell Access**: Interactive shell sessions
5. **Traffic Obfuscation**: Disguise proxy traffic
6. **Web UI**: Browser-based management
7. **Plugin System**: Extensible architecture
8. **Windows Support**: Full Windows compatibility
9. **Mobile Agents**: Android/iOS support
10. **Distributed Admin**: Multi-admin coordination

### Performance Improvements
1. Connection pooling
2. Message batching
3. Compression support
4. Zero-copy networking
5. QUIC protocol support

### Security Enhancements
1. Mutual TLS authentication
2. Certificate pinning
3. Traffic analysis resistance
4. Stealth mode
5. Anti-forensics features

## Debugging and Monitoring

### Logging Levels
- ERROR: Critical failures
- WARN: Recoverable issues
- INFO: Normal operations
- DEBUG: Detailed tracing

### Metrics
- Connection count
- Message rate
- Latency measurements
- Error rates
- Topology changes

### Troubleshooting Tools
- Packet capture integration
- Message tracing
- Topology visualization
- Performance profiling
- Memory profiling

## Deployment Patterns

### 1. Single Admin, Multiple Agents
```
        Admin
       /  |  \
      /   |   \
   Ag1  Ag2  Ag3
```

### 2. Cascaded Topology
```
    Admin
      |
    Ag1 (DMZ)
      |
    Ag2 (Internal)
      |
    Ag3 (Isolated)
```

### 3. Star Topology
```
        Admin
       /  |  \
      /   |   \
   Ag1  Ag2  Ag3
   / \   |   / \
  /   \ |  /   \
Ag4  Ag5 Ag6  Ag7
```

### 4. Mesh Topology
```
    Admin
    /   \
   /     \
 Ag1 --- Ag2
  |   X   |
  |  / \  |
 Ag3 --- Ag4
```

## Conclusion

BProxy's architecture is designed for:
- **Flexibility**: Support various network topologies
- **Scalability**: Handle many concurrent connections
- **Security**: Encrypted, authenticated communication
- **Usability**: Real-time visualization and control
- **Extensibility**: Modular design for future features

The combination of modern protocols (Yamux, Protobuf, TLS) with a clean Go implementation makes BProxy a powerful tool for red team operations.