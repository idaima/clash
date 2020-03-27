package tun

import (
	"errors"
	adapters "github.com/Dreamacro/clash/adapters/inbound"
	"github.com/Dreamacro/clash/common/pool"
	"github.com/Dreamacro/clash/component/socks5"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/proxy/tun/dev"
	"github.com/Dreamacro/clash/tunnel"
	"github.com/kr328/tun2socket"
	"github.com/kr328/tun2socket/binding"
	"github.com/kr328/tun2socket/redirect"
	"net"
	"net/url"
)

type Tun struct {
	device  dev.Device
	adapter *tun2socket.Tun2Socket
}

type fakeConn struct {
	payload  []byte
	endpoint *binding.Endpoint
	sender   redirect.UDPSender
}

func (conn *fakeConn) Data() []byte {
	return conn.payload
}

func (conn *fakeConn) WriteBack(b []byte, addr net.Addr) (n int, err error) {
	if addr == nil {
		addr = &net.UDPAddr{
			IP:   conn.endpoint.Target.IP,
			Port: int(conn.endpoint.Target.Port),
			Zone: "",
		}
	}

	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return 0, errors.New("Invalid udp address")
	}

	ep := &binding.Endpoint{
		Source: binding.Address{
			IP:   udpAddr.IP,
			Port: uint16(udpAddr.Port),
		},
		Target: conn.endpoint.Source,
	}

	return len(b), conn.sender(b, ep)
}

func (conn *fakeConn) Close() error {
	return nil
}

func (conn *fakeConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   conn.endpoint.Source.IP,
		Port: int(conn.endpoint.Source.Port),
		Zone: "",
	}
}

func NewTunProxy(device string, gateway string, mirror string) (*Tun, error) {
	deviceURL, err := url.Parse(device)
	if err != nil {
		return nil, err
	}

	gatewayIP := net.ParseIP(gateway).To4()
	mirrorIP := net.ParseIP(mirror).To4()
	if len(gatewayIP) == 0 || len(mirrorIP) == 0 {
		return nil, errors.New("Invalid gateway or mirror")
	}

	tunDevice, err := dev.OpenTunDevice(deviceURL)
	if err != nil {
		return nil, err
	}

	mtu, err := tunDevice.MTU()
	if err != nil {
		_ = tunDevice.Close()
		return nil, err
	}

	result := &Tun{
		device:  tunDevice,
		adapter: tun2socket.NewTun2Socket(tunDevice, mtu, gatewayIP, mirrorIP),
	}

	result.adapter.SetAllocator(func(length int) []byte {
		buf := pool.BufPool.Get().([]byte)
		if cap(buf) > length {
			return buf[:length]
		} else {
			pool.BufPool.Put(buf)
			return make([]byte, length)
		}
	})
	result.adapter.SetTCPHandler(func(conn net.Conn, endpoint *binding.Endpoint) {
		addr := socks5.ParseAddrToSocksAddr(&net.TCPAddr{
			IP:   endpoint.Target.IP,
			Port: int(endpoint.Target.Port),
			Zone: "",
		})

		tunnel.Add(adapters.NewSocket(addr, conn, C.SOCKS, C.TCP))
	})
	result.adapter.SetUDPHandler(func(payload []byte, endpoint *binding.Endpoint, sender redirect.UDPSender) {
		addr := socks5.ParseAddrToSocksAddr(&net.TCPAddr{
			IP:   endpoint.Target.IP,
			Port: int(endpoint.Target.Port),
			Zone: "",
		})
		conn := &fakeConn{
			payload:  payload,
			endpoint: endpoint,
			sender:   sender,
		}

		tunnel.AddPacket(adapters.NewPacket(addr, conn, C.SOCKS))
	})

	result.adapter.Start()

	return result, nil
}

func (t *Tun) Close() {
	t.adapter.Close()
}

func (t *Tun) Address() string {
	return t.device.Name()
}
