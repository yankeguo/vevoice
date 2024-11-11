# volcvoice

A Go Library for Volcengine (Bytedance) Voice Service

## Notes

**PCM output format for Voice Clone**

After testing, the PCM output format for voice clone should be 16-bit signed integer, little-endian, 24kHz, mono.

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

**Voice Clone**

```go
service := client.VoiceClone().
		SetTextType(volcvoice.TextTypePlain).
		SetText(`水何澹澹，山岛竦峙。树木丛生，百草丰茂。秋风萧瑟，洪波涌起`).
		SetEncoding(volcvoice.EncodingPCM).
		SetRequestID("random-request-id").
		SetCluster("volcano_mega"). // check available clusters in the web console page
		SetUID("test").
		SetVoiceType("S_xxxxxx"). // check available voice types (speaker id) in the web console page
		SetHandler(func(buf []byte) (err error) {
			_, err = f.Write(buf)
			return
		})

err := service.Do(context.Background())
```

## Credits

GUO YANKE, MIT License
