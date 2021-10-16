package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bluele/gcache"
	"github.com/miekg/dns"
)

const (
	upstream = "10.0.10.1:53"
)

var cache gcache.Cache

func main() {
	cache = gcache.New(512).LFU().Build()

	udpclients, err := net.ListenPacket("udp", ":5354")
	if err != nil {
		log.Println(err)
		return
	}
	defer udpclients.Close()

	c := new(dns.Client)
	c.SingleInflight = true

	dns.ActivateAndServe(nil, udpclients, dns.HandlerFunc(func(writer dns.ResponseWriter, request *dns.Msg) {
		response, rtt, err := c.Exchange(request, upstream)
		if err != nil {
			response = new(dns.Msg).SetRcode(request, dns.RcodeServerFailure)
		}

		writer.WriteMsg(response)
		writer.Close()

		if err == nil {
			onResponse(response, rtt)
		}
	}))
}

func onResponse(response *dns.Msg, rtt time.Duration) {
	for _, answer := range response.Answer {
		header := answer.Header()

		if header.Rrtype != dns.TypeA && header.Rrtype != dns.TypeAAAA {
			continue
		}

		key := fmt.Sprintf("%s_%d_%d", header.Name, header.Rrtype, header.Class)

		cache.Set(key, response)

		b, _ := json.Marshal(answer)
		fmt.Println(cache.Keys(false)...)
		fmt.Printf("key: %s\nttl: %d\nrtt: %s\nhdr: %s\n\n", key, header.Ttl, rtt, b)
	}
}
