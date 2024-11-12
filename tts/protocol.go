package tts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/glog"
)

type Event int32

const (
	// 默认事件,对于使用事件的方案，可以通过非0值来校验事件的合法性
	EventNone Event = 0

	// 上行Connection事件
	EventStartConnection  Event = 1
	EventFinishConnection Event = 2

	// 下行Connection事件
	EventConnectionStarted  Event = 50 // 成功建连
	EventConnectionFailed   Event = 51 // 建连失败（可能是无法通过权限认证）
	EventConnectionFinished Event = 52 // 连接结束

	// 上行Session事件
	EventStartSession  Event = 100
	EventFinishSession Event = 102

	// 下行Session事件
	EventSessionStarted  Event = 150
	EventSessionFinished Event = 152
	EventSessionFailed   Event = 153

	// 上行通用事件
	EventTaskRequest Event = 200

	// 下行TTS事件
	EventTTSSentenceStart Event = 350
	EventTTSSentenceEnd   Event = 351
	EventTTSResponse      Event = 352
)

var (
	eventName = map[int32]string{
		0:   "None",
		1:   "StartConnection",
		2:   "FinishConnection",
		50:  "ConnectionStarted",
		51:  "ConnectionFailed",
		52:  "ConnectionFinished",
		100: "StartSession",
		101: "CancelSession",
		102: "FinishSession",
		150: "SessionStarted",
		151: "SessionCanceled",
		152: "SessionFinished",
		153: "SessionFailed",
		200: "TaskRequest",
		350: "TTSSentenceStart",
		351: "TTSSentenceEnd",
		352: "TTSResponse",
	}
)

func (e Event) String() string {
	return eventName[int32(e)]
}

type TTSRequest struct {
	User *TTSUser `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	// Similar to TTSResponse.event field.
	Event     int32         `protobuf:"varint,2,opt,name=event,proto3" json:"event,omitempty"`
	Namespace string        `protobuf:"bytes,3,opt,name=namespace,proto3" json:"namespace,omitempty"`
	ReqParams *TTSReqParams `protobuf:"bytes,10,opt,name=req_params,json=reqParams,proto3" json:"req_params,omitempty"`
}

type TTSUser struct {
	Uid            string `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	Did            string `protobuf:"bytes,2,opt,name=did,proto3" json:"did,omitempty"`
	DevicePlatform string `protobuf:"bytes,3,opt,name=device_platform,json=devicePlatform,proto3" json:"device_platform,omitempty"`
	DeviceType     string `protobuf:"bytes,4,opt,name=device_type,json=deviceType,proto3" json:"device_type,omitempty"`
	VersionCode    string `protobuf:"bytes,5,opt,name=version_code,json=versionCode,proto3" json:"version_code,omitempty"`
	Language       string `protobuf:"bytes,6,opt,name=language,proto3" json:"language,omitempty"`
}

type TTSReqParams struct {
	Text           string            `protobuf:"bytes,1,opt,name=text,proto3" json:"text,omitempty"`
	Texts          *Texts            `protobuf:"bytes,2,opt,name=texts,proto3" json:"texts,omitempty"`
	Ssml           string            `protobuf:"bytes,3,opt,name=ssml,proto3" json:"ssml,omitempty"`
	Speaker        string            `protobuf:"bytes,4,opt,name=speaker,proto3" json:"speaker,omitempty"`
	AudioParams    *AudioParams      `protobuf:"bytes,5,opt,name=audio_params,json=audioParams,proto3" json:"audio_params,omitempty"`
	EngineParams   *EngineParams     `protobuf:"bytes,6,opt,name=engine_params,json=engineParams,proto3" json:"engine_params,omitempty"`
	EnableAudio2Bs bool              `protobuf:"varint,7,opt,name=enable_audio2bs,json=enableAudio2bs,proto3" json:"enable_audio2bs,omitempty"`
	EnableTextSeg  bool              `protobuf:"varint,8,opt,name=enable_text_seg,json=enableTextSeg,proto3" json:"enable_text_seg,omitempty"`
	Additions      map[string]string `protobuf:"bytes,100,rep,name=additions,proto3" json:"additions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

type Texts struct {
	Texts []string `protobuf:"bytes,1,rep,name=texts,proto3" json:"texts,omitempty"`
}

type AudioParams struct {
	Format          string `protobuf:"bytes,1,opt,name=format,proto3" json:"format,omitempty"`
	SampleRate      int32  `protobuf:"varint,2,opt,name=sample_rate,json=sampleRate,proto3" json:"sample_rate,omitempty"`
	Channel         int32  `protobuf:"varint,3,opt,name=channel,proto3" json:"channel,omitempty"`
	SpeechRate      int32  `protobuf:"varint,4,opt,name=speech_rate,json=speechRate,proto3" json:"speech_rate,omitempty"`
	PitchRate       int32  `protobuf:"varint,5,opt,name=pitch_rate,json=pitchRate,proto3" json:"pitch_rate,omitempty"`
	BitRate         int32  `protobuf:"varint,6,opt,name=bit_rate,json=bitRate,proto3" json:"bit_rate,omitempty"`
	Volume          int32  `protobuf:"varint,7,opt,name=volume,proto3" json:"volume,omitempty"`
	Lang            string `protobuf:"bytes,8,opt,name=lang,proto3" json:"lang,omitempty"`
	Emotion         string `protobuf:"bytes,9,opt,name=emotion,proto3" json:"emotion,omitempty"`
	Gender          string `protobuf:"bytes,10,opt,name=gender,proto3" json:"gender,omitempty"`
	EnableTimestamp bool   `protobuf:"varint,11,opt,name=enable_timestamp,json=enableTimestamp,proto3" json:"enable_timestamp,omitempty"`
}

type EngineParams struct {
	EngineContext                string   `protobuf:"bytes,1,opt,name=engine_context,json=engineContext,proto3" json:"engine_context,omitempty"`
	PhonemeSize                  string   `protobuf:"bytes,2,opt,name=phoneme_size,json=phonemeSize,proto3" json:"phoneme_size,omitempty"`
	EnableFastTextSeg            bool     `protobuf:"varint,3,opt,name=enable_fast_text_seg,json=enableFastTextSeg,proto3" json:"enable_fast_text_seg,omitempty"`
	ForceBreak                   bool     `protobuf:"varint,4,opt,name=force_break,json=forceBreak,proto3" json:"force_break,omitempty"`
	BreakByProsody               int32    `protobuf:"varint,5,opt,name=break_by_prosody,json=breakByProsody,proto3" json:"break_by_prosody,omitempty"`
	EnableEngineDebugInfo        bool     `protobuf:"varint,6,opt,name=enable_engine_debug_info,json=enableEngineDebugInfo,proto3" json:"enable_engine_debug_info,omitempty"`
	FlushSentence                bool     `protobuf:"varint,7,opt,name=flush_sentence,json=flushSentence,proto3" json:"flush_sentence,omitempty"`
	LabVersion                   string   `protobuf:"bytes,8,opt,name=lab_version,json=labVersion,proto3" json:"lab_version,omitempty"`
	EnableIpaExtraction          bool     `protobuf:"varint,9,opt,name=enable_ipa_extraction,json=enableIpaExtraction,proto3" json:"enable_ipa_extraction,omitempty"`
	EnableNaiveTn                bool     `protobuf:"varint,10,opt,name=enable_naive_tn,json=enableNaiveTn,proto3" json:"enable_naive_tn,omitempty"`
	EnableLatexTn                bool     `protobuf:"varint,11,opt,name=enable_latex_tn,json=enableLatexTn,proto3" json:"enable_latex_tn,omitempty"`
	DisableNewlineStrategy       bool     `protobuf:"varint,12,opt,name=disable_newline_strategy,json=disableNewlineStrategy,proto3" json:"disable_newline_strategy,omitempty"`
	SupportedLanguages           []string `protobuf:"bytes,13,rep,name=supported_languages,json=supportedLanguages,proto3" json:"supported_languages,omitempty"`
	ContextLanguage              string   `protobuf:"bytes,14,opt,name=context_language,json=contextLanguage,proto3" json:"context_language,omitempty"`
	ContextTexts                 []string `protobuf:"bytes,15,rep,name=context_texts,json=contextTexts,proto3" json:"context_texts,omitempty"`
	EnableRecoverPuncts          bool     `protobuf:"varint,16,opt,name=enable_recover_puncts,json=enableRecoverPuncts,proto3" json:"enable_recover_puncts,omitempty"`
	EosProsody                   int32    `protobuf:"varint,17,opt,name=eos_prosody,json=eosProsody,proto3" json:"eos_prosody,omitempty"`
	PrependSilenceSeconds        float64  `protobuf:"fixed64,18,opt,name=prepend_silence_seconds,json=prependSilenceSeconds,proto3" json:"prepend_silence_seconds,omitempty"`
	MaxParagraphPhonemeSize      int32    `protobuf:"varint,19,opt,name=max_paragraph_phoneme_size,json=maxParagraphPhonemeSize,proto3" json:"max_paragraph_phoneme_size,omitempty"`
	ParagraphSubSentences        []string `protobuf:"bytes,20,rep,name=paragraph_sub_sentences,json=paragraphSubSentences,proto3" json:"paragraph_sub_sentences,omitempty"`
	MaxLengthToFilterParenthesis int32    `protobuf:"varint,21,opt,name=max_length_to_filter_parenthesis,json=maxLengthToFilterParenthesis,proto3" json:"max_length_to_filter_parenthesis,omitempty"`
	EnableLanguageDetector       bool     `protobuf:"varint,22,opt,name=enable_language_detector,json=enableLanguageDetector,proto3" json:"enable_language_detector,omitempty"`
}

var (
	errNoVersionAndSize              = errors.New("no protocol version and header size byte")
	errNoTypeAndFlag                 = errors.New("no message type and specific flag byte")
	errNoSerializationAndCompression = errors.New("no serialization and compression method byte")
	errRedundantBytes                = errors.New("there are redundant bytes in data")
	errInvalidMessageType            = errors.New("invalid message type bits")
	errInvalidSerialization          = errors.New("invalid serialization bits")
	errInvalidCompression            = errors.New("invalid compression bits")
	errNoEnoughHeaderBytes           = errors.New("no enough header bytes")
	errReadEvent                     = errors.New("read event number")
	errReadSessionIDSize             = errors.New("read session ID size")
	errReadConnectIDSize             = errors.New("read connection ID size")
	errReadPayloadSize               = errors.New("read payload size")
	errReadPayload                   = errors.New("read payload")
	errReadSequence                  = errors.New("read sequence number")
	errReadErrorCode                 = errors.New("read error code")
	errReadErrorSize                 = errors.New("read error size")
	errReadError                     = errors.New("read error")
)

type (
	// MsgType defines message type which determines how the message will be
	// serialized with the protocol.
	MsgType int32
	// MsgTypeFlagBits defines the 4-bit message-type specific flags. The specific
	// values should be defined in each specific usage scenario.
	MsgTypeFlagBits uint8

	// VersionBits defines the 4-bit version type.
	VersionBits uint8
	// HeaderSizeBits defines the 4-bit header-size type.
	HeaderSizeBits uint8
	// SerializationBits defines the 4-bit serialization method type.
	SerializationBits uint8
	// CompressionBits defines the 4-bit compression method type.
	CompressionBits uint8
)

// Values that a MsgType variable can take.
const (
	MsgTypeInvalid MsgType = iota
	MsgTypeFullClient
	MsgTypeAudioOnlyClient
	MsgTypeFullServer
	MsgTypeAudioOnlyServer
	MsgTypeFrontEndResultServer
	MsgTypeError

	MsgTypeServerACK = MsgTypeAudioOnlyServer
)

func (t MsgType) String() string {
	switch t {
	case MsgTypeFullClient:
		return "FullClient"
	case MsgTypeAudioOnlyClient:
		return "AudioOnlyClient"
	case MsgTypeFullServer:
		return "FullServer"
	case MsgTypeAudioOnlyServer:
		return "AudioOnlyServer/ServerACK"
	case MsgTypeError:
		return "Error"
	case MsgTypeFrontEndResultServer:
		return "TtsFrontEndResult"
	default:
		return fmt.Sprintf("invalid message type: %d", t)
	}
}

// Values that a MsgTypeFlagBits variable can take.
const (
	// For common protocol.
	MsgTypeFlagNoSeq       MsgTypeFlagBits = 0     // Non-terminal packet with no sequence
	MsgTypeFlagPositiveSeq MsgTypeFlagBits = 0b1   // Non-terminal packet with sequence > 0
	MsgTypeFlagLastNoSeq   MsgTypeFlagBits = 0b10  // last packet with no sequence
	MsgTypeFlagNegativeSeq MsgTypeFlagBits = 0b11  // last packet with sequence < 0
	MsgTypeFlagWithEvent   MsgTypeFlagBits = 0b100 // Payload contains event number (int32)
)

// Values that a VersionBits variable can take.
const (
	Version1 VersionBits = (iota + 1) << 4
	Version2
	Version3
	Version4
)

// Values that a HeaderSizeBits variable can take.
const (
	HeaderSize4 HeaderSizeBits = iota + 1
	HeaderSize8
	HeaderSize12
	HeaderSize16
)

// Values that a SerializationBits variable can take.
const (
	SerializationRaw    SerializationBits = 0
	SerializationJSON   SerializationBits = 0b1 << 4
	SerializationThrift SerializationBits = 0b11 << 4
	SerializationCustom SerializationBits = 0b1111 << 4
)

// Values that a CompressionBits variable can take.
const (
	CompressionNone   CompressionBits = 0
	CompressionGzip   CompressionBits = 0b1
	CompressionCustom CompressionBits = 0b1111
)

var (
	msgTypeToBits = map[MsgType]uint8{
		MsgTypeFullClient:           0b1 << 4,
		MsgTypeAudioOnlyClient:      0b10 << 4,
		MsgTypeFullServer:           0b1001 << 4,
		MsgTypeAudioOnlyServer:      0b1011 << 4,
		MsgTypeFrontEndResultServer: 0b1100 << 4,
		MsgTypeError:                0b1111 << 4,
	}
	bitsToMsgType = make(map[uint8]MsgType, len(msgTypeToBits))

	serializations = map[SerializationBits]bool{
		SerializationRaw:    true,
		SerializationJSON:   true,
		SerializationThrift: true,
		SerializationCustom: true,
	}

	compressions = map[CompressionBits]bool{
		CompressionNone:   true,
		CompressionGzip:   true,
		CompressionCustom: true,
	}
)

func init() {
	// Construct inverse mapping of msgTypeToBits.
	for msgType, bits := range msgTypeToBits {
		bitsToMsgType[bits] = msgType
	}
}

// ContainsSequenceFunc defines the functional type that checks whether the
// MsgTypeFlagBits indicates the existence of a sequence number in serialized
// data. The background is that not all responses contain a sequence number,
// and whether a response contains one depends on the the message type specific
// flag bits. What makes it more complicated is that this dependency varies in
// each use case (eg, TTS protocol has its own dependency specification, more
// details at: https://bytedance.feishu.cn/docs/doccn8MD4cZHQuvobbtouWfUVsV).
type ContainsSequenceFunc func(MsgTypeFlagBits) bool

// CompressFunc defines the functional type that does the compression operation.
type CompressFunc func([]byte) ([]byte, error)

type readFunc func(*bytes.Buffer) error
type writeFunc func(*bytes.Buffer) error

// Unmarshal deserializes the binary `data` into a Message and also returns
// the BinaryProtocol.
func Unmarshal(data []byte, containsSequence ContainsSequenceFunc) (*Message, *BinaryProtocol, error) {
	var (
		buf      = bytes.NewBuffer(data)
		readSize int
	)

	versionSize, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoVersionAndSize
	}
	readSize++

	prot := &BinaryProtocol{
		versionAndHeaderSize: versionSize,
		ContainsSequence:     containsSequence,
	}
	glog.V(2).Infof("Read version: %04b", versionSize>>4)
	glog.V(2).Infof("Read size: %04b", versionSize&0b1111)

	typeAndFlag, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoTypeAndFlag
	}
	readSize++
	glog.V(2).Infof("Read message type: %04b", typeAndFlag>>4)
	glog.V(2).Infof("Read message type specific flag: %04b", typeAndFlag&0b1111)

	msg, err := NewMessageFromByte(typeAndFlag)
	if err != nil {
		return nil, nil, err
	}

	serializationCompression, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoSerializationAndCompression
	}
	glog.V(2).Infof("Read serialization method: %04b", serializationCompression>>4)
	glog.V(2).Infof("Read compression method: %04b", serializationCompression&0b1111)
	readSize++
	prot.serializationAndCompression = serializationCompression
	if _, ok := serializations[prot.Serialization()]; !ok {
		return nil, nil, fmt.Errorf("%w: %b", errInvalidSerialization, prot.Serialization())
	}
	if _, ok := compressions[prot.Compression()]; !ok {
		return nil, nil, fmt.Errorf("%w: %b", errInvalidCompression, prot.Compression())
	}
	// TODO(lucas): handle compressed payload.

	// Read all the remaining zero-padding bytes in the header.
	if paddingSize := prot.HeaderSize() - readSize; paddingSize > 0 {
		if n, err := buf.Read(make([]byte, paddingSize)); err != nil || n < paddingSize {
			return nil, nil, fmt.Errorf("%w: %d", errNoEnoughHeaderBytes, n)
		}
	}

	readers, err := msg.readers(containsSequence)
	if err != nil {
		return nil, nil, err
	}
	for _, read := range readers {
		if err := read(buf); err != nil {
			return nil, nil, err
		}
	}

	if _, err := buf.ReadByte(); err != io.EOF {
		return nil, nil, errRedundantBytes
	}
	return msg, prot, nil
}

// Message defines the general message content type.
type Message struct {
	Type            MsgType
	typeAndFlagBits uint8

	Event     int32
	SessionID string
	ConnectID string
	Sequence  int32
	ErrorCode uint32
	// Raw payload (not Gzip compressed). BinaryProtocol.Marshal will do the
	// compression for you.
	Payload []byte
}

// NewMessage returns a new Message instance of the given message type with the
// specific flag.
func NewMessage(msgType MsgType, typeFlag MsgTypeFlagBits) (*Message, error) {
	bits, ok := msgTypeToBits[msgType]
	if !ok {
		return nil, fmt.Errorf("invalid message type: %d", msgType)
	}
	return &Message{
		Type:            msgType,
		typeAndFlagBits: bits + uint8(typeFlag),
	}, nil
}

// NewMessageFromByte reads the byte as the message type and specific flag bits
// and composes a new Message instance from them.
func NewMessageFromByte(typeAndFlag byte) (*Message, error) {
	bits := typeAndFlag &^ 0b00001111
	msgType, ok := bitsToMsgType[bits]
	if !ok {
		return nil, fmt.Errorf("%w: %b", errInvalidMessageType, bits>>4)
	}
	return &Message{
		Type:            msgType,
		typeAndFlagBits: typeAndFlag,
	}, nil
}

// TypeFlag returns the message type specific flag.
func (m *Message) TypeFlag() MsgTypeFlagBits {
	return MsgTypeFlagBits(m.typeAndFlagBits &^ 0b11110000)
}

func (m *Message) writers(shouldHaveSequence ContainsSequenceFunc, compress CompressFunc) (writers []writeFunc, _ error) {
	if compress != nil {
		payload, err := compress(m.Payload)
		if err != nil {
			return nil, fmt.Errorf("compress payload failed: %w", err)
		}
		m.Payload = payload
	}

	if containsEvent(m.TypeFlag()) {
		writers = append(writers, m.writeEvent, m.writeSessionID)
		glog.V(1).Info("Add Event and SessionID writer.")
	}

	switch m.Type {
	case MsgTypeFullClient, MsgTypeFullServer, MsgTypeFrontEndResultServer:

	case MsgTypeAudioOnlyClient:
		if shouldHaveSequence == nil || shouldHaveSequence(m.TypeFlag()) {
			writers = append(writers, m.writeSequence)
			glog.V(1).Info("AudioOnlyClient message: add Sequence writer.")
		}

	case MsgTypeAudioOnlyServer:
		if shouldHaveSequence == nil || shouldHaveSequence(m.TypeFlag()) {
			writers = append(writers, m.writeSequence)
			glog.V(1).Info("AudioOnlyServer message: add Sequence writer.")
		}

	case MsgTypeError:
		writers = append(writers, m.writeErrorCode)
		glog.V(1).Info("Error message: add Error-Code writer.")

	default:
		return nil, fmt.Errorf("cannot serialize message with invalid type: %d", m.Type)
	}

	writers = append(writers, m.writePayload)
	glog.V(1).Info("Add Payload writers.")
	return writers, nil
}

func (m *Message) writeEvent(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.Event); err != nil {
		return fmt.Errorf("write sequence number (%d): %w", m.Event, err)
	}
	return nil
}

func (m *Message) writeSessionID(buf *bytes.Buffer) error {
	switch e := Event(m.Event); e {
	case EventStartConnection, EventFinishConnection,
		EventConnectionStarted, EventConnectionFailed:
		glog.V(1).Infof("Skip writing session ID for event: %s", e)
		return nil
	}

	size := len(m.SessionID)
	if size > math.MaxUint32 {
		return fmt.Errorf("payload size (%d) exceeds max(uint32)", size)
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(size)); err != nil {
		return fmt.Errorf("write payload size (%d): %w", size, err)
	}
	buf.WriteString(m.SessionID)
	return nil
}

func (m *Message) writeSequence(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.Sequence); err != nil {
		return fmt.Errorf("write sequence number (%d): %w", m.Sequence, err)
	}
	return nil
}

func (m *Message) writeErrorCode(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.ErrorCode); err != nil {
		return fmt.Errorf("write error code (%d): %w", m.ErrorCode, err)
	}
	return nil
}

func (m *Message) writePayload(buf *bytes.Buffer) error {
	size := len(m.Payload)
	if size > math.MaxUint32 {
		return fmt.Errorf("payload size (%d) exceeds max(uint32)", size)
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(size)); err != nil {
		return fmt.Errorf("write payload size (%d): %w", size, err)
	}
	buf.Write(m.Payload)
	return nil
}

func (m *Message) readers(containsSequence ContainsSequenceFunc) (readers []readFunc, _ error) {

	switch m.Type {
	case MsgTypeFullClient, MsgTypeFullServer, MsgTypeFrontEndResultServer:

	case MsgTypeAudioOnlyClient:
		if containsSequence == nil || containsSequence(m.TypeFlag()) {
			readers = append(readers, m.readSequence)
			glog.V(1).Info("AudioOnlyClient message: add Sequence reader.")
		}

	case MsgTypeAudioOnlyServer:
		if containsSequence != nil && containsSequence(m.TypeFlag()) {
			readers = append(readers, m.readSequence)
			glog.V(1).Info("AudioOnlyServer message: add Sequence reader.")
		}

	case MsgTypeError:
		readers = append(readers, m.readErrorCode)
		glog.V(1).Info("Error message: add Error-Code reader.")

	default:
		return nil, fmt.Errorf("cannot deserialize message with invalid type: %d", m.Type)
	}

	if containsEvent(m.TypeFlag()) {
		readers = append(readers, m.readEvent, m.readSessionID, m.readConnectID)
		glog.V(1).Info("Add Event and SessionID readers.")
	}

	readers = append(readers, m.readPayload)
	glog.V(1).Info("Add Payload reader.")
	return readers, nil
}

func (m *Message) readEvent(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.Event); err != nil {
		return fmt.Errorf("%w: %v", errReadEvent, err)
	}
	glog.V(2).Infof("Read Event: %s", Event(m.Event))
	return nil
}

func (m *Message) readSessionID(buf *bytes.Buffer) error {
	switch e := Event(m.Event); e {
	case EventStartConnection, EventFinishConnection,
		EventConnectionStarted, EventConnectionFailed,
		EventConnectionFinished:
		glog.V(1).Infof("Skip reading session ID for event: %s", e)
		return nil
	}

	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadSessionIDSize, err)
	}
	glog.V(2).Infof("Read SessionID length: %d", size)

	if size > 0 {
		m.SessionID = string(buf.Next(int(size)))
	}
	glog.V(2).Infof("Read SessionID content: %s", m.SessionID)
	return nil
}

func (m *Message) readConnectID(buf *bytes.Buffer) error {
	switch e := Event(m.Event); e {
	case EventConnectionStarted, EventConnectionFailed,
		EventConnectionFinished:
	default:
		glog.V(1).Infof("Skip reading session ID for event: %s", e)
		return nil
	}

	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadConnectIDSize, err)
	}
	glog.V(2).Infof("Read connection ID length: %d", size)

	if size > 0 {
		m.ConnectID = string(buf.Next(int(size)))
	}
	glog.V(2).Infof("Read connection ID content: %s", m.ConnectID)
	return nil
}

func (m *Message) readSequence(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.Sequence); err != nil {
		return fmt.Errorf("%w: %v", errReadSequence, err)
	}
	glog.V(2).Infof("Read Sequence: %d", m.Sequence)
	return nil
}

func (m *Message) readErrorCode(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.ErrorCode); err != nil {
		return fmt.Errorf("%w: %v", errReadErrorCode, err)
	}
	glog.V(2).Infof("Read ErrorCode: %d", m.ErrorCode)
	return nil
}

func (m *Message) readPayload(buf *bytes.Buffer) error {
	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadPayloadSize, err)
	}
	glog.V(2).Infof("Read Payload length: %d", size)

	if size > 0 {
		m.Payload = buf.Next(int(size))
	}
	if m.Type == MsgTypeFullClient || m.Type == MsgTypeFullServer || m.Type == MsgTypeError {
		glog.V(2).Infof("Read Payload content: %s", m.Payload)
	}
	return nil
}

// ContainsSequence reports whether a message type specific flag indicates
// messages with this kind of flag contain a sequence number in its serialized
// value. This determiner function should be used for common binary protocol.
func ContainsSequence(bits MsgTypeFlagBits) bool {
	return bits == MsgTypeFlagPositiveSeq || bits == MsgTypeFlagNegativeSeq
}

func containsEvent(bits MsgTypeFlagBits) bool {
	return bits == MsgTypeFlagWithEvent
}

// BinaryProtocol implements the binary protocol serialization and deserialization
// used in Lab-Speech MDD, TTS, ASR, etc. services. For more details, read:
// https://bytedance.feishu.cn/docs/doccnT0t71J4LCQCS0cnB4Eca8D
type BinaryProtocol struct {
	versionAndHeaderSize        uint8
	serializationAndCompression uint8

	ContainsSequence ContainsSequenceFunc
	Compress         CompressFunc
}

// NewBinaryProtocol returns a new BinaryProtocol instance.
func NewBinaryProtocol() *BinaryProtocol {
	return new(BinaryProtocol)
}

// Clone returns a clone of current BinaryProtocol
func (p *BinaryProtocol) Clone() *BinaryProtocol {
	clonedBinaryProtocal := new(BinaryProtocol)
	clonedBinaryProtocal.versionAndHeaderSize = p.versionAndHeaderSize
	clonedBinaryProtocal.serializationAndCompression = p.serializationAndCompression
	clonedBinaryProtocal.ContainsSequence = p.ContainsSequence
	clonedBinaryProtocal.Compress = p.Compress
	return clonedBinaryProtocal
}

// SetVersion sets the protocol version.
func (p *BinaryProtocol) SetVersion(v VersionBits) {
	// Clear the higher 4 bits in `p.versionAndHeaderSize` and reset them to `v`.
	p.versionAndHeaderSize = (p.versionAndHeaderSize &^ 0b11110000) + uint8(v)
}

// Version returns the integral version value.
func (p *BinaryProtocol) Version() int {
	return int(p.versionAndHeaderSize >> 4)
}

// SetHeaderSize sets the protocol header size.
func (p *BinaryProtocol) SetHeaderSize(s HeaderSizeBits) {
	// Clear the lower 4 bits in `p.versionAndHeaderSize` and reset them to `s`.
	p.versionAndHeaderSize = (p.versionAndHeaderSize &^ 0b00001111) + uint8(s)
}

// HeaderSize returns the protocol header size.
func (p *BinaryProtocol) HeaderSize() int {
	return 4 * int(p.versionAndHeaderSize&^0b11110000)
}

// SetSerialization sets the serialization method.
func (p *BinaryProtocol) SetSerialization(s SerializationBits) {
	// Clear the higher 4 bits in `p.serializationAndCompression` and reset them to `s`.
	p.serializationAndCompression = (p.serializationAndCompression &^ 0b11110000) + uint8(s)
}

// Serialization returns the bits value of protocol serialization method.
func (p *BinaryProtocol) Serialization() SerializationBits {
	return SerializationBits(p.serializationAndCompression &^ 0b00001111)
}

// SetCompression sets the compression method.
func (p *BinaryProtocol) SetCompression(c CompressionBits, f CompressFunc) {
	// Clear the lower 4 bits in `p.serializationAndCompression` and reset them to `c`.
	p.serializationAndCompression = (p.serializationAndCompression &^ 0b00001111) + uint8(c)
	p.Compress = f
}

// Compression returns the bits value of protocol compression method.
func (p *BinaryProtocol) Compression() CompressionBits {
	return CompressionBits(p.serializationAndCompression &^ 0b11110000)
}

// Marshal serializes the message to a sequence of binary data.
func (p *BinaryProtocol) Marshal(msg *Message) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := p.writeHeader(buf, msg); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	writers, err := msg.writers(p.ContainsSequence, p.Compress)
	if err != nil {
		return nil, err
	}
	for _, write := range writers {
		if err := write(buf); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (p *BinaryProtocol) writeHeader(buf *bytes.Buffer, msg *Message) error {
	return binary.Write(buf, binary.BigEndian, p.header(msg))
}

func (p *BinaryProtocol) header(msg *Message) []byte {
	header := []uint8{
		p.versionAndHeaderSize,
		msg.typeAndFlagBits,
		p.serializationAndCompression,
	}
	if padding := p.HeaderSize() - len(header); padding > 0 {
		header = append(header, make([]uint8, padding)...)
	}
	return header
}
