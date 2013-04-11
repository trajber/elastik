package elastik

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"maestro/balancer"
	"net"
	"net/url"
	"time"
)

var (
	ErrMaxSizeExceeded = errors.New("Protocol max size exceeded")
)

type ElastikBalancer struct {
	inboundHeartbeatPort  int
	outboundHeartbeatPort int
	alive                 map[string]time.Time
	*balancer.LoadBalancer
}

func NewBalancer(inPort, outPort int) *ElastikBalancer {
	server := new(ElastikBalancer)
	server.alive = make(map[string]time.Time)
	server.inboundHeartbeatPort = inPort
	server.outboundHeartbeatPort = outPort
	server.LoadBalancer = balancer.NewLoadBalancer(nil)
	return server
}

func (lb *ElastikBalancer) ListenIncomingHeartbeats() error {
	socket, err := net.ListenUDP("udp",
		&net.UDPAddr{
			IP:   net.IPv4zero,
			Port: lb.inboundHeartbeatPort,
		},
	)

	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second * 5)
	go func() {
		for _ = range ticker.C {
			lb.checkHeartbeat()
			lb.removeDead()
		}
	}()

	for {
		data := make([]byte, 1024)
		_, remoteAddr, err := socket.ReadFromUDP(data)
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			if err := lb.handleDatagram(data, remoteAddr); err != nil {
				log.Println(err)
			}
		}()
	}

}

func (lb *ElastikBalancer) handleDatagram(data []byte, remote *net.UDPAddr) error {
	buf := bytes.NewBuffer(data)
	for {
		var length int32
		err := binary.Read(buf, binary.LittleEndian, &length)
		if err != nil {
			return err
		}

		if length == 0 {
			return nil
		}

		if length >= 1024 {
			return ErrMaxSizeExceeded
		}

		addr := make([]byte, length, length)
		if _, err := buf.Read(addr); err != nil {
			return err
		}

		host := string(addr)
		if host[len(host)-1] != '/' {
			host += "/"
		}

		if _, exists := lb.alive[host]; !exists {
			// new host arrives, add it to lb
			u, err := url.Parse(host)
			if err != nil {
				return err
			}

			if lb.LoadBalancer == nil {
				targets := []*url.URL{u}
				lb.LoadBalancer = balancer.NewLoadBalancer(targets)
			} else {
				lb.LoadBalancer.AddTarget(u)
			}

			log.Printf("[%s] send a new host [%s] \n", remote, host)
		}

		lb.alive[host] = time.Now()
	}
}

func (lb *ElastikBalancer) checkHeartbeat() error {
	log.Println("Broadcasting heartbeat message")

	remote := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: lb.outboundHeartbeatPort,
	}

	socket, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return err
	}

	if _, err = socket.Write([]byte{1, 2, 3}); err != nil {
		return err
	}

	return nil
}

func (lb *ElastikBalancer) removeDead() {
	now := time.Now()
	for k, v := range lb.alive {
		if now.Sub(v).Seconds() > 10 {
			log.Printf("No answer from %s since %s, removing it\n", k, v)
			delete(lb.alive, k)
			u, err := url.Parse(k)
			if err != nil {
				log.Println("This should not happen", err)
				continue
			}

			if err := lb.LoadBalancer.RemoveTarget(u); err != nil {
				log.Printf("Error removing host [%s] [%s]", u, err)
			} else {
				log.Printf("Host [%s] might be dead, removing it", u)
			}
		}
	}
}
