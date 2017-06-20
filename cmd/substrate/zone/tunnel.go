package zone

import (
	"log"
	"net"

	host "github.com/SimpleFinance/substrate/cmd/substrate/zone/borderjump/host"
	tunnel "github.com/SimpleFinance/substrate/cmd/substrate/zone/borderjump/tunnel"
)

// TunnelInput | params for tunnel initialization
type TunnelInput struct {
	Rip          net.IP
	LocalPort    int
	RemotePort   int
	ManifestPath string
	JumpHostUser string
}

// MakeTunnel | convenience construction function for creating a tunnel
// Fetches and applies defaults for border and director if not provided
func MakeTunnel(params *TunnelInput) error {
	user := params.JumpHostUser
	zoneManifest, err := ReadManifest(params.ManifestPath)

	if err != nil {
		return err
	}

	borderEIP, err := getTerraformOutput("border_eip", zoneManifest.TerraformState)
	if err != nil {
		return err
	}

	// use the director IP as the default host
	rhostIP := params.Rip.String()
	if rhostIP == "<nil>" { // @@ mmmmmkay.
		rhostIP, err = getTerraformOutput("director_ip", zoneManifest.TerraformState)
		log.Printf("Using director ip: %s", rhostIP)
		if err != nil {
			return err
		}
	}

	jhost := host.New(borderEIP, 22)
	lhost := host.New("127.0.0.1", params.LocalPort)

	rhost := host.New(rhostIP, params.RemotePort)
	t := tunnel.New(user, jhost, lhost, rhost)

	err = t.Start()
	if err != nil {
		return err
	}
	return nil
}
