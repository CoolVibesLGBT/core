package push

import (
	"net/http"
	"time"

	"github.com/valyala/fasthttp"

	"core/infrastructure/push/pkg/httpclient"
)

// Options configures the push notification service settings.
type Options struct {
	ApplicationServerPublicKey  string                   // [RFC 8292] ECDH public key.
	ApplicationServerPrivateKey string                   // [RFC 8292] ECDH private key.
	ApplicationServerSubject    string                   // [RFC 8292] Either a "mailto:" (email) or a "https:" URI.
	StatusCodeValidationFunc    StatusCodeValidationFunc // [Optional] If set, use function that validates status codes and returns errors accordingly.
	HttpClient                  httpclient.Client        // [Optional] Custom client for request.
	KeyRotationInterval         time.Duration            // [Optional] If set, enable encryption keys rotation.
	DeliveryTimeout             time.Duration            // Maximum duration of one push-provider request.
}

// DefaultDeliveryTimeout bounds a single call to an external push provider.
const DefaultDeliveryTimeout = 2 * time.Second

// NewOptions creates and returns a new Options instance with default settings.
// By default, it sets the ApplicationServerSubject to the project's GitHub URL.
func NewOptions() *Options {
	return &Options{
		ApplicationServerSubject: "https://github.com/gootsolution/pushbell",
		DeliveryTimeout:          DefaultDeliveryTimeout,
	}
}

// ApplyKeys sets the ECDH public and private keys used for web push encryption.
// These keys are required for secure communication according to RFC 8292.
// Returns the updated Options instance for method chaining.
func (o *Options) ApplyKeys(publicKey, privateKey string) *Options {
	o.ApplicationServerPublicKey = publicKey
	o.ApplicationServerPrivateKey = privateKey

	return o
}

// SetSubject sets the application server subject.
// According to RFC 8292, this should be either a "mailto:" email address
// or an "https:" URI to identify the application server.
// Returns the updated Options instance for method chaining.
func (o *Options) SetSubject(subject string) *Options {
	o.ApplicationServerSubject = subject

	return o
}

// SetStatusCodeValidationFunc sets a custom function for HTTP status code validation.
// This function will be used to determine if a response should be treated as an error
// based on its status code.
// Returns the updated Options instance for method chaining.
func (o *Options) SetStatusCodeValidationFunc(validationFunc StatusCodeValidationFunc) *Options {
	o.StatusCodeValidationFunc = validationFunc

	return o
}

// SetKeyRotationEnabled enables encryption key rotation for improved security.
// Key rotation helps reduce the risk associated with compromised encryption keys.
// Returns the updated Options instance for method chaining.
func (o *Options) SetKeyRotationEnabled() *Options {
	o.KeyRotationInterval = time.Hour

	return o
}

// SetKeyRotationInterval enables key rotation and sets the interval duration
// between key rotations. This provides more control over the key rotation schedule.
// Returns the updated Options instance for method chaining.
func (o *Options) SetKeyRotationInterval(interval time.Duration) *Options {
	o.KeyRotationInterval = interval

	return o
}

// SetDeliveryTimeout sets the maximum duration of one push-provider request.
// Non-positive values keep the safe default instead of disabling the deadline.
func (o *Options) SetDeliveryTimeout(timeout time.Duration) *Options {
	if timeout > 0 {
		o.DeliveryTimeout = timeout
	}

	return o
}

// SetHttpClient sets a custom HTTP client for making web push requests.
// This allows for greater control over HTTP connection parameters.
// Returns the updated Options instance for method chaining.
func (o *Options) SetHttpClient(client httpclient.Client) *Options {
	o.HttpClient = client

	return o
}

// SetFastHttpClient sets a fasthttp client for making web push requests.
// fasthttp can provide better performance in certain scenarios.
// Returns the updated Options instance for method chaining.
func (o *Options) SetFastHttpClient(client *fasthttp.Client) *Options {
	o.HttpClient = httpclient.FastHttp(client)

	return o
}

// SetStdHttpClient sets a standard library http client for making web push requests.
// This is useful when the standard http package is preferred.
// Returns the updated Options instance for method chaining.
func (o *Options) SetStdHttpClient(client *http.Client) *Options {
	o.HttpClient = httpclient.StdHttp(client)

	return o
}
