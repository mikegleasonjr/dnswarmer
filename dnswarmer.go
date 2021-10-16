package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"time"

	"github.com/bluele/gcache"
	"github.com/miekg/dns"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	upstream  = kingpin.Flag("upstream", "Upstream DNS server to query.").Short('u').Required().String()
	listen    = kingpin.Flag("bind", "Bind address.").Short('b').Default(":5353").String()
	cacheSize = kingpin.Flag("max", "Max number of popular domains to warm up.").Default("512").Short('m').Int()
)

var cache gcache.Cache

func main() {
	kingpin.Parse()

	cache = gcache.New(*cacheSize).LFU().Build()

	udpclients, err := net.ListenPacket("udp", *listen)
	if err != nil {
		log.Println(err)
		return
	}
	defer udpclients.Close()

	client := new(dns.Client)
	client.SingleInflight = true

	quit := make(chan struct{})
	go warmer(5*time.Second, cache, client, quit)

	dns.ActivateAndServe(nil, udpclients, dns.HandlerFunc(func(writer dns.ResponseWriter, request *dns.Msg) {
		response, rtt, err := client.Exchange(request, *upstream)
		if err != nil {
			log.Println(err)
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
	Qtype   uint16
	Expires time.Time
}

func onResponse(response *dns.Msg, rtt time.Duration) {
	if len(response.Question) != 1 {
		return
	}

	question := response.Question[0]
	var ok bool
	var ttl uint32

	if ok, ttl = shortestTtl(question.Name, response.Answer); !ok {
		return
	}

	key := fmt.Sprintf("%s_%d", question.Name, question.Qtype)
	entry := &entry{
		Name:    question.Name,
		Qtype:   question.Qtype,
		Expires: time.Now().Add(time.Duration(ttl) * time.Second),
	}

	cache.Set(key, entry)
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
				m.SetQuestion(ent.Name, ent.Qtype)

				res, _, err := client.Exchange(m, *upstream)
				if err != nil {
					log.Println(err)
					continue
				}

				log.Printf("warmed %s\n", ent.Name)
				if ok, ttl := shortestTtl(ent.Name, res.Answer); ok {
					ent.Expires = time.Now().Add(time.Duration(ttl) * time.Second)
				}
			}
		case <-quit:
			return
		}
	}
}

func shortestTtl(name string, answers []dns.RR) (bool, uint32) {
	shortestTtl := uint32(math.MaxUint32)

	for _, answer := range answers {
		header := answer.Header()

		if name != header.Name ||
			header.Ttl > shortestTtl {
			continue
		}

		shortestTtl = header.Ttl
	}

	if shortestTtl == math.MaxUint32 {
		return false, 0
	}

	return true, shortestTtl
}
