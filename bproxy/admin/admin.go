package admin

import (
        "crypto/tls"
        "fmt"
        "io"
        "log"
        "net"
        "sync"
        "time"

        "github.com/hashicorp/yamux"
        pb "github.com/bproxy/bproxy/proto"
        "github.com/bproxy/bproxy/pkg/protocol"
        "github.com/bproxy/bproxy/pkg/socks5"
        "github.com/bproxy/bproxy/pkg/topology"
        tlsutil "github.com/bproxy/bproxy/pkg/tls"
        "google.golang.org/protobuf/proto"
)

type AgentConnection struct {
        ID      string
        Session *yamux.Session
        Conn    net.Conn
        mu      sync.Mutex
}

type Admin struct {
        listener      net.Listener
        agents        map[string]*AgentConnection
        topology      *topology.Topology
        mu            sync.RWMutex
        tlsConfig     *tls.Config
        socks5Servers map[int]net.Listener
        socks5Mu      sync.Mutex
}

func NewAdmin(addr, certFile, keyFile string) (*Admin, error) {
        tlsConfig, err := tlsutil.GetServerTLSConfig(certFile, keyFile)
        if err != nil {
                return nil, fmt.Errorf("failed to setup TLS: %v", err)
        }

        listener, err := tls.Listen("tcp", addr, tlsConfig)
        if err != nil {
                return nil, fmt.Errorf("failed to listen: %v", err)
        }

        return &Admin{
                listener:      listener,
                agents:        make(map[string]*AgentConnection),
                topology:      topology.NewTopology(),
                tlsConfig:     tlsConfig,
                socks5Servers: make(map[int]net.Listener),
        }, nil
}

func (a *Admin) Start() error {
        log.Printf("Admin server listening on %s", a.listener.Addr())

        go a.heartbeatChecker()

        for {
                conn, err := a.listener.Accept()
                if err != nil {
                        log.Printf("Accept error: %v", err)
                        continue
                }

                go a.handleConnection(conn)
        }
}

func (a *Admin) handleConnection(conn net.Conn) {
        defer conn.Close()

        session, err := yamux.Server(conn, nil)
        if err != nil {
                log.Printf("Failed to create yamux session: %v", err)
                return
        }
        defer session.Close()

        stream, err := session.AcceptStream()
        if err != nil {
                log.Printf("Failed to accept stream: %v", err)
                return
        }
        defer stream.Close()

        msg, err := protocol.ReadMessage(stream)
        if err != nil {
                log.Printf("Failed to read initial message: %v", err)
                return
        }

        if msg.Type != pb.MessageType_REGISTER {
                log.Printf("Expected REGISTER message, got %v", msg.Type)
                return
        }

        regPayload := &pb.RegisterPayload{}
        if err := proto.Unmarshal(msg.Payload, regPayload); err != nil {
                log.Printf("Failed to unmarshal register payload: %v", err)
                return
        }

        agentID := regPayload.AgentId
        parentID := regPayload.ParentId
        log.Printf("Agent registered: %s (hostname: %s, IPs: %v, parent: %s)", agentID, regPayload.Hostname, regPayload.LocalIps, parentID)

        a.mu.Lock()
        a.agents[agentID] = &AgentConnection{
                ID:      agentID,
                Session: session,
                Conn:    conn,
        }
        a.mu.Unlock()

        a.topology.AddNode(agentID, regPayload.Hostname, regPayload.LocalIps, regPayload.Os, regPayload.Arch)
        
        // If this is a cascaded agent, establish parent-child relationship in topology
        if parentID != "" && parentID != "admin" {
                if err := a.topology.AddEdge(parentID, agentID); err != nil {
                        log.Printf("Warning: Failed to add topology edge %s -> %s: %v", parentID, agentID, err)
                }
        }

        ackMsg := &pb.Message{
                Type:      pb.MessageType_COMMAND,
                SessionId: msg.SessionId,
                SourceId:  "admin",
                TargetId:  agentID,
                Timestamp: time.Now().Unix(),
                Payload:   []byte("OK"),
        }
        if err := protocol.WriteMessage(stream, ackMsg); err != nil {
                log.Printf("Failed to send ACK: %v", err)
                return
        }

        a.handleAgent(agentID, session)
}

func (a *Admin) handleAgent(agentID string, session *yamux.Session) {
        defer func() {
                a.mu.Lock()
                delete(a.agents, agentID)
                a.mu.Unlock()
                a.topology.RemoveNode(agentID)
                log.Printf("Agent disconnected: %s", agentID)
        }()

        for {
                stream, err := session.AcceptStream()
                if err != nil {
                        log.Printf("Agent %s: failed to accept stream: %v", agentID, err)
                        return
                }

                go a.handleStream(agentID, stream)
        }
}

func (a *Admin) handleStream(agentID string, stream net.Conn) {
        defer stream.Close()

        msg, err := protocol.ReadMessage(stream)
        if err != nil {
                log.Printf("Agent %s: failed to read message: %v", agentID, err)
                return
        }

        switch msg.Type {
        case pb.MessageType_HEARTBEAT:
                // Get the original sender of the heartbeat (may be a cascaded agent)
                heartbeatSourceId := msg.SourceId
                if heartbeatSourceId == "" {
                        heartbeatSourceId = agentID
                }
                
                a.topology.UpdateHeartbeat(heartbeatSourceId)
                log.Printf("Heartbeat from %s", heartbeatSourceId)

                ackMsg := &pb.Message{
                        Type:      pb.MessageType_HEARTBEAT,
                        SessionId: msg.SessionId,
                        SourceId:  "admin",
                        TargetId:  heartbeatSourceId,
                        Timestamp: time.Now().Unix(),
                }
                protocol.WriteMessage(stream, ackMsg)

        case pb.MessageType_REGISTER:
                // Handle cascaded agent registration
                a.handleCascadeRegister(msg, stream)

        case pb.MessageType_DATA:
                log.Printf("Data from %s: %d bytes", agentID, len(msg.Payload))

        case pb.MessageType_RELAY:
                log.Printf("Relay message from %s to %s", msg.SourceId, msg.TargetId)
                a.relayMessage(msg)

        default:
                log.Printf("Unknown message type from %s: %v", agentID, msg.Type)
        }
}

func (a *Admin) handleCascadeRegister(msg *pb.Message, stream net.Conn) {
        regPayload := &pb.RegisterPayload{}
        if err := proto.Unmarshal(msg.Payload, regPayload); err != nil {
                log.Printf("Failed to unmarshal cascade register payload: %v", err)
                return
        }

        childID := regPayload.AgentId
        parentID := regPayload.ParentId

        log.Printf("Cascade agent registered: %s (hostname: %s, parent: %s)", 
                childID, regPayload.Hostname, parentID)

        // Add node to topology
        a.topology.AddNode(childID, regPayload.Hostname, regPayload.LocalIps, 
                regPayload.Os, regPayload.Arch)

        // Establish parent-child relationship
        if parentID != "" && parentID != "admin" {
                if err := a.topology.AddEdge(parentID, childID); err != nil {
                        log.Printf("Warning: Failed to add topology edge %s -> %s: %v", 
                                parentID, childID, err)
                }
        }

        // Send ACK response
        ackMsg := &pb.Message{
                Type:      pb.MessageType_COMMAND,
                SessionId: msg.SessionId,
                SourceId:  "admin",
                TargetId:  childID,
                Timestamp: time.Now().Unix(),
                Payload:   []byte("OK"),
        }

        if err := protocol.WriteMessage(stream, ackMsg); err != nil {
                log.Printf("Failed to send cascade ACK: %v", err)
                return
        }

        log.Printf("Cascade agent %s registered successfully", childID)
}

func (a *Admin) relayMessage(msg *pb.Message) error {
        a.mu.RLock()
        targetConn, exists := a.agents[msg.TargetId]
        a.mu.RUnlock()

        if !exists {
                return fmt.Errorf("target agent %s not found", msg.TargetId)
        }

        stream, err := targetConn.Session.OpenStream()
        if err != nil {
                return fmt.Errorf("failed to open stream to target: %v", err)
        }
        defer stream.Close()

        return protocol.WriteMessage(stream, msg)
}

func (a *Admin) SendCommand(targetID string, cmd *pb.CommandPayload) error {
        path := a.topology.GetPath(targetID)
        if len(path) == 0 {
                return fmt.Errorf("no path to target %s", targetID)
        }

        payload, err := proto.Marshal(cmd)
        if err != nil {
                return err
        }

        msg := &pb.Message{
                Type:      pb.MessageType_COMMAND,
                SessionId: fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
                SourceId:  "admin",
                TargetId:  targetID,
                Timestamp: time.Now().Unix(),
                Payload:   payload,
        }

        if len(path) == 1 {
                return a.sendDirectMessage(targetID, msg)
        }

        return a.relayMessage(msg)
}

func (a *Admin) sendDirectMessage(agentID string, msg *pb.Message) error {
        a.mu.RLock()
        agentConn, exists := a.agents[agentID]
        a.mu.RUnlock()

        if !exists {
                return fmt.Errorf("agent %s not connected", agentID)
        }

        stream, err := agentConn.Session.OpenStream()
        if err != nil {
                return err
        }
        defer stream.Close()

        return protocol.WriteMessage(stream, msg)
}

func (a *Admin) heartbeatChecker() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
                deadNodes := a.topology.CheckDeadNodes(60 * time.Second)
                for _, nodeID := range deadNodes {
                        log.Printf("Node %s marked as dead", nodeID)
                        a.mu.Lock()
                        if conn, exists := a.agents[nodeID]; exists {
                                conn.Conn.Close()
                                delete(a.agents, nodeID)
                        }
                        a.mu.Unlock()
                }
        }
}

func (a *Admin) GetTopology() *topology.Topology {
        return a.topology
}

func (a *Admin) GetAgents() map[string]*AgentConnection {
        a.mu.RLock()
        defer a.mu.RUnlock()

        agents := make(map[string]*AgentConnection)
        for k, v := range a.agents {
                agents[k] = v
        }
        return agents
}

func (a *Admin) Close() error {
        a.socks5Mu.Lock()
        for _, listener := range a.socks5Servers {
                listener.Close()
        }
        a.socks5Mu.Unlock()
        return a.listener.Close()
}

func (a *Admin) StartSocks5(port int, targetID string) error {
        a.socks5Mu.Lock()
        if _, exists := a.socks5Servers[port]; exists {
                a.socks5Mu.Unlock()
                return fmt.Errorf("SOCKS5 server already running on port %d", port)
        }
        a.socks5Mu.Unlock()

        listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
        if err != nil {
                return fmt.Errorf("failed to start SOCKS5 listener: %v", err)
        }

        a.socks5Mu.Lock()
        a.socks5Servers[port] = listener
        a.socks5Mu.Unlock()

        log.Printf("SOCKS5 proxy started on 127.0.0.1:%d -> agent %s", port, targetID)

        go func() {
                defer func() {
                        a.socks5Mu.Lock()
                        delete(a.socks5Servers, port)
                        a.socks5Mu.Unlock()
                        listener.Close()
                }()

                for {
                        conn, err := listener.Accept()
                        if err != nil {
                                log.Printf("SOCKS5 accept error: %v", err)
                                return
                        }

                        go a.handleSocks5Connection(conn, targetID)
                }
        }()

        return nil
}

func (a *Admin) StopSocks5(port int) error {
        a.socks5Mu.Lock()
        listener, exists := a.socks5Servers[port]
        if !exists {
                a.socks5Mu.Unlock()
                return fmt.Errorf("no SOCKS5 server running on port %d", port)
        }
        delete(a.socks5Servers, port)
        a.socks5Mu.Unlock()

        return listener.Close()
}

func (a *Admin) handleSocks5Connection(clientConn net.Conn, targetID string) {
        defer clientConn.Close()

        if err := socks5.HandleSocks5Handshake(clientConn); err != nil {
                log.Printf("SOCKS5 handshake failed: %v", err)
                return
        }

        req, err := socks5.ParseRequest(clientConn)
        if err != nil {
                log.Printf("SOCKS5 parse request failed: %v", err)
                socks5.SendReply(clientConn, socks5.ReplyGeneralFailure)
                return
        }

        log.Printf("SOCKS5 request: %s:%d via agent %s", req.DstAddr, req.DstPort, targetID)

        // Get the path to the target agent using topology
        path := a.topology.GetPath(targetID)
        if len(path) == 0 {
                log.Printf("No path to target agent %s", targetID)
                socks5.SendReply(clientConn, socks5.ReplyHostUnreachable)
                return
        }

        // The first hop is always the direct connection
        firstHop := path[0]
        
        a.mu.RLock()
        agentConn, exists := a.agents[firstHop]
        a.mu.RUnlock()

        if !exists {
                log.Printf("First hop agent %s not found", firstHop)
                socks5.SendReply(clientConn, socks5.ReplyHostUnreachable)
                return
        }

        stream, err := agentConn.Session.OpenStream()
        if err != nil {
                log.Printf("Failed to open stream to agent: %v", err)
                socks5.SendReply(clientConn, socks5.ReplyGeneralFailure)
                return
        }
        defer stream.Close()

        connectPayload := &pb.ConnectPayload{
                TargetAgentId: targetID,
                TargetAddress: req.DstAddr,
                TargetPort:    int32(req.DstPort),
        }

        payload, err := proto.Marshal(connectPayload)
        if err != nil {
                log.Printf("Failed to marshal connect payload: %v", err)
                socks5.SendReply(clientConn, socks5.ReplyGeneralFailure)
                return
        }

        msg := &pb.Message{
                Type:      pb.MessageType_CONNECT,
                SessionId: fmt.Sprintf("socks5-%d", time.Now().UnixNano()),
                SourceId:  "admin",
                TargetId:  targetID,
                Timestamp: time.Now().Unix(),
                Payload:   payload,
        }

        if err := protocol.WriteMessage(stream, msg); err != nil {
                log.Printf("Failed to send connect message: %v", err)
                socks5.SendReply(clientConn, socks5.ReplyGeneralFailure)
                return
        }

        response, err := protocol.ReadMessage(stream)
        if err != nil {
                log.Printf("Failed to read connect response: %v", err)
                socks5.SendReply(clientConn, socks5.ReplyGeneralFailure)
                return
        }

        if response.Type != pb.MessageType_DATA || string(response.Payload) != "Connected" {
                log.Printf("Agent connection failed")
                socks5.SendReply(clientConn, socks5.ReplyConnectionRefused)
                return
        }

        if err := socks5.SendReply(clientConn, socks5.ReplySuccess); err != nil {
                log.Printf("Failed to send SOCKS5 success reply: %v", err)
                return
        }

        log.Printf("SOCKS5 tunnel established: %s:%d via path %v", req.DstAddr, req.DstPort, path)

        errChan := make(chan error, 2)

        go func() {
                _, err := io.Copy(stream, clientConn)
                errChan <- err
        }()

        go func() {
                _, err := io.Copy(clientConn, stream)
                errChan <- err
        }()

        <-errChan
        log.Printf("SOCKS5 tunnel closed: %s:%d", req.DstAddr, req.DstPort)
}

func (a *Admin) GetSocks5Servers() map[int]string {
        a.socks5Mu.Lock()
        defer a.socks5Mu.Unlock()

        servers := make(map[int]string)
        for port := range a.socks5Servers {
                servers[port] = fmt.Sprintf("127.0.0.1:%d", port)
        }
        return servers
}