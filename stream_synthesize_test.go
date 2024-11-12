package volcvoice

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSteamSynthesizeService(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err)

	requestId, err := uuid.NewV7()
	require.NoError(t, err)

	f, err := os.OpenFile("test.pcm", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	require.NoError(t, err)
	defer f.Close()

	s := client.StreamSynthesize().
		SetInput(`“水何澹澹，山岛竦峙。树木丛生，百草丰茂。秋风萧瑟，洪波涌起”是实写眼前的景观，神奇而又壮观。“水何澹澹，山岛竦峙”是望海初得的大致印象，有点像绘画的轮廓。`).
		SetFormat(FormatPCM).
		SetRequestID(requestId.String()).
		SetCluster(StreamSynthesizeClusterV2).
		SetUserID("test").
		SetSpeakerID(os.Getenv("VOLCVOICE_SPEAKER_ID")).
		SetOutput(func(ctx context.Context, buf []byte) (err error) {
			t.Logf("len(buf): %d", len(buf))
			_, err = f.Write(buf)
			return
		})

	err = s.Do(context.Background())
	require.NoError(t, err)
}
