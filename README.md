# volcvoice

A Go Library for Volcengine (Bytedance) Voice Service

## Notes

**PCM output format**

`16-bit signed integer, little-endian, mono`, default to `24k` sample rate.

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
	SetSpeakerID(os.Getenv("VOLCVOICE_SPEAKER_ID")).
	AddAudio(buf, FormatMP3, "").
	SetModelType(VoiceCloneUploadModelTypeV2)

err := service.Do(context.Background())
```

**Stream Synthesize**

```go
service := client.StreamSynthesize().
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

err := service.Do(context.Background())
```

**Bi-directional Stream Synthesize**

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

service := client.DuplexSynthesize().
	SetResourceID(SynthesizeResourceVoiceClone2).
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
