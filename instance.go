package elastik

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"net/url"
)

type ElastikInstance struct {
	aliveMessage          []byte
	inboundHeartbeatPort  int
	outboundHeartbeatPort int
	servingOn             []*url.URL
}

func NewInstance(inPort, outPort int) *ElastikInstance {
	instance := new(ElastikInstance)
	instance.inboundHeartbeatPort = inPort
	instance.outboundHeartbeatPort = outPort
	return instance
}

func (ei *ElastikInstance) AddURL(u *url.URL) {
	for _, v := range ei.servingOn {
		if v.String() == u.String() {
			return
		}
	}

	ei.servingOn = append(ei.servingOn, u)
	ei.buildAliveMessage()
}

func (ei *ElastikInstance) buildAliveMessage() {
	buf := bytes.NewBuffer(nil)
	for _, v := range ei.servingOn {
		httpURL := v.String()
		length := int32(len(httpURL))
		binary.Write(buf, binary.LittleEndian, length)
		buf.WriteString(httpURL)
	}
	ei.aliveMessage = buf.Bytes()
}

func (ei *ElastikInstance) ListenIncomingHeartbeats() error {
	socket, err := net.ListenUDP("udp",
		&net.UDPAddr{
			IP:   net.IPv4zero,
			Port: ei.inboundHeartbeatPort,
		},
	)

	if err != nil {
		return err
	}

	for {
		data := make([]byte, 1024)
		_, remoteAddr, err := socket.ReadFromUDP(data)
		if err != nil {
			log.Println(err)
		}
		go func() {
			if err := ei.handleDatagram(data, remoteAddr); err != nil {
				log.Println(err)
			}
		}()
	}
}

func (ei *ElastikInstance) handleDatagram(data []byte, remote *net.UDPAddr) error {
	log.Println("Handling heartbeat message from", remote)

	buf := bytes.NewBuffer(data)
	// check if it is a heart beat message
	for i := 1; i <= 3; i++ {
		n, err := buf.ReadByte()
		if err != nil {
			return err
		}

		if int(n) != i {
			return errors.New("Not a heartbeat message")
		}
	}

	// it seams to be a heartbeat message... sending alive response
	remote.Port = ei.outboundHeartbeatPort

	log.Println("Sending heartbeat response to", remote)
	socket, err := net.DialUDP("udp", nil, remote)
	if err != nil {
		return err
	}

	_, err = socket.Write(ei.aliveMessage)
	if err != nil {
		return err
	}

	return nil
}
