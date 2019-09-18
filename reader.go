package tcpclient

import (
	"encoding/binary"
	"net"
)

// Reader is a framework function for using in TCP host
func Reader(conn net.Conn, process func(input []byte) ([]byte, error)) (err error) {
	// receive data length
	dataSize := make([]byte, 4)
	_, err = conn.Read(dataSize)
	if err != nil {
		return err
	}
	// receive data
	input := make([]byte, binary.LittleEndian.Uint32(dataSize))
	_, err = conn.Read(input)
	if err != nil {
		return err
	}
	// process data
	output, err := process(input)
	if err != nil {
		return err
	}
	// send data length
	dataSize = make([]byte, 4)
	binary.LittleEndian.PutUint32(dataSize, uint32(len(output)))
	_, err = conn.Write(dataSize)
	if err != nil {
		return err
	}
	// send data
	_, err = conn.Write(input)
	if err != nil {
		return err
	}
	return nil
}
