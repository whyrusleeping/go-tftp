package client

import (
	"errors"
	"fmt"
	pkt "github.com/whyrusleeping/go-tftp/packet"
	"io"
	"net"
	"time"
)

type TftpClient struct {
	servaddr  *net.UDPAddr
	udpconn   *net.UDPConn
	Blocksize int
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
		servaddr:  raddr,
		udpconn:   uconn,
		Blocksize: 512,
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

func (cl *TftpClient) recvPacket(buf []byte) (pkt.Packet, *net.UDPAddr, error) {
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
		Filename:  filename,
		Mode:      "octet",
		Type:      pkt.WRQ,
		BlockSize: cl.Blocksize,
	}

	err := cl.sendPacket(req, cl.servaddr)
	if err != nil {
		return 0, err
	}

	blknum := uint16(0)
	xferred := 0
	pktbuf := make([]byte, 1000)
	buf := make([]byte, cl.Blocksize)
	for {
		p, addr, err := cl.recvPacket(pktbuf)
		if err != nil {
			return 0, err
		}
		switch p := p.(type) {
		case *pkt.ErrorPacket:
			fmt.Println("Error packet.")
			return 0, p
		case *pkt.AckPacket:
			if p.GetBlocknum() != blknum {
				fmt.Printf("Wrong blocknumber! (%d != %d)\n", p.GetBlocknum(), blknum)
				continue
			}
			if blknum == 0 && cl.Blocksize != 512 {
				fmt.Println("Didnt get expected OACK.")
			}
		case *pkt.OAckPacket:
			if blknum != 0 {
				return 0, errors.New("Received OACK at unexpected time...")
			}
			if p.Options["blksize"] != fmt.Sprint(cl.Blocksize) {
				fmt.Printf("Blocksize Negotiation failed!\ngot '%s'\n", p.Options["blocksize"])
			}
		default:
			return 0, fmt.Errorf("unexpected packet: %v, %d", p, p.GetType())
		}
		blknum++
		buf = buf[:cl.Blocksize]
		n, err := data.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
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
		if n == 0 {
			break
		}
	}

	return xferred, nil
}

func (cl *TftpClient) GetFile(filename string) (int, time.Duration, error) {
	before := time.Now()
	req := &pkt.ReqPacket{
		Filename:  filename,
		Mode:      "octet",
		Type:      pkt.RRQ,
		BlockSize: cl.Blocksize,
	}

	err := cl.sendPacket(req, cl.servaddr)
	if err != nil {
		return 0, 0, err
	}

	xfersize := 0
	blknum := uint16(1)
	pktbuf := make([]byte, cl.Blocksize+100)
	for {
		datapkt, sendaddr, err := cl.recvPacket(pktbuf)
		if err != nil {
			return 0, 0, err
		}

		var data []byte
		switch datapkt.GetType() {
		case pkt.ERROR:
			return 0, 0, datapkt.(*pkt.ErrorPacket)
		case pkt.DATA:
			datapkt := datapkt.(*pkt.DataPacket)
			data = datapkt.Data
		case pkt.OACK:
			fmt.Println("GOT OACK!!!")
			blknum--
			oack := datapkt.(*pkt.OAckPacket)
			if oack.Options["blksize"] != fmt.Sprint(cl.Blocksize) {
				return 0, 0, errors.New("failed to negotiate blocksize")
			}
		default:
			fmt.Printf("Got: %d\n", datapkt.GetType())
			fmt.Println(datapkt.(*pkt.AckPacket).GetBlocknum())
			fmt.Println(blknum)
			return 0, 0, errors.New("Expected DATA packet!")
		}

		ack := pkt.NewAck(blknum)
		err = cl.sendPacket(ack, sendaddr)
		if err != nil {
			return 0, 0, err
		}

		xfersize += len(data)
		if len(data) < cl.Blocksize {
			break
		}
		blknum++
	}
	return xfersize, time.Now().Sub(before), nil
}
