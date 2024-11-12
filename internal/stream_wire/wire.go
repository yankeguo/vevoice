package stream_wire

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

var (
	requestHeader = []byte{
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

func compressGzip(buf []byte) ([]byte, error) {
	b := &bytes.Buffer{}
	w := gzip.NewWriter(b)
	if _, err := w.Write(buf); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompressGzip(buf []byte) ([]byte, error) {
	if r, err := gzip.NewReader(bytes.NewReader(buf)); err != nil {
		return nil, err
	} else {
		defer r.Close()
		return io.ReadAll(r)
	}
}

func EncodeRequest(opts any) (out []byte, err error) {
	var buf []byte
	if buf, err = json.Marshal(opts); err != nil {
		return
	}
	if buf, err = compressGzip(buf); err != nil {
		return
	}

	out = make([]byte, len(requestHeader)+4+len(buf))
	copy(out, requestHeader)
	binary.BigEndian.PutUint32(out[len(requestHeader):], uint32(len(buf)))
	copy(out[len(requestHeader)+4:], buf)
	return
}

type Response struct {
	IsPayload    bool
	PayloadData  []byte
	PayloadIndex int32

	IsError      bool
	ErrorCode    int32
	ErrorMessage string
}

func DecodeResponse(buf []byte) (out Response, err error) {
	if len(buf) < 4 {
		err = errors.New("invalid response, header too short")
		return
	}

	if proto := buf[0] >> 4; proto != 1 {
		err = errors.New("invalid response, invalid protocol")
		return
	}

	headerSize := buf[0] & 0b00001111

	if len(buf) < int(headerSize*4) {
		err = errors.New("invalid response, header too short")
		return
	}

	messageType := buf[1] >> 4
	messageFlag := buf[1] & 0b00001111
	_ = buf[2] >> 4 // serialization
	compression := buf[2] & 0b00001111
	_ = buf[3]                // reserved
	_ = buf[4 : headerSize*4] // extensions
	payload := buf[headerSize*4:]

	if messageType == 0b1011 {
		out.IsPayload = true
		if messageFlag == 0 {
		} else {
			out.PayloadIndex = int32(binary.BigEndian.Uint32(payload[0:4]))
			_ = int(int32(binary.BigEndian.Uint32(payload[4:8]))) // payload size
			out.PayloadData = payload[8:]
			if compression == 1 {
				if out.PayloadData, err = decompressGzip(out.PayloadData); err != nil {
					return
				}
			}
		}
	} else if messageType == 0b1111 {
		out.IsError = true
		out.ErrorCode = int32(binary.BigEndian.Uint32(payload[0:4]))
		_ = int(int32(binary.BigEndian.Uint32(payload[4:8]))) // error message size
		messageBuf := payload[8:]
		if compression == 1 {
			if messageBuf, err = decompressGzip(messageBuf); err != nil {
				return
			}
		}
		out.ErrorMessage = string(messageBuf)
	}

	return
}
