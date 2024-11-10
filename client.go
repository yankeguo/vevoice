package vevoice

import (
	"errors"
	"os"
	"strings"
)

const (
	DefaultEndpoint = "openspeech.bytedance.com"

	EnvEndpoint = "VEVOICE_ENDPOINT"
	EnvToken    = "VEVOICE_TOKEN"
	EnvAppID    = "VEVOICE_APPID"
)

type options struct {
	endpoint string
	token    string
	appID    string
	debug    bool
}

type Option func(opts *options)

// WithEndpoint sets a custom endpoint for the client, default to openspeech.bytedance.com, also can be set by VEVOICE_ENDPOINT env.
func WithEndpoint(endpoint string) Option {
	return func(opts *options) {
		opts.endpoint = endpoint
	}
}

// WithToken sets the token for the client, default to VEVOICE_TOKEN env.
func WithToken(token string) Option {
	return func(opts *options) {
		opts.token = token
	}
}

// WithAppID sets the appID for the client, default to VEVOICE_APPID env.
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

type Client interface{}

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
		return nil, errors.New("endpoint is required")
	}
	if opts.token == "" {
		return nil, errors.New("token is required")
	}
	if opts.appID == "" {
		return nil, errors.New("appID is required")
	}
	return &client{opts: opts}, nil
}

func (c *client) VoiceClone() *VoiceCloneService {
	return NewVoiceCloneService(c)
}
