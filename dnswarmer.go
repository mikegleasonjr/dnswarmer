package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bluele/gcache"
	"github.com/miekg/dns"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	upstream = kingpin.Flag("upstream", "Upstream DNS server to query.").Short('u').Required().String()
	listen   = kingpin.Flag("bind", "Bind address.").Short('b').Default(":5353").String()
	count    = kingpin.Flag("count", "Number of popular domains to keep warm.").Default("512").Short('c').Uint16()
)

var cache gcache.Cache

func main() {
	kingpin.Parse()

	udpclients, err := net.ListenPacket("udp", *listen)
	if err != nil {
		log.Println(err)
		return
	}
	defer udpclients.Close()

	client := new(dns.Client)
	cache = gcache.New(int(*count)).LFU().Build()
	quit := make(chan struct{})

	go warmer(60*time.Second, cache, client, quit)

	dns.ActivateAndServe(nil, udpclients, dns.HandlerFunc(func(writer dns.ResponseWriter, request *dns.Msg) {
		response, rtt, err := client.Exchange(request, *upstream)
		if err != nil {
			log.Println(fmt.Errorf("[%s] error forwarding %s to upstream: %w", dns.TypeToString[request.Question[0].Qtype], request.Question[0].Name, err))
			response = new(dns.Msg).SetRcode(request, dns.RcodeServerFailure)
		}

		writer.WriteMsg(response)
		writer.Close()

		if err == nil {
			onResponse(response, rtt)
		}
	}))
}

type entry struct {
	Name    string
	Type    uint16
	Expires time.Time
}

func onResponse(response *dns.Msg, rtt time.Duration) {
	if len(response.Question) != 1 {
		return
	}

	question := response.Question[0]

	if ok, ttl := firstTTL(question.Name, response.Answer); ok {
		key := fmt.Sprintf("[%s]%s", dns.TypeToString[question.Qtype], question.Name)
		entry := &entry{
			Name:    question.Name,
			Type:    question.Qtype,
			Expires: time.Now().Add(time.Duration(ttl) * time.Second),
		}

		cache.Set(key, entry)
	}
}

func warmer(interval time.Duration, cache gcache.Cache, client *dns.Client, quit <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, v := range cache.GetALL(false) {
				ent := v.(*entry)

				if !time.Now().After(ent.Expires) {
					continue
				}

				m := new(dns.Msg)
				m.SetEdns0(4096, true)
				m.SetQuestion(ent.Name, ent.Type)

				res, _, err := client.Exchange(m, *upstream)
				if err != nil {
					log.Println(fmt.Errorf("[%s] error warming %s to upstream: %w", dns.TypeToString[ent.Type], ent.Name, err))
					continue
				}

				log.Printf("[%s] warmed up %s\n", dns.TypeToString[ent.Type], ent.Name)
				if ok, ttl := firstTTL(ent.Name, res.Answer); ok {
					ent.Expires = time.Now().Add(time.Duration(ttl) * time.Second)
				}
			}
		case <-quit:
			return
		}
	}
}

func firstTTL(name string, answers []dns.RR) (bool, uint32) {
	for _, answer := range answers {
		header := answer.Header()

		if name == header.Name {
			return true, header.Ttl
		}
	}

	return false, 0
}
