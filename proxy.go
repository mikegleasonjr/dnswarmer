package dnswarmer

import (
	"bytes"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	// "github.com/miekg/dns"
)

type Proxy struct {
	clients  net.PacketConn
	upstream *net.UDPAddr // TODO, strategy
}

func NewProxy(clients net.PacketConn, upstream *net.UDPAddr) *Proxy {
	return &Proxy{clients: clients, upstream: upstream}
}

func (p *Proxy) Serve() error {
	buf := make([]byte, 1500)
	r := new(bytes.Reader)

	srv, err := net.DialUDP("udp", nil, p.upstream)
	if err != nil {
		return err
	}
	defer srv.Close()

	for {
		n, client, err := p.clients.ReadFrom(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		if n == 0 {
			continue
		}

		srv.SetWriteDeadline(time.Now().Add(10 * time.Second))
		r.Reset(buf[:n])
		_, err = io.Copy(srv, r)
		if err != nil {
			log.Println(err)
			continue
		}

		srv.SetReadDeadline(time.Now().Add(10 * time.Second))
		buf = buf[:cap(buf)]
		n, err = srv.Read(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		buf = buf[:n]
		_, err = p.clients.WriteTo(buf[:n], client)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println("served ", client.String())

		msg := dnsmessage.Message{}
		err = msg.Unpack(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println(msg)
	}

	return nil
}
