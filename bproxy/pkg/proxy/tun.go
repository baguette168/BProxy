package proxy

import (
	"fmt"
	"log"
	"net"
	"os/exec"

	"github.com/songgao/water"
)

type TunProxy struct {
	iface     *water.Interface
	ipAddr    string
	netmask   string
	targetNet string
}

func NewTunProxy(ipAddr, netmask, targetNet string) (*TunProxy, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}

	iface, err := water.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %v", err)
	}

	tp := &TunProxy{
		iface:     iface,
		ipAddr:    ipAddr,
		netmask:   netmask,
		targetNet: targetNet,
	}

	if err := tp.setupInterface(); err != nil {
		iface.Close()
		return nil, err
	}

	return tp, nil
}

func (tp *TunProxy) setupInterface() error {
	ifaceName := tp.iface.Name()

	cmd := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%s", tp.ipAddr, tp.netmask), "dev", ifaceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set IP address: %v", err)
	}

	cmd = exec.Command("ip", "link", "set", "dev", ifaceName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring interface up: %v", err)
	}

	cmd = exec.Command("ip", "route", "add", tp.targetNet, "dev", ifaceName)
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to add route (may already exist): %v", err)
	}

	log.Printf("TUN interface %s configured: %s/%s -> %s", ifaceName, tp.ipAddr, tp.netmask, tp.targetNet)
	return nil
}

func (tp *TunProxy) Read(buf []byte) (int, error) {
	return tp.iface.Read(buf)
}

func (tp *TunProxy) Write(buf []byte) (int, error) {
	return tp.iface.Write(buf)
}

func (tp *TunProxy) Close() error {
	return tp.iface.Close()
}

func (tp *TunProxy) Start(handler func([]byte) error) error {
	buf := make([]byte, 2048)

	for {
		n, err := tp.Read(buf)
		if err != nil {
			return fmt.Errorf("read error: %v", err)
		}

		packet := make([]byte, n)
		copy(packet, buf[:n])

		go func(pkt []byte) {
			if err := handler(pkt); err != nil {
				log.Printf("Handler error: %v", err)
			}
		}(packet)
	}
}

func ParseIPPacket(packet []byte) (srcIP, dstIP net.IP, protocol uint8, payload []byte, err error) {
	if len(packet) < 20 {
		return nil, nil, 0, nil, fmt.Errorf("packet too short")
	}

	version := packet[0] >> 4
	if version != 4 {
		return nil, nil, 0, nil, fmt.Errorf("not IPv4 packet")
	}

	headerLen := int(packet[0]&0x0F) * 4
	if len(packet) < headerLen {
		return nil, nil, 0, nil, fmt.Errorf("invalid header length")
	}

	protocol = packet[9]
	srcIP = net.IP(packet[12:16])
	dstIP = net.IP(packet[16:20])
	payload = packet[headerLen:]

	return srcIP, dstIP, protocol, payload, nil
}