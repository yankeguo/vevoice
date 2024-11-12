package volcvoice

import (
	"context"
	"encoding/base64"
	"fmt"
)

const (
	voiceCloneUploadResource = "volc.megatts.voiceclone"

	VoiceCloneUploadLanguageCN = 0
	VoiceCloneUploadLanguageEN = 1
	VoiceCloneUploadLanguageJA = 2
	VoiceCloneUploadLanguageES = 3
	VoiceCloneUploadLanguageID = 4
	VoiceCloneUploadLanguagePT = 5

	VoiceCloneUploadModelTypeV1 = 0
	VoiceCloneUploadModelTypeV2 = 1
)

type VoiceCloneUploadAudio struct {
	AudioBytes  string `json:"audio_bytes"`
	AudioFormat string `json:"audio_format,omitempty"`
	Text        string `json:"text,omitempty"`
}

type VoiceCloneUploadResponse struct {
	BaseResp struct {
		StatusCode    int    `json:"StatusCode"`
		StatusMessage string `json:"StatusMessage"`
	} `json:"BaseResp"`
	SpeakerID string `json:"speaker_id"`
}

type VoiceCloneUploadService struct {
	c *client

	speakerID string
	audios    []VoiceCloneUploadAudio
	language  *int
	modelType *int
}

func newVoiceCloneUploadService(c *client) *VoiceCloneUploadService {
	s := &VoiceCloneUploadService{
		c: c,
	}
	return s
}

func (s *VoiceCloneUploadService) SetSpeakerID(speakerID string) *VoiceCloneUploadService {
	s.speakerID = speakerID
	return s
}

func (s *VoiceCloneUploadService) AddAudio(buf []byte, format string, text string) *VoiceCloneUploadService {
	s.audios = append(s.audios, VoiceCloneUploadAudio{
		AudioBytes:  base64.StdEncoding.EncodeToString(buf),
		AudioFormat: format,
		Text:        text,
	})
	return s
}

func (s *VoiceCloneUploadService) SetLanguage(language int) *VoiceCloneUploadService {
	s.language = &language
	return s
}

func (s *VoiceCloneUploadService) SetModelType(modelType int) *VoiceCloneUploadService {
	s.modelType = &modelType
	return s
}

func (s *VoiceCloneUploadService) buildBody() map[string]any {
	body := map[string]any{
		"appid":      s.c.appID,
		"speaker_id": s.speakerID,
		"audios":     s.audios,
		"source":     2,
	}
	if s.language != nil {
		body["language"] = *s.language
	}
	if s.modelType != nil {
		body["model_type"] = *s.modelType
	}
	return body
}

func (s *VoiceCloneUploadService) Do(ctx context.Context) (err error) {
	var res VoiceCloneUploadResponse

	if err = s.c.httpPost(
		ctx,
		"/api/v1/mega_tts/audio/upload",
		map[string]string{
			"Authorization": "Bearer;" + s.c.token,
			"Resource-Id":   voiceCloneUploadResource,
		},
		s.buildBody(),
		&res,
	); err != nil {
		return
	}

	if res.BaseResp.StatusCode != 0 {
		err = fmt.Errorf("voice_clone_upload: %d %s", res.BaseResp.StatusCode, res.BaseResp.StatusMessage)
		return
	}

	return
}
