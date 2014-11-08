package server

import (
	"errors"
	"log"
	"net"
	"os"
	"time"

	pkt "github.com/whyrusleeping/tftp/packet"
)

func sendDataPacket(d *pkt.DataPacket, con *net.UDPConn) error {
	log.Println("Sending data!")
	_, err := con.Write(d.Bytes())
	if err != nil {
		return err
	}

	// Now wait for the ACK...
	maxtimeout := time.After(AckTimeout)
	ackch := make(chan error)

	// Move it to its own function
	go func() {
		ack := make([]byte, 256)
		n, _, err := con.ReadFromUDP(ack)
		if err != nil {
			ackch <- err
			return
		}

		pack, err := pkt.ParsePacket(ack[:n])
		if err != nil {
			ackch <- err
			return
		}

		switch pack := pack.(type) {
		case *pkt.AckPacket:
			if uint16(*pack) != d.BlockNum {
				ackch <- errors.New("wrong blocknum in ack")
				return
			}
			ackch <- nil
		default:
			ackch <- pkt.ErrPacketType
		}
	}()

	// Loop and retransmit until ack or timeout
	retransmit := time.After(RetransmitTime)
	for {
		select {
		case <-maxtimeout:
			return ErrTimeout
		case <-retransmit:
			_, err := con.Write(d.Bytes())
			if err != nil {
				return err
			}
			retransmit = time.After(RetransmitTime)
		case err := <-ackch:
			return err
		}
	}
}

func (s *Server) HandleReadReq(rrq *pkt.ReqPacket, addr *net.UDPAddr) error {
	log.Printf("Read Request: %s", rrq.Filename)
	log.Printf("Dialing out %s", addr.String())

	// 'Our' Address
	listaddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return err
	}

	// Connection directly to their open port
	con, err := net.DialUDP("udp", listaddr, addr)
	if err != nil {
		return err
	}

	fi, err := os.Open(s.servdir + "/" + rrq.Filename)
	if err != nil {
		return err
	}

	buf := make([]byte, 512)
	blknum := uint16(1)
	for len(buf) == 512 {
		n, err := fi.Read(buf)
		if err != nil {
			return err
		}

		buf = buf[:n]

		data := &pkt.DataPacket{
			Data:     buf,
			BlockNum: blknum,
		}

		err = sendDataPacket(data, con)
		if err != nil {
			return err
		}
	}
	return nil
}
