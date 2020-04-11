package outboundgroup

import (
	"context"
	"time"

	"github.com/Dreamacro/clash/adapters/provider"
	C "github.com/Dreamacro/clash/constant"
)

const (
	defaultGetProxiesDuration = time.Second * 5
)

type groupDialer struct {
	C.ProxyAdapter
	dialer C.ProxyDialer
}

func (g *groupDialer) DialContext(ctx context.Context, metadata *C.Metadata) (conn C.Conn, connection C.Connection, err error) {
	conn, connection, err = g.dialer.DialContext(ctx, metadata)
	if err != nil {
		return
	}
	connection.AppendToChains(g)
	return
}

func (g *groupDialer) DialUDP(metadata *C.Metadata) (pc C.PacketConn, connection C.Connection, err error) {
	pc, connection, err = g.dialer.DialUDP(metadata)
	if err != nil {
		return
	}

	connection.AppendToChains(g)
	return
}

func getProvidersProxies(providers []provider.ProxyProvider) []C.Proxy {
	proxies := []C.Proxy{}
	for _, provider := range providers {
		proxies = append(proxies, provider.Proxies()...)
	}
	return proxies
}

func newGroupDialer(group C.ProxyAdapter, dialer C.ProxyDialer) *groupDialer {
	return &groupDialer{
		ProxyAdapter: group,
		dialer:       dialer,
	}
}
