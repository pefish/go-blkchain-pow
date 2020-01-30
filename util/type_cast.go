package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func MustToBuffer(data interface{}) ([]byte) {
	bytes_, err := ToBuffer(data)
	if err != nil {
		panic(err)
	}
	return bytes_
}

func ToBuffer(data interface{}) ([]byte, error) {
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.BigEndian, data)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func BufferToHexString(data []byte, prefix bool) string {
	hex := fmt.Sprintf(`%x`, data)
	if prefix {
		return `0x` + hex
	}
	return hex
}
