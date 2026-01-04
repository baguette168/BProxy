package socks5

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	Socks5Version = 0x05
	NoAuth        = 0x00
	ConnectCmd    = 0x01
	IPv4Address   = 0x01
	DomainName    = 0x03
	IPv6Address   = 0x04
)

type Request struct {
	Version  byte
	Command  byte
	Reserved byte
	AddrType byte
	DstAddr  string
	DstPort  uint16
}

func HandleSocks5Handshake(conn net.Conn) error {
	buf := make([]byte, 256)

	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read handshake failed: %v", err)
	}

	if n < 2 || buf[0] != Socks5Version {
		return fmt.Errorf("invalid SOCKS5 version")
	}

	_, err = conn.Write([]byte{Socks5Version, NoAuth})
	if err != nil {
		return fmt.Errorf("write auth response failed: %v", err)
	}

	return nil
}

func ParseRequest(conn net.Conn) (*Request, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("read request header failed: %v", err)
	}

	req := &Request{
		Version:  buf[0],
		Command:  buf[1],
		Reserved: buf[2],
		AddrType: buf[3],
	}

	if req.Version != Socks5Version {
		return nil, fmt.Errorf("invalid SOCKS5 version in request")
	}

	if req.Command != ConnectCmd {
		return nil, fmt.Errorf("unsupported command: %d", req.Command)
	}

	switch req.AddrType {
	case IPv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, fmt.Errorf("read IPv4 address failed: %v", err)
		}
		req.DstAddr = net.IP(addr).String()

	case DomainName:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return nil, fmt.Errorf("read domain length failed: %v", err)
		}
		domainLen := int(lenBuf[0])
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return nil, fmt.Errorf("read domain failed: %v", err)
		}
		req.DstAddr = string(domain)

	case IPv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return nil, fmt.Errorf("read IPv6 address failed: %v", err)
		}
		req.DstAddr = net.IP(addr).String()

	default:
		return nil, fmt.Errorf("unsupported address type: %d", req.AddrType)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return nil, fmt.Errorf("read port failed: %v", err)
	}
	req.DstPort = binary.BigEndian.Uint16(portBuf)

	return req, nil
}

func SendReply(conn net.Conn, rep byte) error {
	reply := []byte{
		Socks5Version,
		rep,
		0x00,
		0x01,
		0, 0, 0, 0,
		0, 0,
	}
	_, err := conn.Write(reply)
	return err
}

const (
	ReplySuccess                 = 0x00
	ReplyGeneralFailure          = 0x01
	ReplyConnectionNotAllowed    = 0x02
	ReplyNetworkUnreachable      = 0x03
	ReplyHostUnreachable         = 0x04
	ReplyConnectionRefused       = 0x05
	ReplyTTLExpired              = 0x06
	ReplyCommandNotSupported     = 0x07
	ReplyAddressTypeNotSupported = 0x08
)