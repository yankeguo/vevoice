package volcvoice

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVoiceCloneUpload(t *testing.T) {
	client, err := NewClient(WithVerbose(true))
	require.NoError(t, err)

	buf, err := os.ReadFile(filepath.Join("testdata", "upload.mp3"))
	require.NoError(t, err)

	service := client.VoiceCloneUpload().
		SetSpeakerID(os.Getenv("VOLCVOICE_SPEAKER_ID")).
		AddAudio(buf, FormatMP3, "").
		SetModelType(VoiceCloneUploadModelTypeV2)

	err = service.Do(context.Background())
	require.NoError(t, err)
}
