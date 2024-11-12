package volcvoice

import (
	"context"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/yankeguo/rg"
	"github.com/yankeguo/volcvoice/tts"
)

func TestTTS(t *testing.T) {
	client, err := NewClient(WithDebug(true))
	require.NoError(t, err)

	f, err := os.OpenFile("test.pcm", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	require.NoError(t, err)
	defer f.Close()

	var (
		input = []string{
			"离离原上草，一岁一枯荣。",
			"野火烧不尽，春风吹又生。",
			"远芳侵古道，晴翠接荒城。",
			"又送王孙去，萋萋满别情。",
		}
		inputIdx int64 = -1
	)

	err = client.TTS().
		SetResourceID(tts.ResourceVoiceClone2_0).
		SetRequestID(rg.Must(uuid.NewV7()).String()).
		SetConnectID(rg.Must(uuid.NewV7()).String()).
		SetFormat(tts.AudioFormatPCM).
		SetSampleRate(tts.SampleRate16K).
		SetSpeakerID(os.Getenv("VOLCVOICE_SPEAKER_ID")).
		SetInput(func(ctx context.Context) (chunk string, err error) {
			idx := atomic.AddInt64(&inputIdx, 1)
			if idx >= int64(len(input)) {
				err = io.EOF
				return
			}
			time.Sleep(400 * time.Millisecond)
			chunk = input[idx]
			return
		}).
		SetOutput(func(ctx context.Context, chunk []byte) (err error) {
			_, err = f.Write(chunk)
			return
		}).
		SetUserID("test-user").
		Do(context.Background())

	require.NoError(t, err)
}
