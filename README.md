# volcvoice

A Go Library for Volcengine (Bytedance) Voice Service

## Notes

**PCM output format**

16-bit signed integer, little-endian, mono

Example ffmpeg command:

```
ffmpeg -f s16le -ar 24k -ac 1 -i test.pcm test.mp3
```

## Usages

**Create Client**

```go
// using VOLCVOICE_APPID and VOLCVOICE_TOKEN
client := volcvoice.NewClient()

// explicit set appid and token
client := volcvoice.NewClient(
    volcvoice.WithAppID("your_app_id"),
    volcvoice.WithToken("your_token"),
)
```

**Upload Voice Clone Sample**

```go
service := client.VoiceCloneUpload().
	SetResourceID(VoiceCloneUploadResource2_0).
	SetSpeakerID(os.Getenv("VOLCVOICE_SPEAKER_ID")).
	AddAudio(buf, AudioFormatMP3, "").
	SetModelType(VoiceCloneModelType2_0)

err := service.Do(context.Background())
```

**Bi-directional stream TTS**

```go
var (
	input = []string{
		"离离原上草，一岁一枯荣。",
		"野火烧不尽，春风吹又生。",
		"远芳侵古道，晴翠接荒城。",
		"又送王孙去，萋萋满别情。",
	}
	inputIdx int64 = -1
)

service := client.Synthesize().
	SetResourceID(SynthesizeResourceVoiceClone2_0).
	SetRequestID(rg.Must(uuid.NewV7()).String()).
	SetConnectID(rg.Must(uuid.NewV7()).String()).
	SetFormat(AudioFormatPCM).
	SetSampleRate(SampleRate16K).
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
	SetUserID("test-user")

err := service.Do(context.Background())
```

## Credits

GUO YANKE, MIT License
