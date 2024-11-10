package vevoice

import (
	"encoding/binary"
	"encoding/json"
)

var (
	voiceCloneClientHeader = []byte{
		// protocol = 1, header size = 1
		0b00010001,
		// message type = 1 (full client request), message flag = 0
		0b00010000,
		// serialization type = 1 (json), compression type = 1 (gzip)
		0b00010001,
		// reserved
		0b00000000,
	}
)

func marshalVoiceCloneRequest(opts *voiceCloneOptions) (out []byte, err error) {
	var buf []byte
	if buf, err = json.Marshal(opts); err != nil {
		return
	}
	if buf, err = compressGzip(buf); err != nil {
		return
	}

	out = make([]byte, len(voiceCloneClientHeader)+4+len(buf))
	copy(out, voiceCloneClientHeader)
	binary.BigEndian.PutUint32(out[len(voiceCloneClientHeader):], uint32(len(buf)))
	copy(out[len(voiceCloneClientHeader)+4:], buf)
	return
}
