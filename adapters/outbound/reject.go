package outbound

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	C "github.com/Dreamacro/clash/constant"
)

type Reject struct {
	*Base
}

type rejectDialer struct {
	*Reject
}

func (r *rejectDialer) DialContext(_ context.Context, _ *C.Metadata) (C.Conn, C.Connection, error) {
	connection := newConnection()
	connection.AppendToChains(r)
	return &NopConn{}, connection, nil
}

func (r *rejectDialer) DialUDP(metadata *C.Metadata) (C.PacketConn, C.Connection, error) {
	return nil, nil, errors.New("match reject rule")
}

func (r *Reject) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return &rejectDialer{r}
}

func NewReject() *Reject {
	return &Reject{
		Base: &Base{
			name: "REJECT",
			tp:   C.Reject,
			udp:  true,
		},
	}
}

type NopConn struct{}

func (rw *NopConn) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (rw *NopConn) Write(_ []byte) (int, error) {
	return 0, io.EOF
}

// Close is fake function for net.Conn
func (rw *NopConn) Close() error { return nil }

// LocalAddr is fake function for net.Conn
func (rw *NopConn) LocalAddr() net.Addr { return nil }

// RemoteAddr is fake function for net.Conn
func (rw *NopConn) RemoteAddr() net.Addr { return nil }

// SetDeadline is fake function for net.Conn
func (rw *NopConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline is fake function for net.Conn
func (rw *NopConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline is fake function for net.Conn
func (rw *NopConn) SetWriteDeadline(time.Time) error { return nil }
