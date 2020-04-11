package outbound

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/Dreamacro/clash/common/queue"
	C "github.com/Dreamacro/clash/constant"
)

var (
	defaultURLTestTimeout = time.Second * 5
)

type Base struct {
	name string
	addr string
	tp   C.AdapterType
	udp  bool
}

func (b *Base) Name() string {
	return b.name
}

func (b *Base) Type() C.AdapterType {
	return b.tp
}

func (b *Base) SupportUDP() bool {
	return b.udp
}

func (b *Base) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type": b.Type().String(),
	})
}

func (b *Base) Addr() string {
	return b.addr
}

func NewBase(name string, addr string, tp C.AdapterType, udp bool) *Base {
	return &Base{name, addr, tp, udp}
}

type connection struct {
	chain C.Chain
}

func (c *connection) Chains() C.Chain {
	return c.chain
}

func (c *connection) AppendToChains(a C.ProxyAdapter) {
	c.chain = append(c.chain, a.Name())
}

func newConnection() C.Connection {
	return &connection{
		chain: C.Chain{},
	}
}

type Proxy struct {
	C.ProxyAdapter
	history *queue.Queue
	alive   bool
}

type proxyDialer struct {
	*Proxy
	dialer C.ProxyDialer
}

func (p *proxyDialer) DialContext(ctx context.Context, metadata *C.Metadata) (C.Conn, C.Connection, error) {
	conn, connection, err := p.dialer.DialContext(ctx, metadata)
	if err != nil {
		p.alive = false
	}
	return conn, connection, err
}

func (p *proxyDialer) DialUDP(metadata *C.Metadata) (C.PacketConn, C.Connection, error) {
	return p.dialer.DialUDP(metadata)
}

func (p *Proxy) Alive() bool {
	return p.alive
}

func (p *Proxy) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return &proxyDialer{
		Proxy:  p,
		dialer: p.ProxyAdapter.Dialer(parent),
	}
}

func (p *Proxy) DelayHistory() []C.DelayHistory {
	queue := p.history.Copy()
	histories := []C.DelayHistory{}
	for _, item := range queue {
		histories = append(histories, item.(C.DelayHistory))
	}
	return histories
}

// LastDelay return last history record. if proxy is not alive, return the max value of uint16.
func (p *Proxy) LastDelay() (delay uint16) {
	var max uint16 = 0xffff
	if !p.alive {
		return max
	}

	last := p.history.Last()
	if last == nil {
		return max
	}
	history := last.(C.DelayHistory)
	if history.Delay == 0 {
		return max
	}
	return history.Delay
}

func (p *Proxy) MarshalJSON() ([]byte, error) {
	inner, err := p.ProxyAdapter.MarshalJSON()
	if err != nil {
		return inner, err
	}

	mapping := map[string]interface{}{}
	_ = json.Unmarshal(inner, &mapping)
	mapping["history"] = p.DelayHistory()
	mapping["name"] = p.Name()
	return json.Marshal(mapping)
}

// URLTest get the delay for the specified URL
func (p *Proxy) URLTest(ctx context.Context, url string) (t uint16, err error) {
	defer func() {
		p.alive = err == nil
		record := C.DelayHistory{Time: time.Now()}
		if err == nil {
			record.Delay = t
		}
		p.history.Put(record)
		if p.history.Len() > 10 {
			p.history.Pop()
		}
	}()

	addr, err := urlToMetadata(url)
	if err != nil {
		return
	}

	start := time.Now()
	dialer := p.Dialer(NewRootDialer())

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (conn net.Conn, err error) {
			conn, _, err = dialer.DialContext(ctx, addr)
			return
		},
		// from http.DefaultTransport
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
	t = uint16(time.Since(start) / time.Millisecond)
	return
}

func NewProxy(adapter C.ProxyAdapter) *Proxy {
	return &Proxy{adapter, queue.New(10), true}
}
