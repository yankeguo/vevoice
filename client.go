package volcvoice

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	defaultEndpoint = "openspeech.bytedance.com"

	envKeyDebug    = "VOLCVOICE_DEBUG"
	envKeyEndpoint = "VOLCVOICE_ENDPOINT"
	envKeyToken    = "VOLCVOICE_TOKEN"
	envKeyAppID    = "VOLCVOICE_APPID"
)

type options struct {
	endpoint string
	token    string
	appID    string
	debug    bool
	dialer   *websocket.Dialer
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

func WithDebug(debug bool) Option {
	return func(opts *options) {
		opts.debug = debug
	}
}

// Client is the interface for the volcvoice client.
type Client interface {
	// Synthesize create a new bidirectional voice synthesize service.
	Synthesize() *SynthesizeService
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
	opts.debug, _ = strconv.ParseBool(strings.TrimSpace(os.Getenv(envKeyDebug)))

	for _, fn := range fns {
		fn(&opts)
	}

	if opts.token == "" {
		return nil, errors.New("volcvoice.NewClient: token is required")
	}
	if opts.appID == "" {
		return nil, errors.New("volcvoice.NewClient: appId is required")
	}
	if opts.endpoint == "" {
		opts.endpoint = defaultEndpoint
		if opts.debug {
			log.Println("volcvoice.NewClient: using default endpoint:", opts.endpoint)
		}
	}
	if opts.dialer == nil {
		opts.dialer = websocket.DefaultDialer
	}

	return &client{options: opts}, nil
}

func (c *client) log(items ...any) {
	if c.debug {
		log.Println(items...)
	}
}

// Synthesize create a new bidirectional Synthesize service.
func (c *client) Synthesize() *SynthesizeService {
	return newSynthesizeService(c)
}
