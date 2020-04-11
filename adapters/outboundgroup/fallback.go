package outboundgroup

import (
	"encoding/json"

	"github.com/Dreamacro/clash/adapters/outbound"
	"github.com/Dreamacro/clash/adapters/provider"
	"github.com/Dreamacro/clash/common/singledo"
	C "github.com/Dreamacro/clash/constant"
)

type Fallback struct {
	*outbound.Base
	single    *singledo.Single
	providers []provider.ProxyProvider
}

func (f *Fallback) Now() string {
	proxy := f.findAliveProxy()
	return proxy.Name()
}

func (f *Fallback) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	p := f.findAliveProxy()
	return newGroupDialer(f, p.Dialer(parent))
}

func (f *Fallback) SupportUDP() bool {
	proxy := f.findAliveProxy()
	return proxy.SupportUDP()
}

func (f *Fallback) MarshalJSON() ([]byte, error) {
	var all []string
	for _, proxy := range f.proxies() {
		all = append(all, proxy.Name())
	}
	return json.Marshal(map[string]interface{}{
		"type": f.Type().String(),
		"now":  f.Now(),
		"all":  all,
	})
}

func (f *Fallback) proxies() []C.Proxy {
	elm, _, _ := f.single.Do(func() (interface{}, error) {
		return getProvidersProxies(f.providers), nil
	})

	return elm.([]C.Proxy)
}

func (f *Fallback) findAliveProxy() C.Proxy {
	proxies := f.proxies()
	for _, proxy := range proxies {
		if proxy.Alive() {
			return proxy
		}
	}

	return f.proxies()[0]
}

func NewFallback(name string, providers []provider.ProxyProvider) *Fallback {
	return &Fallback{
		Base:      outbound.NewBase(name, "", C.Fallback, false),
		single:    singledo.NewSingle(defaultGetProxiesDuration),
		providers: providers,
	}
}
