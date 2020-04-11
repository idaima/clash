package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/Dreamacro/clash/common/structure"
	obfs "github.com/Dreamacro/clash/component/simple-obfs"
	"github.com/Dreamacro/clash/component/snell"
	C "github.com/Dreamacro/clash/constant"
)

type Snell struct {
	*Base
	psk        []byte
	obfsOption *simpleObfsOption
}

type SnellOption struct {
	Name     string                 `proxy:"name"`
	Server   string                 `proxy:"server"`
	Port     int                    `proxy:"port"`
	Psk      string                 `proxy:"psk"`
	ObfsOpts map[string]interface{} `proxy:"obfs-opts,omitempty"`
}

type snellDialer struct {
	*Snell
	parent C.ProxyDialer
}

func (s *snellDialer) StreamConn(c net.Conn, metadata *C.Metadata) (net.Conn, error) {
	switch s.obfsOption.Mode {
	case "tls":
		c = obfs.NewTLSObfs(c, s.obfsOption.Host)
	case "http":
		_, port, _ := net.SplitHostPort(s.addr)
		c = obfs.NewHTTPObfs(c, s.obfsOption.Host, port)
	}
	c = snell.StreamConn(c, s.psk)
	port, _ := strconv.Atoi(metadata.DstPort)
	err := snell.WriteHeader(c, metadata.String(), uint(port))
	return c, err
}

func (s *snellDialer) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, C.Connection, error) {
	c, connection, err := s.parent.DialContext(ctx, newRelayMetadata(C.TCP, s.addr))
	if err != nil {
		return nil, nil, fmt.Errorf("%s connect error: %w", s.addr, err)
	}

	c, err = s.StreamConn(c, metadata)

	connection.AppendToChains(s)
	return c, connection, nil
}

func (s *snellDialer) DialUDP(metadata *C.Metadata) (C.PacketConn, C.Connection, error) {
	return nil, nil, errors.New("unsupported")
}

func (s *Snell) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return &snellDialer{
		Snell:  s,
		parent: parent,
	}
}

func NewSnell(option SnellOption) (*Snell, error) {
	addr := net.JoinHostPort(option.Server, strconv.Itoa(option.Port))
	psk := []byte(option.Psk)

	decoder := structure.NewDecoder(structure.Option{TagName: "obfs", WeaklyTypedInput: true})
	obfsOption := &simpleObfsOption{Host: "bing.com"}
	if err := decoder.Decode(option.ObfsOpts, obfsOption); err != nil {
		return nil, fmt.Errorf("snell %s initialize obfs error: %w", addr, err)
	}

	if obfsOption.Mode != "tls" && obfsOption.Mode != "http" {
		return nil, fmt.Errorf("snell %s obfs mode error: %s", addr, obfsOption.Mode)
	}

	return &Snell{
		Base: &Base{
			name: option.Name,
			addr: addr,
			tp:   C.Snell,
		},
		psk:        psk,
		obfsOption: obfsOption,
	}, nil
}
