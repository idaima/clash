package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/component/trojan"
	C "github.com/Dreamacro/clash/constant"
)

type Trojan struct {
	*Base
	instance *trojan.Trojan
}

type TrojanOption struct {
	Name           string   `proxy:"name"`
	Server         string   `proxy:"server"`
	Port           int      `proxy:"port"`
	Password       string   `proxy:"password"`
	ALPN           []string `proxy:"alpn,omitempty"`
	SNI            string   `proxy:"sni,omitempty"`
	SkipCertVerify bool     `proxy:"skip-cert-verify,omitempty"`
	UDP            bool     `proxy:"udp,omitempty"`
}

type trojanDialer struct {
	*Trojan
	parent C.ProxyDialer
}

func (t *trojanDialer) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	c, err := t.instance.StreamConn(c)
	if err != nil {
		return nil, fmt.Errorf("%s connect error: %w", t.addr, err)
	}

	err = t.instance.WriteHeader(c, trojan.CommandTCP, serializesSocksAddr(metadata))
	return c, err
}

func (t *trojanDialer) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, C.Connection, error) {
	c, connection, err := t.parent.DialContext(ctx, newRelayMetadata(C.TCP, t.addr))
	if err != nil {
		return nil, nil, fmt.Errorf("%s connect error: %w", t.addr, err)
	}
	c, err = t.StreamConn(c, metadata)
	if err != nil {
		return nil, nil, err
	}

	connection.AppendToChains(t)
	return c, connection, nil
}

func (t *trojanDialer) DialUDP(metadata *C.Metadata) (C.PacketConn, C.Connection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpTimeout)
	defer cancel()

	c, connection, err := t.parent.DialContext(ctx, newRelayMetadata(C.TCP, t.addr))
	if err != nil {
		return nil, nil, fmt.Errorf("%s connect error: %w", t.addr, err)
	}
	c, err = t.instance.StreamConn(c)
	if err != nil {
		return nil, nil, fmt.Errorf("%s connect error: %w", t.addr, err)
	}

	err = t.instance.WriteHeader(c, trojan.CommandUDP, serializesSocksAddr(metadata))
	if err != nil {
		return nil, nil, err
	}

	pc := t.instance.PacketConn(c)

	connection.AppendToChains(t)
	return &trojanPacketConn{pc, c}, connection, nil
}

func (t *Trojan) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return &trojanDialer{
		Trojan: t,
		parent: parent,
	}
}

func (t *Trojan) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type": t.Type().String(),
	})
}

func NewTrojan(option TrojanOption) (*Trojan, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))

	tOption := &trojan.Option{
		Password:           option.Password,
		ALPN:               option.ALPN,
		ServerName:         option.Server,
		SkipCertVerify:     option.SkipCertVerify,
		ClientSessionCache: getClientSessionCache(),
	}

	if option.SNI != "" {
		tOption.ServerName = option.SNI
	}

	return &Trojan{
		Base: &Base{
			name: option.Name,
			addr: addr,
			tp:   C.Trojan,
			udp:  option.UDP,
		},
		instance: trojan.New(tOption),
	}, nil
}

type trojanPacketConn struct {
	net.PacketConn
	conn net.Conn
}

func (tpc *trojanPacketConn) WriteWithMetadata(p []byte, metadata *C.Metadata) (n int, err error) {
	return trojan.WritePacket(tpc.conn, serializesSocksAddr(metadata), p)
}
