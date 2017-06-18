package nettools

import (
	"net"
	"time"
	"encoding/binary"
)

func hash32to16(i uint32) uint16 {
	return uint16((i >> 16) ^ ((i & 0xffff) * 7))
}

func parsePayload(payload []byte) (net.IP, time.Time) {
	var ip [4]byte

	copy(ip[:], payload[0:4])

	ts := time.Unix(0, int64(binary.BigEndian.Uint64(payload[4:12])))

	return net.IP(ip[:]), ts
}

func encodePayload(ip net.IP, ts time.Time) []byte {
	payload := make([]byte, 12)

	copy(payload[0:4], ip[12:16])
	binary.BigEndian.PutUint64(payload[4:12], uint64(ts.UnixNano()))

	return payload
}

