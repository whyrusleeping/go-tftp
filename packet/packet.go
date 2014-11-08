// package packet implements methods to serialize and deserialize TFTP protocol packets
package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
)

// ErrInvalidPacket is returned when given malformed data
var ErrInvalidPacket = errors.New("invalid packet")

// ErrPacketType is returned when given an invalid packet type value
var ErrPacketType = errors.New("unrecognized packet type")

// Packet Type codes as defined in rfc 1350
const (
	RRQ = uint16(iota + 1)
	WRQ
	DATA
	ACK
	ERROR
	OACK
)

// Packet represents any TFTP packet
type Packet interface {
	// GetType returns the packet type
	GetType() uint16

	// Bytes serializes the packet
	Bytes() []byte
}

type ReqPacket struct {
	Filename string
	Mode     string
	Type     uint16
}

func (p *ReqPacket) GetType() uint16 {
	return p.Type
}

// we will never need to serialize a Request Packet
// as the server, so it is safe to return nil just to
// satisfy the interface (consider adding this later for
// client use)
func (p *ReqPacket) Bytes() []byte {
	return nil
}

type DataPacket struct {
	Data     []byte
	BlockNum uint16
}

func (p *DataPacket) GetType() uint16 {
	return DATA
}

func (p *DataPacket) Bytes() []byte {
	buf := make([]byte, 4+len(p.Data))
	binary.BigEndian.PutUint16(buf[:2], DATA)
	binary.BigEndian.PutUint16(buf[2:4], p.BlockNum)
	copy(buf[4:], p.Data)
	return buf
}

type AckPacket uint16

func NewAck(blknum uint16) *AckPacket {
	a := AckPacket(blknum)
	return &a
}

func (p *AckPacket) GetType() uint16 {
	return ACK
}

func (p *AckPacket) Bytes() []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[:2], ACK)
	binary.BigEndian.PutUint16(buf[2:], uint16(*p))
	return buf
}

// ReadPacket deserializes a packet from the given buffer
func ParsePacket(buf []byte) (Packet, error) {
	if len(buf) < 2 {
		return nil, ErrInvalidPacket
	}

	pktType := binary.BigEndian.Uint16(buf[0:2])
	log.Println(pktType)
	switch pktType {
	case RRQ, WRQ:
		vals := bytes.Split(buf[2:], []byte{0})
		if len(vals) < 2 {
			return nil, ErrInvalidPacket
		}
		return &ReqPacket{
			Type:     pktType,
			Filename: string(vals[0]),
			Mode:     string(vals[1]),
		}, nil
	case ACK:
		blknum := binary.BigEndian.Uint16(buf[2:4])
		return NewAck(blknum), nil
	case DATA:
		blknum := binary.BigEndian.Uint16(buf[2:4])
		return &DataPacket{
			BlockNum: blknum,
			Data:     buf[4:],
		}, nil
	default:
		return nil, ErrPacketType
	}
}
