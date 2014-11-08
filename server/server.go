// package server implements a udp tftp server
package server

import (
	"errors"
	"log"
	"net"
	"time"

	pkt "github.com/whyrusleeping/tftp/packet"
)

// TftpMTftpMaxPacketSize is the practical limit of the size of a UDP
// packet, which is the size of an Ethernet MTU minus the headers of
// TFTP (4 bytes), UDP (8 bytes) and IP (20 bytes). (from google)
const TftpMaxPacketSize = 1468

// AckTimeout is the total time to wait before timing out on an ACK
var AckTimeout = time.Second * 20

// RetransmitTime is how long to wait before retransmitting a packet
// if an ACK has not yet been received
var RetransmitTime = time.Second * 5

// ErrTimeout is returned when an action times out
var ErrTimeout = errors.New("timed out")

// ErrUnexpectedPacket is returned when one packet type is
// received when a different one was expected
var ErrUnexpectedPacket = errors.New("unexpected packet received")

// Server is a TFTP server
type Server struct {
	// the directory to read and write files from
	servdir string
}

func NewServer(dir string) *Server {
	return &Server{
		servdir: dir,
	}
}

// Handle a new client request
func (s *Server) HandleClient(addr *net.UDPAddr, req pkt.Packet) {
	log.Println("Handle Client!")

	// Re-resolve for verification
	clientaddr, err := net.ResolveUDPAddr("udp", addr.String())
	if err != nil {
		log.Printf("Error: %s", err)
		return
	}

	switch req := req.(type) {
	case *pkt.ReqPacket:
		if req.GetType() == pkt.RRQ {
			s.HandleReadReq(req, clientaddr)
		} else if req.GetType() == pkt.WRQ {
			s.HandleWriteReq(req, clientaddr)
		} else {
			log.Println("Invalid Packet Type!")
		}
	default:
		log.Printf("Invalid packet type for new connection!")
	}
}

// Serve opens up a udp socket listening on the given
// address and handles incoming connections received on it
func (s *Server) Serve(addr string) error {
	uaddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	uconn, err := net.ListenUDP("udp", uaddr)
	if err != nil {
		return err
	}

	for { // read in new requests
		buf := make([]byte, TftpMaxPacketSize) // TODO: sync.Pool
		n, ua, err := uconn.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		log.Println("New Connection!")

		buf = buf[:n]
		packet, err := pkt.ParsePacket(buf)
		if err != nil {
			log.Printf("Got bad packet: %s", err)
			continue
		}

		go s.HandleClient(ua, packet)
	}
}
