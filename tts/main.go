package tts

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/lang/fastrand"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// speakers
// 请根据大模型音色列表更新，请见：https://www.volcengine.com/docs/6561/1257544
const (
	SpeakerMeiLiNvYou = "zh_female_meilinvyou_moon_bigtts"
)

// 接口 Endpoint
const (
	Host = "openspeech.bytedance.com"
)

// 账号参数
const (
	fakeAppKey    = "your_app_key"
	fakeAccessKey = "your_access_key"
)

var (
	endpoint   = flag.String("endpoint", "v3/tts/bidirection", "Endpoint path")
	resourceID = flag.String("resource_id", "volc.service_type.10029", "Commodity resource ID")
	appKey     = flag.String("app_key", "your_app_key", "VolcEngine app ID")
	accessKey  = flag.String("access_key", "your_access_key", "Access Token for authorization")
	bidiTTS    = flag.Bool("bidirectional_tts", true, "Use Bidirection TTS or not")
	speaker    = flag.String("speaker", SpeakerMeiLiNvYou, "Speaker to use")
)

var (
	protocol = NewBinaryProtocol()
)

func init() {
	// Initialize binary protocol settings.
	protocol.SetVersion(Version1)
	protocol.SetHeaderSize(HeaderSize4)
	protocol.SetSerialization(SerializationJSON)
	protocol.SetCompression(CompressionNone, nil)
	protocol.ContainsSequence = ContainsSequence
}

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	if *appKey == fakeAppKey || *accessKey == fakeAccessKey {
		glog.Exitf("Please use your valid app key and access key")
	}
	runDemo()
}

func runDemo() {
	// Establish WebSocket connection with server.
	logID := genLogID()
	conn, err := dial(uuid.New().String(), logID)
	if err != nil {
		glog.Exitf("Dial server: %v", err)
	}
	defer conn.Close()

	// Start application level connection.
	if err := startConnection(conn); err != nil {
		glog.Exitf("Start connection: %v", err)
	}

	sessionID := uuid.New().String()
	namespace := "TTS"
	if *bidiTTS {
		namespace = "BidirectionalTTS"
	}
	if err := startTTSSession(conn, sessionID, namespace, &TTSReqParams{
		Speaker: *speaker,
		AudioParams: &AudioParams{
			Format:     "mp3",
			SampleRate: 24000,
		},
	}); err != nil {
		glog.Exitf("Start session: %v", err)
	}

	// Send TTS requests.
	go func() {
		for _, text := range []string{"帮我", "合成", "一个", "音频"} {
			if err := sendTTSMessage(conn, sessionID, text); err != nil {
				glog.Exitf("Send TTS message: %v", err)
			}
		}

		// Tell server all request has been sent.
		if err := finishSession(conn, sessionID); err != nil {
			glog.Exitf("Finish session: %v", err)
		}
	}()

	// Receive TTS responses.
	var audio []byte
loopReceiveAudio:
	for {
		glog.Infof("Waiting for message...")
		msg, err := receiveMessage(conn)
		if err != nil {
			glog.Errorf("Receive message error: %v", err)
			break
		}
		glog.Infof("Receive message: %+v", msg)

		switch msg.Type {
		case MsgTypeFullServer:
			glog.Infof("Receive text message (event=%s, session_id=%s): %s", Event(msg.Event), msg.SessionID, msg.Payload)
			if msg.Event == int32(EventSessionFinished) {
				break loopReceiveAudio
			}

		case MsgTypeAudioOnlyServer:
			glog.Infof("Receive audio message (event=%s): session_id=%s", Event(msg.Event), msg.SessionID)
			audio = append(audio, msg.Payload...)

		case MsgTypeError:
			glog.Exitf("Receive Error message (code=%d): %s", msg.ErrorCode, msg.Payload)

		default:
			glog.Exitf("Received unexpected message type: %s", msg.Type)
		}
	}
	if len(audio) > 0 {
		if err := os.WriteFile("audio.mp3", audio, 0644); err != nil {
			glog.Exitf("Save audio file: %v", err)
		}
		glog.Info("Session finished, audio is saved.")
	} else {
		glog.Exit("Session finished, no audio data is received.")
	}

	if err := finishConnection(conn); err != nil {
		glog.Exitf("Finish connection: %v", err)
	}
	glog.Info("Connection finished.")
}

func dial(connID, logID string) (*websocket.Conn, error) {
	addr := fmt.Sprintf("wss://%s/api/%s", Host, *endpoint)
	header := buildHTTPHeader(connID, logID)
	glog.Infof("with logID: %s , header: %+v", logID, header)
	conn, r, connErr := websocket.DefaultDialer.DialContext(context.Background(), addr, header)
	if connErr != nil {
		if r != nil {
			body, parseErr := io.ReadAll(r.Body)
			if parseErr != nil {
				parseErr = fmt.Errorf("parse response body failed: %w", parseErr)
				body = []byte(parseErr.Error())
			}
			connErr = fmt.Errorf("[code=%s] [body=%s] %w", r.Status, body, connErr)
		}
		return nil, connErr
	}
	glog.Infof("Dial server with LogID: %s", logID)
	return conn, nil
}

func buildHTTPHeader(connID, logID string) http.Header {
	h := http.Header{
		"X-Tt-Logid":        []string{logID},
		"X-Api-Resource-Id": []string{*resourceID},
		"X-Api-Access-Key":  []string{*accessKey},
		"X-Api-App-Key":     []string{*appKey},
		"X-Api-Connect-Id":  []string{connID},
	}
	return h
}

func startConnection(conn *websocket.Conn) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("create StartSession request message: %w", err)
	}
	msg.Event = int32(EventStartConnection)
	msg.Payload = []byte("{}")

	frame, err := protocol.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal StartConnection request message: %w", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("send StartConnection request: %w", err)
	}

	// Read ConnectionStarted message.
	mt, frame, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read ConnectionStarted response: %w", err)
	}
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return fmt.Errorf("unexpected Websocket message type: %d", mt)
	}

	msg, _, err = Unmarshal(frame, protocol.ContainsSequence)
	if err != nil {
		glog.Infof("StartConnection response: %s", frame)
		return fmt.Errorf("unmarshal ConnectionStarted response message: %w", err)
	}
	if msg.Type != MsgTypeFullServer {
		return fmt.Errorf("unexpected ConnectionStarted message type: %s", msg.Type)
	}
	if Event(msg.Event) != EventConnectionStarted {
		return fmt.Errorf("unexpected response event (%s) for StartConnection request", Event(msg.Event))
	}
	glog.Infof("Connection started (event=%s) connectID: %s, payload: %s", Event(msg.Event), msg.ConnectID, msg.Payload)

	return nil
}

func startTTSSession(conn *websocket.Conn, sessionID, namespace string, params *TTSReqParams) error {
	req := TTSRequest{
		Event:     int32(EventStartSession),
		Namespace: namespace,
		ReqParams: params,
	}
	payload, err := json.Marshal(&req)
	glog.Infof("StartSession request payload: %s", string(payload))
	if err != nil {
		return fmt.Errorf("marshal StartSession request payload: %w", err)
	}

	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("create StartSession request message: %w", err)
	}
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame, err := protocol.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal StartSession request message: %w", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("send StartSession request: %w", err)
	}

	// Read SessionStarted message.
	mt, frame, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read SessionStarted response: %w", err)
	}
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return fmt.Errorf("unexpected Websocket message type: %d", mt)
	}

	// Validate SessionStarted message.
	msg, _, err = Unmarshal(frame, protocol.ContainsSequence)
	if err != nil {
		glog.Infof("StartSession response: %s", frame)
		return fmt.Errorf("unmarshal SessionStarted response message: %w", err)
	}
	if msg.Type != MsgTypeFullServer {
		return fmt.Errorf("unexpected SessionStarted message type: %s", msg.Type)
	}
	if Event(msg.Event) != EventSessionStarted {
		return fmt.Errorf("unexpected response event (%s) for StartSession request", Event(msg.Event))
	}
	glog.Infof("%s session started with ID: %s", namespace, msg.SessionID)

	return nil
}

func sendTTSMessage(conn *websocket.Conn, sessionID, text string) error {
	req := TTSRequest{
		Event:     int32(EventTaskRequest),
		Namespace: "BidirectionalTTS",
		ReqParams: &TTSReqParams{
			Text:    text,
			Speaker: *speaker,
			AudioParams: &AudioParams{
				Format:     "mp3",
				SampleRate: 24000,
			},
		},
	}
	payload, err := json.Marshal(&req)
	if err != nil {
		return fmt.Errorf("marshal TaskRequest request payload: %w", err)
	}

	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("create TaskRequest request message: %w", err)
	}
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame, err := protocol.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal TaskRequest request message: %w", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("send TaskRequest request: %w", err)
	}

	glog.Info("TaskRequest request is sent.")
	return nil
}

func finishSession(conn *websocket.Conn, sessionID string) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("create FinishSession request message: %w", err)
	}
	msg.Event = int32(EventFinishSession)
	msg.SessionID = sessionID
	msg.Payload = []byte("{}")

	frame, err := protocol.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal FinishSession request message: %w", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("send FinishSession request: %w", err)
	}

	glog.Info("FinishSession request is sent.")
	return nil
}

func receiveMessage(conn *websocket.Conn) (*Message, error) {
	mt, frame, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return nil, fmt.Errorf("unexpected Websocket message type: %d", mt)
	}

	msg, _, err := Unmarshal(frame, ContainsSequence)
	if err != nil {
		if len(frame) > 500 {
			frame = frame[:500]
		}
		glog.Infof("Data response: %s", frame)
		return nil, fmt.Errorf("unmarshal response message: %w", err)
	}
	return msg, nil
}

func finishConnection(conn *websocket.Conn) error {
	msg, err := NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent)
	if err != nil {
		return fmt.Errorf("create FinishConnection request message: %w", err)
	}
	msg.Event = int32(EventFinishConnection)
	msg.Payload = []byte("{}")

	frame, err := protocol.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal FinishConnection request message: %w", err)
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("send FinishConnection request: %w", err)
	}

	// Read ConnectionStarted message.
	mt, frame, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read ConnectionFinished response: %w", err)
	}
	if mt != websocket.BinaryMessage && mt != websocket.TextMessage {
		return fmt.Errorf("unexpected Websocket message type: %d", mt)
	}

	msg, _, err = Unmarshal(frame, protocol.ContainsSequence)
	if err != nil {
		glog.Infof("FinishConnection response: %s", frame)
		return fmt.Errorf("unmarshal ConnectionFinished response message: %w", err)
	}
	if msg.Type != MsgTypeFullServer {
		return fmt.Errorf("unexpected ConnectionFinished message type: %s", msg.Type)
	}
	if Event(msg.Event) != EventConnectionFinished {
		return fmt.Errorf("unexpected response event (%s) for FinishConnection request", Event(msg.Event))
	}

	glog.Infof("Connection finished (event=%s)", Event(msg.Event))
	return nil
}

func genLogID() string {
	const (
		maxRandNum = 1<<24 - 1<<20
		length     = 53
		version    = "02"
		localIP    = "00000000000000000000000000000000"
	)
	ts := uint64(time.Now().UnixNano() / int64(time.Millisecond))
	r := uint64(fastrand.Uint32n(maxRandNum) + 1<<20)
	var sb strings.Builder
	sb.Grow(length)
	sb.WriteString(version)
	sb.WriteString(strconv.FormatUint(ts, 10))
	sb.WriteString(localIP)
	sb.WriteString(strconv.FormatUint(r, 16))
	return sb.String()
}
