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
	for i := 0; i <= len(input)/limitMessageSize; i++ {
		start := i * limitMessageSize
		end := (i + 1) * limitMessageSize
		if end > len(input) {
			end = len(input)
		}
		_, err = conn.Read(input[start:end])
		if err != nil {
			return err
		}
	}
	for i := range input {
		if input[i] != byte(i%256) {
			break
		}
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
	for i := 0; i <= len(output)/limitMessageSize; i++ {
		start := i * limitMessageSize
		end := (i + 1) * limitMessageSize
		if end > len(output) {
			end = len(output)
		}
		_, err = conn.Write(output[start:end])
		if err != nil {
			return err
		}
	}
	return nil
}
