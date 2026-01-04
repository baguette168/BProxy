package agent

import (
        "crypto/tls"
        "fmt"
        "io"
        "log"
        "net"
        "os"
        "runtime"
        "sync"
        "time"

        "github.com/google/uuid"
        "github.com/hashicorp/yamux"
        pb "github.com/bproxy/bproxy/proto"
        "github.com/bproxy/bproxy/pkg/protocol"
        tlsutil "github.com/bproxy/bproxy/pkg/tls"
        "google.golang.org/protobuf/proto"
)

type Agent struct {
        id           string
        adminAddr    string
        session      *yamux.Session
        conn         net.Conn
        mu           sync.Mutex
        relayMap     map[string]*yamux.Session
        tlsConfig    *tls.Config
        cascadePort  int
        cascadeListener net.Listener
}

func NewAgent(adminAddr string, cascadePort int) *Agent {
        return &Agent{
                id:          uuid.New().String(),
                adminAddr:   adminAddr,
                relayMap:    make(map[string]*yamux.Session),
                tlsConfig:   tlsutil.GetClientTLSConfig(),
                cascadePort: cascadePort,
        }
}

func (a *Agent) Start() error {
        for {
                if err := a.connect(); err != nil {
                        log.Printf("Connection failed: %v, retrying in 5s...", err)
                        time.Sleep(5 * time.Second)
                        continue
                }

                if a.cascadePort > 0 {
                        go a.startCascadeListener()
                }

                if err := a.run(); err != nil {
                        log.Printf("Agent error: %v, reconnecting...", err)
                }

                time.Sleep(5 * time.Second)
        }
}

func (a *Agent) connect() error {
        conn, err := tls.Dial("tcp", a.adminAddr, a.tlsConfig)
        if err != nil {
                return fmt.Errorf("failed to connect: %v", err)
        }

        session, err := yamux.Client(conn, nil)
        if err != nil {
                conn.Close()
                return fmt.Errorf("failed to create yamux session: %v", err)
        }

        a.conn = conn
        a.session = session

        if err := a.register(); err != nil {
                session.Close()
                conn.Close()
                return fmt.Errorf("failed to register: %v", err)
        }

        log.Printf("Agent %s connected to admin", a.id)
        return nil
}

func (a *Agent) register() error {
        stream, err := a.session.OpenStream()
        if err != nil {
                return err
        }
        defer stream.Close()

        hostname, _ := os.Hostname()
        localIPs := a.getLocalIPs()

        regPayload := &pb.RegisterPayload{
                AgentId:  a.id,
                Hostname: hostname,
                LocalIps: localIPs,
                Os:       runtime.GOOS,
                Arch:     runtime.GOARCH,
        }

        payload, err := proto.Marshal(regPayload)
        if err != nil {
                return err
        }

        msg := &pb.Message{
                Type:      pb.MessageType_REGISTER,
                SessionId: uuid.New().String(),
                SourceId:  a.id,
                TargetId:  "admin",
                Timestamp: time.Now().Unix(),
                Payload:   payload,
        }

        if err := protocol.WriteMessage(stream, msg); err != nil {
                return err
        }

        ackMsg, err := protocol.ReadMessage(stream)
        if err != nil {
                return err
        }

        if ackMsg.Type != pb.MessageType_COMMAND || string(ackMsg.Payload) != "OK" {
                return fmt.Errorf("registration failed")
        }

        return nil
}

func (a *Agent) run() error {
        go a.heartbeatLoop()

        for {
                stream, err := a.session.AcceptStream()
                if err != nil {
                        return fmt.Errorf("failed to accept stream: %v", err)
                }

                go a.handleStream(stream)
        }
}

func (a *Agent) handleStream(stream net.Conn) {
        defer stream.Close()

        msg, err := protocol.ReadMessage(stream)
        if err != nil {
                log.Printf("Failed to read message: %v", err)
                return
        }

        switch msg.Type {
        case pb.MessageType_HEARTBEAT:
                log.Printf("Heartbeat ACK received")

        case pb.MessageType_COMMAND:
                a.handleCommand(msg, stream)

        case pb.MessageType_CONNECT:
                a.handleConnect(msg, stream)

        case pb.MessageType_RELAY:
                a.handleRelay(msg, stream)

        case pb.MessageType_DATA:
                log.Printf("Data received: %d bytes", len(msg.Payload))

        default:
                log.Printf("Unknown message type: %v", msg.Type)
        }
}

func (a *Agent) handleCommand(msg *pb.Message, stream net.Conn) {
        cmdPayload := &pb.CommandPayload{}
        if err := proto.Unmarshal(msg.Payload, cmdPayload); err != nil {
                log.Printf("Failed to unmarshal command: %v", err)
                return
        }

        log.Printf("Command received: %s %v", cmdPayload.Command, cmdPayload.Args)

        response := &pb.Message{
                Type:      pb.MessageType_DATA,
                SessionId: msg.SessionId,
                SourceId:  a.id,
                TargetId:  msg.SourceId,
                Timestamp: time.Now().Unix(),
                Payload:   []byte("Command executed"),
        }

        protocol.WriteMessage(stream, response)
}

func (a *Agent) handleConnect(msg *pb.Message, stream net.Conn) {
        connectPayload := &pb.ConnectPayload{}
        if err := proto.Unmarshal(msg.Payload, connectPayload); err != nil {
                log.Printf("Failed to unmarshal connect payload: %v", err)
                return
        }

        targetAddr := fmt.Sprintf("%s:%d", connectPayload.TargetAddress, connectPayload.TargetPort)
        log.Printf("Connecting to %s", targetAddr)

        targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
        if err != nil {
                log.Printf("Failed to connect to target: %v", err)
                response := &pb.Message{
                        Type:      pb.MessageType_DATA,
                        SessionId: msg.SessionId,
                        SourceId:  a.id,
                        TargetId:  msg.SourceId,
                        Timestamp: time.Now().Unix(),
                        Payload:   []byte("Failed"),
                }
                protocol.WriteMessage(stream, response)
                return
        }
        defer targetConn.Close()

        response := &pb.Message{
                Type:      pb.MessageType_DATA,
                SessionId: msg.SessionId,
                SourceId:  a.id,
                TargetId:  msg.SourceId,
                Timestamp: time.Now().Unix(),
                Payload:   []byte("Connected"),
        }

        if err := protocol.WriteMessage(stream, response); err != nil {
                log.Printf("Failed to send connect response: %v", err)
                return
        }

        log.Printf("Tunnel established to %s", targetAddr)

        errChan := make(chan error, 2)

        go func() {
                _, err := io.Copy(targetConn, stream)
                errChan <- err
        }()

        go func() {
                _, err := io.Copy(stream, targetConn)
                errChan <- err
        }()

        <-errChan
        log.Printf("Tunnel closed to %s", targetAddr)
}

func (a *Agent) handleRelay(msg *pb.Message, stream net.Conn) {
        if msg.TargetId == a.id {
                a.handleCommand(msg, stream)
                return
        }

        a.mu.Lock()
        relaySession, exists := a.relayMap[msg.TargetId]
        a.mu.Unlock()

        if !exists {
                log.Printf("No relay session for target %s", msg.TargetId)
                return
        }

        relayStream, err := relaySession.OpenStream()
        if err != nil {
                log.Printf("Failed to open relay stream: %v", err)
                return
        }
        defer relayStream.Close()

        protocol.WriteMessage(relayStream, msg)
}

func (a *Agent) heartbeatLoop() {
        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
                if err := a.sendHeartbeat(); err != nil {
                        log.Printf("Failed to send heartbeat: %v", err)
                        return
                }
        }
}

func (a *Agent) sendHeartbeat() error {
        stream, err := a.session.OpenStream()
        if err != nil {
                return err
        }
        defer stream.Close()

        hbPayload := &pb.HeartbeatPayload{
                AgentId:   a.id,
                Timestamp: time.Now().Unix(),
        }

        payload, err := proto.Marshal(hbPayload)
        if err != nil {
                return err
        }

        msg := &pb.Message{
                Type:      pb.MessageType_HEARTBEAT,
                SessionId: uuid.New().String(),
                SourceId:  a.id,
                TargetId:  "admin",
                Timestamp: time.Now().Unix(),
                Payload:   payload,
        }

        if err := protocol.WriteMessage(stream, msg); err != nil {
                return err
        }

        _, err = protocol.ReadMessage(stream)
        return err
}

func (a *Agent) getLocalIPs() []string {
        ips := []string{}
        addrs, err := net.InterfaceAddrs()
        if err != nil {
                return ips
        }

        for _, addr := range addrs {
                if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
                        if ipnet.IP.To4() != nil {
                                ips = append(ips, ipnet.IP.String())
                        }
                }
        }

        return ips
}

func (a *Agent) Close() error {
        if a.cascadeListener != nil {
                a.cascadeListener.Close()
        }
        if a.session != nil {
                a.session.Close()
        }
        if a.conn != nil {
                a.conn.Close()
        }
        return nil
}

func (a *Agent) startCascadeListener() error {
        tlsConfig, err := tlsutil.GetServerTLSConfig("", "")
        if err != nil {
                return fmt.Errorf("failed to setup TLS for cascade: %v", err)
        }

        listener, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", a.cascadePort), tlsConfig)
        if err != nil {
                return fmt.Errorf("failed to start cascade listener: %v", err)
        }

        a.cascadeListener = listener
        log.Printf("Cascade listener started on port %d", a.cascadePort)

        for {
                conn, err := listener.Accept()
                if err != nil {
                        log.Printf("Cascade accept error: %v", err)
                        return err
                }

                go a.handleCascadeConnection(conn)
        }
}

func (a *Agent) handleCascadeConnection(conn net.Conn) {
        defer conn.Close()

        childSession, err := yamux.Server(conn, nil)
        if err != nil {
                log.Printf("Failed to create yamux session for cascade: %v", err)
                return
        }
        defer childSession.Close()

        stream, err := childSession.AcceptStream()
        if err != nil {
                log.Printf("Failed to accept stream from child: %v", err)
                return
        }
        defer stream.Close()

        msg, err := protocol.ReadMessage(stream)
        if err != nil {
                log.Printf("Failed to read message from child: %v", err)
                return
        }

        if msg.Type != pb.MessageType_REGISTER {
                log.Printf("Expected REGISTER from child, got %v", msg.Type)
                return
        }

        regPayload := &pb.RegisterPayload{}
        if err := proto.Unmarshal(msg.Payload, regPayload); err != nil {
                log.Printf("Failed to unmarshal child register: %v", err)
                return
        }

        childID := regPayload.AgentId
        log.Printf("Child agent %s connected via cascade", childID)

        a.mu.Lock()
        a.relayMap[childID] = childSession
        a.mu.Unlock()

        defer func() {
                a.mu.Lock()
                delete(a.relayMap, childID)
                a.mu.Unlock()
        }()

        parentStream, err := a.session.OpenStream()
        if err != nil {
                log.Printf("Failed to open stream to admin for child registration: %v", err)
                return
        }
        defer parentStream.Close()

        regPayload.AgentId = childID
        payload, _ := proto.Marshal(regPayload)

        relayMsg := &pb.Message{
                Type:      pb.MessageType_REGISTER,
                SessionId: uuid.New().String(),
                SourceId:  childID,
                TargetId:  "admin",
                Timestamp: time.Now().Unix(),
                Payload:   payload,
        }

        if err := protocol.WriteMessage(parentStream, relayMsg); err != nil {
                log.Printf("Failed to relay child registration: %v", err)
                return
        }

        ackMsg, err := protocol.ReadMessage(parentStream)
        if err != nil {
                log.Printf("Failed to read ack from admin: %v", err)
                return
        }

        if err := protocol.WriteMessage(stream, ackMsg); err != nil {
                log.Printf("Failed to send ack to child: %v", err)
                return
        }

        log.Printf("Child agent %s registered with admin via relay", childID)

        for {
                childStream, err := childSession.AcceptStream()
                if err != nil {
                        log.Printf("Child session closed: %v", err)
                        return
                }

                go a.relayChildMessage(childID, childStream)
        }
}

func (a *Agent) relayChildMessage(childID string, childStream net.Conn) {
        defer childStream.Close()

        msg, err := protocol.ReadMessage(childStream)
        if err != nil {
                log.Printf("Failed to read message from child %s: %v", childID, err)
                return
        }

        msg.SourceId = childID

        parentStream, err := a.session.OpenStream()
        if err != nil {
                log.Printf("Failed to open stream to admin: %v", err)
                return
        }
        defer parentStream.Close()

        if err := protocol.WriteMessage(parentStream, msg); err != nil {
                log.Printf("Failed to relay message to admin: %v", err)
                return
        }

        response, err := protocol.ReadMessage(parentStream)
        if err != nil {
                log.Printf("Failed to read response from admin: %v", err)
                return
        }

        if err := protocol.WriteMessage(childStream, response); err != nil {
                log.Printf("Failed to send response to child: %v", err)
                return
        }
}