package outboundgroup

import (
	"encoding/json"
	"time"

	"github.com/Dreamacro/clash/adapters/outbound"
	"github.com/Dreamacro/clash/adapters/provider"
	"github.com/Dreamacro/clash/common/singledo"
	C "github.com/Dreamacro/clash/constant"
)

type URLTest struct {
	*outbound.Base
	single     *singledo.Single
	fastSingle *singledo.Single
	providers  []provider.ProxyProvider
}

func (u *URLTest) Now() string {
	return u.fast().Name()
}

func (u *URLTest) Dialer(parent C.ProxyDialer) C.ProxyDialer {
	return newGroupDialer(u, u.fast().Dialer(parent))
}

func (u *URLTest) proxies() []C.Proxy {
	elm, _, _ := u.single.Do(func() (interface{}, error) {
		return getProvidersProxies(u.providers), nil
	})

	return elm.([]C.Proxy)
}

func (u *URLTest) fast() C.Proxy {
	elm, _, _ := u.fastSingle.Do(func() (interface{}, error) {
		proxies := u.proxies()
		fast := proxies[0]
		min := fast.LastDelay()
		for _, proxy := range proxies[1:] {
			if !proxy.Alive() {
				continue
			}

			delay := proxy.LastDelay()
			if delay < min {
				fast = proxy
				min = delay
			}
		}
		return fast, nil
	})

	return elm.(C.Proxy)
}

func (u *URLTest) SupportUDP() bool {
	return u.fast().SupportUDP()
}

func (u *URLTest) MarshalJSON() ([]byte, error) {
	var all []string
	for _, proxy := range u.proxies() {
		all = append(all, proxy.Name())
	}
	return json.Marshal(map[string]interface{}{
		"type": u.Type().String(),
		"now":  u.Now(),
		"all":  all,
	})
}

func NewURLTest(name string, providers []provider.ProxyProvider) *URLTest {
	return &URLTest{
		Base:       outbound.NewBase(name, "", C.URLTest, false),
		single:     singledo.NewSingle(defaultGetProxiesDuration),
		fastSingle: singledo.NewSingle(time.Second * 10),
		providers:  providers,
	}
}
