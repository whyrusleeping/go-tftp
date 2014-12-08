package client

import (
	"errors"
	"fmt"
	pkt "github.com/whyrusleeping/go-tftp/packet"
	"io"
	"net"
	"time"
)

// TftpMTftpMaxPacketSize is the practical limit of the size of a UDP
// packet, which is the size of an Ethernet MTU minus the headers of
// TFTP (4 bytes), UDP (8 bytes) and IP (20 bytes). (source: google).
const TftpMaxPacketSize = 1468

type TftpClient struct {
	servaddr *net.UDPAddr
	udpconn  *net.UDPConn
}

func NewTftpClient(addr string) (*TftpClient, error) {
	laddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}

	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	uconn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}

	return &TftpClient{
		servaddr: raddr,
		udpconn:  uconn,
	}, nil
}

func (cl *TftpClient) sendPacket(p pkt.Packet, addr *net.UDPAddr) error {
	data := p.Bytes()
	n, err := cl.udpconn.WriteToUDP(p.Bytes(), addr)
	if err != nil {
		return err
	}

	if n != len(data) {
		return errors.New("Failed to send entire packet")
	}

	return nil
}

func (cl *TftpClient) recvPacket() (pkt.Packet, *net.UDPAddr, error) {
	buf := make([]byte, TftpMaxPacketSize)
	n, addr, err := cl.udpconn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}

	if n == len(buf) {
		fmt.Println("Warning! Read entire buffer size, possible errors occurred!")
	}
	buf = buf[:n]

	pkt, err := pkt.ParsePacket(buf)
	if err != nil {
		return nil, nil, err
	}

	return pkt, addr, nil
}

func (cl *TftpClient) PutFile(filename string, data io.Reader) (int, error) {
	req := &pkt.ReqPacket{
		Filename: filename,
		Mode:     "octet",
		Type:     pkt.WRQ,
	}

	err := cl.sendPacket(req, cl.servaddr)
	if err != nil {
		return 0, nil
	}

	blockSize := 512
	blknum := uint16(0)
	xferred := 0
	buf := make([]byte, blockSize)
	for {
		p, addr, err := cl.recvPacket()
		if err != nil {
			return 0, err
		}
		switch p := p.(type) {
		case *pkt.ErrorPacket:
			return 0, p
		case *pkt.AckPacket:
			if p.GetBlocknum() != blknum {
				fmt.Printf("Wrong blocknumber! (%d != %d)\n", p.GetBlocknum(), blknum)
				continue
			}
		default:
			return 0, fmt.Errorf("unexpected packet: %v, %d", p, p.GetType())
		}
		blknum++
		buf = buf[:blockSize]
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}
		if n == 0 {
			break
		}
		buf = buf[:n]
		xferred += n
		err = cl.sendPacket(&pkt.DataPacket{
			BlockNum: blknum,
			Data:     buf,
		}, addr)
		if err != nil {
			return 0, err
		}
	}

	return xferred, nil
}

func (cl *TftpClient) GetFile(filename string) (int, time.Duration, error) {
	before := time.Now()
	req := &pkt.ReqPacket{
		Filename: filename,
		Mode:     "octet",
		Type:     pkt.RRQ,
	}

	err := cl.sendPacket(req, cl.servaddr)
	if err != nil {
		return 0, 0, err
	}

	blockSize := 512
	xfersize := 0
	blknum := uint16(1)
	for {
		datapkt, sendaddr, err := cl.recvPacket()
		if err != nil {
			return 0, 0, err
		}
		if datapkt.GetType() == pkt.ERROR {
			return 0, 0, datapkt.(*pkt.ErrorPacket)
		}
		if datapkt.GetType() != pkt.DATA {
			return 0, 0, errors.New("Expected DATA packet!")
		}
		data := datapkt.(*pkt.DataPacket)

		//fmt.Printf("got data:\n%s\n", data.Data)

		ack := pkt.NewAck(blknum)
		err = cl.sendPacket(ack, sendaddr)
		if err != nil {
			return 0, 0, err
		}

		xfersize += len(data.Data)
		if len(data.Data) < blockSize {
			break
		}
		blknum++
	}
	return xfersize, time.Now().Sub(before), nil
}
