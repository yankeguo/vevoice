package stream_wire

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressDecompressGzip(t *testing.T) {
	buf := []byte("test")
	compressed, err := compressGzip(buf)
	require.NoError(t, err)

	decompressed, err := decompressGzip(compressed)
	require.NoError(t, err)
	require.Equal(t, buf, decompressed)
}

func TestEncodeVoiceCloneRequest(t *testing.T) {
	opts := map[string]any{
		"app": map[string]any{
			"appid": "test",
			"token": "test",
			"uid":   "test",
		},
		"request": map[string]any{
			"text": "test",
		},
	}

	buf, err := EncodeRequest(opts)
	require.NoError(t, err)
	require.Equal(t, requestHeader, buf[:4])
	size := binary.BigEndian.Uint32(buf[4:8])
	require.Len(t, buf, 8+int(size))

	buf, err = decompressGzip(buf[8:])
	require.NoError(t, err)

	var opts2 map[string]any
	err = json.Unmarshal(buf, &opts2)
	require.NoError(t, err)

	require.Equal(t, opts, opts2)
}
