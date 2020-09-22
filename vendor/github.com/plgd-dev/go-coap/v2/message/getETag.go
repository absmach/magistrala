package message

import (
	"encoding/binary"
	"hash/crc64"
	"io"
)

// GetETag calculate ETag from payload via CRC64
func GetETag(r io.ReadSeeker) ([]byte, error) {
	if r == nil {
		return make([]byte, 8), nil
	}
	c64 := crc64.New(crc64.MakeTable(crc64.ISO))
	orig, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	for {
		bufR := buf
		n, err := r.Read(bufR)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		bufR = bufR[:n]
		c64.Write(bufR)
	}
	_, err = r.Seek(orig, io.SeekStart)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, c64.Sum64())
	return b, nil
}
