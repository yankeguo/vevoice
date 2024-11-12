package volcvoice

import (
	"errors"
	"os"
	"strings"

	"github.com/yankeguo/volcvoice/tts"
)

const (
	DefaultEndpoint = "openspeech.bytedance.com"

	EnvEndpoint = "VOLCVOICE_ENDPOINT"
	EnvToken    = "VOLCVOICE_TOKEN"
	EnvAppID    = "VOLCVOICE_APPID"
)

type options struct {
	endpoint string
	token    string
	appID    string
	debug    bool
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

// WithDebug sets the debug mode for the client.
func WithDebug(debug bool) Option {
	return func(opts *options) {
		opts.debug = debug
	}
}

// Client is the interface for the volcvoice client.
type Client interface {
	// TTS create a new bidirectional TTS service.
	TTS() *tts.Service
}

type client struct {
	opts options
}

// NewClient creates a new client with the given options.
func NewClient(fns ...Option) (Client, error) {
	opts := options{
		endpoint: strings.TrimSpace(os.Getenv(EnvEndpoint)),
		token:    strings.TrimSpace(os.Getenv(EnvToken)),
		appID:    strings.TrimSpace(os.Getenv(EnvAppID)),
	}
	if opts.endpoint == "" {
		opts.endpoint = DefaultEndpoint
	}
	for _, fn := range fns {
		fn(&opts)
	}
	if opts.endpoint == "" {
		return nil, errors.New("volcvoice.Client: endpoint is required")
	}
	if opts.token == "" {
		return nil, errors.New("volcvoice.Client: token is required")
	}
	if opts.appID == "" {
		return nil, errors.New("volcvoice.Client: appId is required")
	}
	return &client{opts: opts}, nil
}

// TTS create a new bidirectional TTS service.
func (c *client) TTS() *tts.Service {
	return tts.New().
		SetEndpoint(c.opts.endpoint).
		SetAppID(c.opts.appID).
		SetToken(c.opts.token).
		SetDebug(c.opts.debug)
}
