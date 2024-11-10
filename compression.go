package volcvoice

import (
	"bytes"
	"compress/gzip"
	"io"
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
