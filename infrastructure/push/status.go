package push

import (
	"errors"
	"net/http"
)

var (
	// ErrPushBadRequest is returned when the push subscription request is malformed
	ErrPushBadRequest = errors.New("bad request: check the push subscription data and payload format")
	// ErrPushUnauthorized is returned when push service authentication is required
	ErrPushUnauthorized = errors.New("unauthorized: the push service requires valid authentication")
	// ErrPushForbidden is returned when the push service refuses to fulfill the request
	ErrPushForbidden = errors.New("forbidden: the push service understood the request, but is refusing to fulfill it")
	// ErrPushNotFound is returned when the push subscription cannot be found
	ErrPushNotFound = errors.New("push subscription not found: the user may have unsubscribed or the subscription may have expired")
	// ErrPushGone is returned when the push subscription is no longer active
	ErrPushGone = errors.New("push subscription is no longer active: delete it from your database")
	// ErrPushTooManyRequests is returned when the push service rate limit is exceeded or when GSB ban occurs
	ErrPushTooManyRequests = errors.New("too many push requests or GSB ban: try again later")
	// ErrPushInternalServerError is returned for push service server errors
	ErrPushInternalServerError = errors.New("push service internal server error: try again later")
	// ErrPushServiceUnavailable is returned when the push service is temporarily unavailable
	ErrPushServiceUnavailable = errors.New("push service unavailable: try again later")
	// ErrPushUnexpectedResponse is returned for unexpected status codes from push service
	ErrPushUnexpectedResponse = errors.New("unexpected response from the push service")
)

// StatusCodeValidationFunc is a function type that validates HTTP status codes
type StatusCodeValidationFunc func(statusCode int) error

// ValidateStatusCode takes an HTTP status code and returns an error if the status code
// indicates an error condition. It can be used to easily handle different types of
// errors that can occur when making HTTP requests.
func ValidateStatusCode(statusCode int) error {
	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusBadRequest:
		return ErrPushBadRequest
	case http.StatusUnauthorized:
		return ErrPushUnauthorized
	case http.StatusForbidden:
		return ErrPushForbidden
	case http.StatusNotFound:
		return ErrPushNotFound
	case http.StatusGone:
		return ErrPushGone
	case http.StatusTooManyRequests:
		return ErrPushTooManyRequests
	case http.StatusInternalServerError:
		return ErrPushInternalServerError
	case http.StatusServiceUnavailable:
		return ErrPushServiceUnavailable
	default:
		return ErrPushUnexpectedResponse
	}
}
