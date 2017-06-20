package host

import (
	"fmt"
)

// Host address + port
type Host struct {
	address string
	port    int
}

// New constructor for a new Host
func New(address string, port int) *Host {
	host := Host{
		address: address,
		port:    port,
	}
	return &host
}

func (h *Host) String() string {
	s := fmt.Sprintf("%s:%d", h.address, h.port)
	return s
}
