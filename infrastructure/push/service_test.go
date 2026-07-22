package push

import (
	"bytes"
	"context"
	"errors"
	"log"
	"testing"
	"time"

	"core/infrastructure/push/pkg/httpclient"
)

type deadlineClient struct {
	hadDeadline bool
}

func (c *deadlineClient) RequestDelivery(ctx context.Context, _ string, _ *httpclient.Headers, _ *bytes.Buffer) (int, error) {
	_, c.hadDeadline = ctx.Deadline()
	<-ctx.Done()
	return 0, ctx.Err()
}

func TestServiceAppliesPerDeliveryTimeout(t *testing.T) {
	client := &deadlineClient{}
	service, err := NewService(
		NewOptions().
			ApplyKeys(
				"BIRM67G3W1fva-ephDo220BbiaOOy-SBk2uzHsmlqMXp_OmkKxYW96cOK5EWnKdkLg2i7N4FYfuxIwm7JWThVSY",
				"QxfAyO5dMMrSvDT2_xHxW5aktYPWGE_hT42RKlHilpQ",
			).
			SetHttpClient(client).
			SetDeliveryTimeout(30 * time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = service.Send(&Push{
		Endpoint:  "https://push.example.test/subscription",
		Auth:      "rm_owGF0xliyVXsrZk1LzQ",
		P256DH:    "BKm5pKbGwkTxu7dJuuLyTCBOCuCi1Fs01ukzjUL5SEX1-b-filqeYASY6gy_QpPHGErGqAyQDYAtprNWYdcsM3Y",
		Plaintext: []byte(`{"title":"bounded"}`),
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send() error = %v; want deadline exceeded", err)
	}
	if !client.hadDeadline {
		t.Fatal("HTTP client received no deadline")
	}
}

func ExampleNewService() {
	applicationServerPublicKey := "BIRM67G3W1fva-ephDo220BbiaOOy-SBk2uzHsmlqMXp_OmkKxYW96cOK5EWnKdkLg2i7N4FYfuxIwm7JWThVSY"
	applicationServerPrivateKey := "QxfAyO5dMMrSvDT2_xHxW5aktYPWGE_hT42RKlHilpQ"

	opts := NewOptions().ApplyKeys(applicationServerPublicKey, applicationServerPrivateKey)

	pb, err := NewService(opts)
	if err != nil {
		log.Println(err)
		return
	}

	subscriptionEndpoint := "https://fcm.googleapis.com/fcm/send/e2CN0r8ft38:APA91bES3NaBHe_GgsRp_3Ir7f18L38wA5XYRoqZCbjMPEWnkKa07uxheWE5MGZncsPOr0_34zLaFljVqmNqW76KhPSrjdy_pdInnHPEIYAZpdcIYk8oIfo1F_84uKMSqIDXRhngL76S"
	subscriptionAuth := "rm_owGF0xliyVXsrZk1LzQ"
	subscriptionP256DH := "BKm5pKbGwkTxu7dJuuLyTCBOCuCi1Fs01ukzjUL5SEX1-b-filqeYASY6gy_QpPHGErGqAyQDYAtprNWYdcsM3Y"
	message := []byte("{\"title\": \"My first message\"}")

	err = pb.Send(&Push{
		Endpoint:  subscriptionEndpoint,
		Auth:      subscriptionAuth,
		P256DH:    subscriptionP256DH,
		Plaintext: message,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrPushGone):
			log.Println("user unsubscribed")
		default:
			log.Println(err)
		}
	}
}
