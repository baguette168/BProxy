package admin

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/yamux"
	pb "github.com/bproxy/bproxy/proto"
	"github.com/bproxy/bproxy/pkg/protocol"
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
	listener  net.Listener
	agents    map[string]*AgentConnection
	topology  *topology.Topology
	mu        sync.RWMutex
	tlsConfig *tls.Config
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
		listener:  listener,
		agents:    make(map[string]*AgentConnection),
		topology:  topology.NewTopology(),
		tlsConfig: tlsConfig,
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
	log.Printf("Agent registered: %s (hostname: %s, IPs: %v)", agentID, regPayload.Hostname, regPayload.LocalIps)

	a.mu.Lock()
	a.agents[agentID] = &AgentConnection{
		ID:      agentID,
		Session: session,
		Conn:    conn,
	}
	a.mu.Unlock()

	a.topology.AddNode(agentID, regPayload.Hostname, regPayload.LocalIps, regPayload.Os, regPayload.Arch)

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
		a.topology.UpdateHeartbeat(agentID)
		log.Printf("Heartbeat from %s", agentID)

		ackMsg := &pb.Message{
			Type:      pb.MessageType_HEARTBEAT,
			SessionId: msg.SessionId,
			SourceId:  "admin",
			TargetId:  agentID,
			Timestamp: time.Now().Unix(),
		}
		protocol.WriteMessage(stream, ackMsg)

	case pb.MessageType_DATA:
		log.Printf("Data from %s: %d bytes", agentID, len(msg.Payload))

	case pb.MessageType_RELAY:
		log.Printf("Relay message from %s to %s", msg.SourceId, msg.TargetId)
		a.relayMessage(msg)

	default:
		log.Printf("Unknown message type from %s: %v", agentID, msg.Type)
	}
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
	return a.listener.Close()
}