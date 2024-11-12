package tts

const (
	AudioFormatMP3 = "mp3"
	AudioFormatOGG = "ogg_opus"
	AudioFormatPCM = "pcm"

	SampleRate8K  = 8000
	SampleRate16K = 16000
	SampleRate24K = 24000
	SampleRate32K = 32000
	SampleRate44K = 44100
	SampleRate48K = 48000
)

type streamSynthesizePayload struct {
	User struct {
		UID string `json:"uid,omitempty"`
	} `json:"user"`
	Event     int    `json:"event"`
	Namespace string `json:"namespace,omitempty"`
	ReqParams struct {
		Text        string `json:"text,omitempty"`
		SSML        string `json:"ssml,omitempty"`
		Speaker     string `json:"speaker,omitempty"`
		AudioParams struct {
			Format     string `json:"format,omitempty"`
			SampleRate *int   `json:"sample_rate,omitempty"`
			SpeechRate *int   `json:"speech_rate,omitempty"`
			PitchRate  *int   `json:"pitch_rate,omitempty"`
		} `json:"audio_params"`
	} `json:"req_params"`
}
