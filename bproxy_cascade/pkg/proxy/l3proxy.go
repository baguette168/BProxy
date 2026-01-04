package proxy

import (
	"fmt"
	"log"
	"sync"

	pb "github.com/bproxy/bproxy/proto"
	"github.com/bproxy/bproxy/admin"
	"google.golang.org/protobuf/proto"
)

type L3Proxy struct {
	tun       *TunProxy
	admin     *admin.Admin
	targetID  string
	sessions  map[string]*ProxySession
	mu        sync.RWMutex
}

type ProxySession struct {
	ID       string
	SrcIP    string
	DstIP    string
	Protocol uint8
}

func NewL3Proxy(tunIP, tunMask, targetNet string, adminServer *admin.Admin, targetAgentID string) (*L3Proxy, error) {
	tun, err := NewTunProxy(tunIP, tunMask, targetNet)
	if err != nil {
		return nil, err
	}

	return &L3Proxy{
		tun:      tun,
		admin:    adminServer,
		targetID: targetAgentID,
		sessions: make(map[string]*ProxySession),
	}, nil
}

func (lp *L3Proxy) Start() error {
	log.Printf("Starting L3 Proxy for target agent: %s", lp.targetID)

	return lp.tun.Start(lp.handlePacket)
}

func (lp *L3Proxy) handlePacket(packet []byte) error {
	srcIP, dstIP, protocol, _, err := ParseIPPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to parse packet: %v", err)
	}

	log.Printf("L3 Proxy: %s -> %s (protocol: %d)", srcIP, dstIP, protocol)

	sessionID := fmt.Sprintf("%s-%s-%d", srcIP, dstIP, protocol)

	lp.mu.Lock()
	if _, exists := lp.sessions[sessionID]; !exists {
		lp.sessions[sessionID] = &ProxySession{
			ID:       sessionID,
			SrcIP:    srcIP.String(),
			DstIP:    dstIP.String(),
			Protocol: protocol,
		}
	}
	lp.mu.Unlock()

	dataPayload := &pb.DataPayload{
		Data:     packet,
		Sequence: 0,
	}

	payload, err := proto.Marshal(dataPayload)
	if err != nil {
		return err
	}

	_ = &pb.Message{
		Type:      pb.MessageType_DATA,
		SessionId: sessionID,
		SourceId:  "admin",
		TargetId:  lp.targetID,
		Payload:   payload,
	}

	return lp.admin.SendCommand(lp.targetID, &pb.CommandPayload{
		Command: "forward_packet",
		Args:    []string{sessionID},
	})
}

func (lp *L3Proxy) WritePacket(packet []byte) error {
	_, err := lp.tun.Write(packet)
	return err
}

func (lp *L3Proxy) Close() error {
	return lp.tun.Close()
}