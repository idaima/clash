package outbound

import (
	"context"
	"github.com/Dreamacro/clash/component/dialer"
	"github.com/Dreamacro/clash/component/resolver"
	C "github.com/Dreamacro/clash/constant"
	"net"
	"time"
)

type rootDialer struct{}

func (r *rootDialer) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, C.Connection, error) {
	address := net.JoinHostPort(metadata.Host, metadata.DstPort)
	if metadata.DstIP != nil {
		address = net.JoinHostPort(metadata.DstIP.String(), metadata.DstPort)
	}

	c, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, err
	}

	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(30 * time.Second)
	}

	return c, newConnection(), nil
}

func (r *rootDialer) DialUDP(_ *C.Metadata) (C.PacketConn, C.Connection, error) {
	pc, err := dialer.ListenPacket("udp", "")
	if err != nil {
		return nil, nil, err
	}
	return &rootPacketConn{pc}, newConnection(), nil
}

type rootPacketConn struct {
	net.PacketConn
}

func (rp *rootPacketConn) WriteWithMetadata(p []byte, metadata *C.Metadata) (n int, err error) {
	if !metadata.Resolved() {
		ip, err := resolver.ResolveIP(metadata.Host)
		if err != nil {
			return 0, err
		}
		metadata.DstIP = ip
	}
	return rp.WriteTo(p, metadata.UDPAddr())
}

func NewRootDialer() C.ProxyDialer {
	return &rootDialer{}
}
