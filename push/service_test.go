package push

import (
	"errors"
	"log"
)

func ExampleNewService() {
	applicationServerPublicKey := "BIRM67G3W1fva-ephDo220BbiaOOy-SBk2uzHsmlqMXp_OmkKxYW96cOK5EWnKdkLg2i7N4FYfuxIwm7JWThVSY"
	applicationServerPrivateKey := "QxfAyO5dMMrSvDT2_xHxW5aktYPWGE_hT42RKlHilpQ"

	opts := NewOptions().ApplyKeys(applicationServerPublicKey, applicationServerPrivateKey)

	pb, err := NewService(opts)
	if err != nil {
		panic(err)
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
