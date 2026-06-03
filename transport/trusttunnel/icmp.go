package trusttunnel

import (
	"encoding/binary"
	"net/netip"

	"github.com/sagernet/sing/common/buf"
)

type IcmpConn struct {
	httpConn
}

func (i *IcmpConn) WritePing(id uint16, destination netip.Addr, sequenceNumber uint16, ttl uint8, size uint16) error {
	request := buf.NewSize(2 + 16 + 2 + 1 + 2)
	defer request.Release()
	must(binary.Write(request, binary.BigEndian, id))
	destinationAddress := buildPaddingIP(destination)
	must1(request.Write(destinationAddress[:]))
	must(binary.Write(request, binary.BigEndian, sequenceNumber))
	must(binary.Write(request, binary.BigEndian, ttl))
	must(binary.Write(request, binary.BigEndian, size))
	_, err := i.writeFlush(request.Bytes())
	return err
}

func (i *IcmpConn) ReadPing() (id uint16, sourceAddress netip.Addr, icmpType uint8, code uint8, sequenceNumber uint16, err error) {
	err = i.waitCreated()
	if err != nil {
		return
	}
	response := buf.NewSize(2 + 16 + 1 + 1 + 2)
	defer response.Release()
	_, err = response.ReadFullFrom(i.body, response.FreeLen())
	if err != nil {
		return
	}
	must(binary.Read(response, binary.BigEndian, &id))
	var sourceAddressBuffer [16]byte
	must1(response.Read(sourceAddressBuffer[:]))
	sourceAddress = parse16BytesIP(sourceAddressBuffer)
	must(binary.Read(response, binary.BigEndian, &icmpType))
	must(binary.Read(response, binary.BigEndian, &code))
	must(binary.Read(response, binary.BigEndian, &sequenceNumber))
	return
}

func (i *IcmpConn) Close() error {
	return i.httpConn.Close()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func must1[T any](_ T, err error) {
	if err != nil {
		panic(err)
	}
}
