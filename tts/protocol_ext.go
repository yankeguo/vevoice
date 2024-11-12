package tts

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yankeguo/rg"
	"golang.org/x/net/websocket"
)

func (protocol *BinaryProtocol) startConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent))
	msg.Event = int32(EventStartConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(protocol.Marshal(msg))

	rg.Must0(websocket.Message.Send(conn, frame))
	rg.Must0(websocket.Message.Receive(conn, &frame))

	msg, _ = rg.Must2(Unmarshal(frame, protocol.ContainsSequence))

	if msg.Type != MsgTypeFullServer {
		err = fmt.Errorf("tts.startConnection: unexpected message type: %d", msg.Type)
		return
	}

	if Event(msg.Event) != EventConnectionStarted {
		err = fmt.Errorf("tts.startConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (protocol *BinaryProtocol) startTTSSession(ctx context.Context, conn *websocket.Conn, sessionID, namespace string, params *TTSReqParams) (err error) {
	defer rg.Guard(&err)

	req := TTSRequest{
		Event:     int32(EventStartSession),
		Namespace: namespace,
		ReqParams: params,
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(protocol.Marshal(msg))

	rg.Must0(websocket.Message.Send(conn, frame))
	rg.Must0(websocket.Message.Receive(conn, &frame))

	msg, _ = rg.Must2(Unmarshal(frame, protocol.ContainsSequence))

	if msg.Type != MsgTypeFullServer {
		err = fmt.Errorf("tts.startTTSSession: unexpected message type: %d", msg.Type)
		return
	}
	if Event(msg.Event) != EventSessionStarted {
		err = fmt.Errorf("tts.startTTSSession: unexpected event: %d", msg.Event)
		return
	}

	return
}

func (protocol *BinaryProtocol) sendTTSMessage(ctx context.Context, conn *websocket.Conn, sessionID, namespace string, params *TTSReqParams) (err error) {
	defer rg.Guard(&err)

	req := TTSRequest{
		Event:     int32(EventTaskRequest),
		Namespace: namespace,
		ReqParams: params,
	}

	payload := rg.Must(json.Marshal(&req))

	msg := rg.Must(NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent))
	msg.Event = req.Event
	msg.SessionID = sessionID
	msg.Payload = payload

	frame := rg.Must(protocol.Marshal(msg))

	rg.Must0(websocket.Message.Send(conn, frame))
	return
}

func (protocol *BinaryProtocol) finishSession(ctx context.Context, conn *websocket.Conn, sessionID string) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent))
	msg.Event = int32(EventFinishSession)
	msg.SessionID = sessionID
	msg.Payload = []byte("{}")

	frame := rg.Must(protocol.Marshal(msg))
	rg.Must0(websocket.Message.Send(conn, frame))
	return
}

func (protocol *BinaryProtocol) receiveMessage(ctx context.Context, conn *websocket.Conn) (msg *Message, err error) {
	defer rg.Guard(&err)

	var frame []byte
	rg.Must0(websocket.Message.Receive(conn, &frame))

	msg, _ = rg.Must2(Unmarshal(frame, protocol.ContainsSequence))
	return
}

func (protocol *BinaryProtocol) finishConnection(ctx context.Context, conn *websocket.Conn) (err error) {
	defer rg.Guard(&err)

	msg := rg.Must(NewMessage(MsgTypeFullClient, MsgTypeFlagWithEvent))
	msg.Event = int32(EventFinishConnection)
	msg.Payload = []byte("{}")

	frame := rg.Must(protocol.Marshal(msg))
	rg.Must0(websocket.Message.Send(conn, frame))
	rg.Must0(websocket.Message.Receive(conn, &frame))

	msg, _ = rg.Must2(Unmarshal(frame, protocol.ContainsSequence))

	if msg.Type != MsgTypeFullServer {
		err = fmt.Errorf("tts.finishConnection: unexpected message type: %d", msg.Type)
		return
	}
	if Event(msg.Event) != EventConnectionFinished {
		err = fmt.Errorf("tts.finishConnection: unexpected event: %d", msg.Event)
		return
	}

	return
}
