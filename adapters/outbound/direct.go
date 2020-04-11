package outbound

import (
	"context"
	C "github.com/Dreamacro/clash/constant"
)

type Direct struct {
	*Base
}

type directDialer struct {
	*Direct
	parent C.ProxyDialer
}

func (d *directDialer) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, C.Connection, error) {
	c, connection, err := d.parent.DialContext(ctx, metadata)
	if err != nil {
		return nil, nil, err
	}

	connection.AppendToChains(d)
	return c, connection, nil
}

func (d *directDialer) DialUDP(metadata *C.Metadata) (C.PacketConn, C.Connection, error) {
	pc, connection, err := d.parent.DialUDP(metadata)
	if err != nil {
		return nil, nil, err
	}

	connection.AppendToChains(d)
	return pc, connection, nil
}

func (d *Direct) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return &directDialer{
		Direct: d,
		parent: parent,
	}
}

func NewDirect() *Direct {
	return &Direct{
		Base: &Base{
			name: "DIRECT",
			tp:   C.Direct,
			udp:  true,
		},
	}
}
