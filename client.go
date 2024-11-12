package volcvoice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/yankeguo/rg"
)

const (
	defaultEndpoint = "openspeech.bytedance.com"

	envKeyDebug    = "VOLCVOICE_VERBOSE"
	envKeyEndpoint = "VOLCVOICE_ENDPOINT"
	envKeyToken    = "VOLCVOICE_TOKEN"
	envKeyAppID    = "VOLCVOICE_APPID"
)

type options struct {
	endpoint string
	token    string
	appID    string
	verbose  bool
	ws       *websocket.Dialer
	http     *http.Client
}

type Option func(opts *options)

// WithEndpoint sets a custom endpoint for the client, default to openspeech.bytedance.com, also can be set by VOLCVOICE_ENDPOINT env.
func WithEndpoint(endpoint string) Option {
	return func(opts *options) {
		opts.endpoint = endpoint
	}
}

// WithToken sets the token for the client, default to VOLCVOICE_TOKEN env.
func WithToken(token string) Option {
	return func(opts *options) {
		opts.token = token
	}
}

// WithAppID sets the appID for the client, default to VOLCVOICE_APPID env.
func WithAppID(appID string) Option {
	return func(opts *options) {
		opts.appID = appID
	}
}

// WithVerbose enables verbose mode for the client, default to VOLCVOICE_VERBOSE env.
func WithVerbose(verbose bool) Option {
	return func(opts *options) {
		opts.verbose = verbose
	}
}

// WithWebsocketDialer sets a custom websocket dialer for the client.
func WithWebsocketDialer(dialer *websocket.Dialer) Option {
	return func(opts *options) {
		opts.ws = dialer
	}
}

// WithHTTPClient sets a custom http client for the client.
func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) {
		opts.http = client
	}
}

// Client is the interface for the volcvoice client.
type Client interface {
	// Synthesize create a new bidirectional voice synthesize service.
	Synthesize() *SynthesizeService

	// VoiceCloneUpload create a new service for voice clone upload.
	VoiceCloneUpload() *VoiceCloneUploadService

	// VoiceCloneSynthesize create a new service for voice clone synthesize.
	VoiceCloneSynthesize() *VoiceCloneSynthesizeService
}

type client struct {
	options
}

// NewClient creates a new client with the given options.
func NewClient(fns ...Option) (Client, error) {
	opts := options{
		endpoint: strings.TrimSpace(os.Getenv(envKeyEndpoint)),
		token:    strings.TrimSpace(os.Getenv(envKeyToken)),
		appID:    strings.TrimSpace(os.Getenv(envKeyAppID)),
	}
	opts.verbose, _ = strconv.ParseBool(strings.TrimSpace(os.Getenv(envKeyDebug)))

	for _, fn := range fns {
		fn(&opts)
	}

	opts.endpoint = strings.TrimSuffix(strings.TrimPrefix(opts.endpoint, "/"), "/")

	if opts.token == "" {
		return nil, errors.New("volcvoice.NewClient: token is required")
	}
	if opts.appID == "" {
		return nil, errors.New("volcvoice.NewClient: appId is required")
	}
	if opts.endpoint == "" {
		opts.endpoint = defaultEndpoint
		if opts.verbose {
			log.Println("volcvoice.NewClient: using default endpoint:", opts.endpoint)
		}
	}
	if opts.ws == nil {
		opts.ws = websocket.DefaultDialer
	}
	if opts.http == nil {
		opts.http = http.DefaultClient
	}

	return &client{options: opts}, nil
}

func (c *client) debug(items ...any) {
	if c.verbose {
		log.Println(items...)
	}
}

func (c *client) httpPost(ctx context.Context, path string, header map[string]string, valueIn any, valueOut any) (err error) {
	defer rg.Guard(&err)

	buf := rg.Must(json.Marshal(valueIn))

	c.debug("POST", path)

	req := rg.Must(http.NewRequestWithContext(ctx, http.MethodPost, "https://"+c.endpoint+path, bytes.NewReader(buf)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}

	res := rg.Must(c.http.Do(req))
	defer res.Body.Close()

	buf = rg.Must(io.ReadAll(res.Body))

	c.debug("RESPONSE", res.Status)

	if res.StatusCode >= 400 {
		err = errors.New("client.httpPost: " + res.Status + " " + string(buf))
		return
	}

	rg.Must0(json.Unmarshal(buf, valueOut))

	return
}

func (c *client) Synthesize() *SynthesizeService {
	return newSynthesizeService(c)
}

func (c *client) VoiceCloneUpload() *VoiceCloneUploadService {
	return newVoiceCloneUploadService(c)
}

func (c *client) VoiceCloneSynthesize() *VoiceCloneSynthesizeService {
	return newVoiceCloneSynthesizeService(c)
}
