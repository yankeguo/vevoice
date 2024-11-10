package volcvoice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressGzip(t *testing.T) {
	buf, err := compressGzip([]byte("hello, world"))
	require.NoError(t, err)
	buf, err = decompressGzip(buf)
	require.NoError(t, err)
	require.Equal(t, "hello, world", string(buf))
}
