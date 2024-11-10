package volcvoice

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeVoiceCloneRequest(t *testing.T) {
	opts := &voiceCloneOptions{}
	opts.App.AppID = "test"
	opts.App.Token = "test"
	opts.User.UID = "test"
	opts.Request.Text = "test"

	buf, err := encodeVoiceCloneRequest(opts)
	require.NoError(t, err)
	require.Equal(t, voiceCloneClientHeader, buf[:4])
	size := binary.BigEndian.Uint32(buf[4:8])
	require.Len(t, buf, 8+int(size))

	buf, err = decompressGzip(buf[8:])
	require.NoError(t, err)

	var opts2 voiceCloneOptions
	err = json.Unmarshal(buf, &opts2)
	require.NoError(t, err)

	require.Equal(t, opts, &opts2)
}
