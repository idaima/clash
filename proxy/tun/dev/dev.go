package dev

import "github.com/kr328/tun2socket"

type Device interface {
	tun2socket.TunDevice
	MTU() (int, error)
	Name() string
	URL() string
}
