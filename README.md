# volcvoice

A Go Library for Volcengine (Bytedance) Voice Service

## Notes

**PCM output format for Voice Clone**

After testing, the PCM output format for voice clone should be 16-bit signed integer, little-endian, 24kHz, mono.

Example ffmpeg command:

```
ffmpeg -f s16le -ar 24k -ac 1 -i test.pcm test.mp3
```

## Credits

GUO YANKE, MIT License