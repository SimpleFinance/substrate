package tunnel

import (
	"github.com/SimpleFinance/substrate/cmd/substrate/zone/borderjump/host"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"io"
	"log"
	"net"
	"os"
)

// Tunnel describes an ssh tunnel
type Tunnel struct {
	lhost     *host.Host
	jhost     *host.Host
	rhost     *host.Host
	sshconfig ssh.ClientConfig
	control   *chan int
}

// New initializes an ssh tunnel
func New(user string, jhost *host.Host, lhost *host.Host, rhost *host.Host) *Tunnel {
	log.Printf("%s@%s -> %s@%s", user, jhost, user, rhost)
	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			SSHAgent(),
		},
	}

	t := &Tunnel{
		lhost:     lhost,
		jhost:     jhost,
		rhost:     rhost,
		sshconfig: config,
		control:   nil,
	}
	return t
}

// wrapper that can handle an error
func (tunnel *Tunnel) handleForward(localCon net.Conn, exitchan chan int) {
	err := tunnel.forward(localCon)
	if err != nil {
		log.Print("Tunnel exit")
		exitchan <- 1
	}
}

func (tunnel *Tunnel) forward(localConn net.Conn) error {
	log.Printf("Start foward: connect to border: %s", tunnel.jhost)
	serverConn, err := ssh.Dial("tcp", tunnel.jhost.String(), &tunnel.sshconfig)

	if err != nil {
		log.Printf("jump host dial error <( %s )>: %s\n", tunnel.jhost, err)
		return err
	}

	log.Printf("Start remote: %s", tunnel.rhost)
	remoteConn, err := serverConn.Dial("tcp", tunnel.rhost.String())
	if err != nil {
		log.Printf("Remote host dial error <( %s )> %s\n", tunnel.rhost, err)
		return err
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			log.Printf("io.Copy error: %s", err)
		}
	}
	log.Printf("Forward away")
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
	return nil
}

// Start | launch an ssh tunnel
func (tunnel *Tunnel) Start() error {
	listener, err := net.Listen("tcp", tunnel.lhost.String())
	log.Print("lhost opened")
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		log.Print("Accepted")
		if err != nil {
			return err
		}
		exitchan := make(chan int)
		go tunnel.handleExit(exitchan)
		go tunnel.handleForward(conn, exitchan)
	}
}

func (tunnel *Tunnel) handleExit(exitchan chan int) {
	status := <-exitchan
	os.Exit(status)
}

// Stop | shutdown the tunnel with an exit status of 0
func (tunnel *Tunnel) Stop(exitchan chan int) {
	exitchan <- 0
}

// SSHAgent | ssh authmethod wrapper
func SSHAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		pubkeys := ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
		return pubkeys
	}
	return nil
}
