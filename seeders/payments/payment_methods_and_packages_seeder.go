package payments

import (
	payment "coolvibes/models/payment"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedPackagesAndPaymentMethods(db *gorm.DB) error {

	GOOGLE_PAY_TEST_MERCHANT_ID := "BCR2DN5T5DXM7QZ7"
	GOOGLE_PAY_TEST_API_KEY := "AIzaSyD-ExampleAPIKey"
	GOOGLE_PAY_PROD_MERCHANT_ID := "BCR2DN5T5DXM7QZ7"
	GOOGLE_PAY_PROD_API_KEY := "AIzaSyD-ExampleAPIKey"

	STRIPE_TEST_SECRET_KEY := "sk_test_51SWgHnK3hyIJvOd1Ftps5qgeqcmeD7CEeTAQ1EPBTMcTwP1D2DJRcnSUyQ6jgLWewdfhYZd61k3vx1J8uqoqi4R000BaYuGChN"
	STRIPE_TEST_PUBLIC_KEY := "pk_test_51SWgHnK3hyIJvOd1eWTHRXuYEOrhXcfiy7ISj2sP4p5EzlU62kvrTnRl4jQDcpuPmIdImF6UeCK0p5IYI7rWIogv00wmH0pdcw"

	secrets := []map[string]string{
		{
			"stripe_test_secret_key": STRIPE_TEST_SECRET_KEY,
			"stripe_test_public_key": STRIPE_TEST_PUBLIC_KEY,
		},
	}

	ibans := []map[string]string{
		{
			"kind":            string(payment.PaymentKind_IBAN),
			"bank_name":       "Yapı ve Kredi Bankası A.Ş.",
			"bank_short_name": "Yapı Kredi",
			"swift_bic":       "YAPITRISXXX",
			"iban":            "TR12 3456 7890 1234 5678 9012 34",
			"account_holder":  "ERSAN YAKIT",
			"currency":        "TRY",
			"branch_name":     "Zeytinburnu Şubesi",
			"branch_code":     "450",
			"country":         "Türkiye",
			"description":     "Yapı Kredi Türk Lirasi",
			"logo":            "/images/payments/yapi_kredi.svg",
		},
		{
			"kind":            string(payment.PaymentKind_IBAN),
			"bank_name":       "Türkiye İş Bankası A.Ş.",
			"bank_short_name": "İş Bankası",
			"swift_bic":       "YAPITRISXXX",
			"iban":            "TR80 0006 4000 0011 4440 0152 65",
			"account_holder":  "ERSAN YAKIT",
			"currency":        "TRY",
			"branch_name":     "Zeytinburnu Şubesi",
			"branch_code":     "1062",
			"country":         "Türkiye",
			"description":     "İş Bankası Türk Lirasi",
			"logo":            "/images/payments/is_bankasi.svg",
		},
	}

	cryptos := []map[string]string{
		{
			"kind":             string(payment.PaymentKind_CRYPTO),
			"chain_id":         "1",
			"contract_address": "0x0000000000000000000000000000000000000000",
			"name":             "Ethereum",
			"symbol":           "ETH",
			"decimals":         "18",
			"logo":             "/images/payments/eth.svg",
		},
		{
			"kind":             string(payment.PaymentKind_CRYPTO),
			"chain_id":         "88888",
			"contract_address": "0x0000000000000000000000000000000000000000",
			"name":             "Chiliz",
			"symbol":           "CHZ",
			"decimals":         "18",
			"logo":             "/images/payments/chz.svg",
		},
		{
			"kind":             string(payment.PaymentKind_CRYPTO),
			"chain_id":         "56",
			"contract_address": "0x0000000000000000000000000000000000000000",
			"name":             "Binance BNB",
			"symbol":           "BNB",
			"decimals":         "18",
			"logo":             "/images/payments/bnb.svg",
		},
		{
			"kind":             string(payment.PaymentKind_CRYPTO),
			"chain_id":         "43114",
			"contract_address": "0x0000000000000000000000000000000000000000",
			"name":             "Avalanche",
			"symbol":           "AVAX",
			"decimals":         "18",
			"logo":             "/images/payments/avax.svg",
		},
	}

	googlePays := []map[string]interface{}{
		{
			"name":        "Google Pay Test",
			"enabled":     true,
			"merchant_id": GOOGLE_PAY_TEST_MERCHANT_ID,
			"api_key":     GOOGLE_PAY_TEST_API_KEY,
			"description": "COOLVIBES LGBTIQ SOCIAL MEDIA APPLICATION STRIPE TEST MERCHANT",
			"environment": "TEST", // veya PRODUCTION
			"logo":        "/images/payments/google_pay.svg",
			"provider": map[string]string{
				"gateway":    "stripe",
				"name":       "Stripe",
				"logo":       "/images/payments/stripe.svg",
				"version":    "2025-11-17.clover",
				"public_key": STRIPE_TEST_PUBLIC_KEY,
			},
		},
		{
			"name":        "Google Pay",
			"enabled":     true,
			"merchant_id": GOOGLE_PAY_PROD_MERCHANT_ID,
			"api_key":     GOOGLE_PAY_PROD_API_KEY,
			"description": "COOLVIBES LGBTIQ SOCIAL MEDIA APPLICATION STRIPE MERCHANT",
			"environment": "PRODUCTION", // veya PRODUCTION
			"logo":        "/images/payments/google_pay.svg",
			"provider": map[string]string{
				"gateway":    "stripe",
				"name":       "Stripe",
				"logo":       "/images/payments/stripe.svg",
				"version":    "2025-11-17.clover",
				"public_key": STRIPE_TEST_PUBLIC_KEY,
			},
		},
	}

	packages := []map[string]interface{}{
		{
			"id": uuid.MustParse("d290f1ee-6c54-4b01-90e6-d701748f0851"),
			"name": map[string]string{
				"en": "Aldebaran",
				"tr": "Aldebaran",
			},
			"priceUSD": 10,
			"description": map[string]string{
				"en": "A bright and powerful start, like the shining Aldebaran star.",
				"tr": "Parlak ve güçlü bir başlangıç, parlayan Aldebaran yıldızı gibi.",
			},
			"appstore_sku":   "aldebaran_001",
			"googleplay_sku": "aldebaran_001_gp",
			"web_sku":        "aldebaran_web_001",
			"logo":           "/images/packages/package_1.png",
		},
		{
			"id": uuid.MustParse("a1e5f8d6-4f5a-4c9a-9b1a-4d9a5b3c678f"),
			"name": map[string]string{
				"en": "Bellatrix",
				"tr": "Bellatrix",
			},
			"priceUSD": 20,
			"description": map[string]string{
				"en": "Harness the courage and strength of Bellatrix, the warrior star.",
				"tr": "Savaşçı yıldız Bellatrix’in cesaret ve gücünü kullanın.",
			},
			"appstore_sku":   "bellatrix_002",
			"googleplay_sku": "bellatrix_002_gp",
			"web_sku":        "bellatrix_web_002",
			"logo":           "/images/packages/package_2.png",
		},
		{
			"id": uuid.MustParse("b7a1d9c8-3f2a-4e8a-9c4f-8a4b9c7d1234"),
			"name": map[string]string{
				"en": "Alnitak",
				"tr": "Alnitak",
			},
			"priceUSD": 30,
			"description": map[string]string{
				"en": "Reliable and steady, Alnitak guides you through the Cool Vibes experience.",
				"tr": "Güvenilir ve sağlam, Alnitak sizi Cool Vibes deneyiminde yönlendirir.",
			},
			"appstore_sku":   "alnitak_003",
			"googleplay_sku": "alnitak_003_gp",
			"web_sku":        "alnitak_web_003",
			"logo":           "/images/packages/package_3.png",
		},
		{
			"id": uuid.MustParse("c5f6a1b9-7e8c-4a2f-8d3c-1a2b3c4d5e6f"),
			"name": map[string]string{
				"en": "Alcyone",
				"tr": "Alcyone",
			},
			"priceUSD": 40,
			"description": map[string]string{
				"en": "Embrace the mystery and elegance of Alcyone, the brightest Pleiades star.",
				"tr": "En parlak Pleiades yıldızı Alcyone’un gizemini ve zerafetini kucaklayın.",
			},
			"appstore_sku":   "alcyone_004",
			"googleplay_sku": "alcyone_004_gp",
			"web_sku":        "alcyone_web_004",
			"logo":           "/images/packages/package_4.png",
		},
		{
			"id": uuid.MustParse("d4b5c6f7-9e0a-4d8b-9f1c-2b3a4c5d6e7f"),
			"name": map[string]string{
				"en": "Mintaka",
				"tr": "Mintaka",
			},
			"priceUSD": 50,
			"description": map[string]string{
				"en": "Steady and determined, Mintaka shines as a beacon in your social journey.",
				"tr": "Kararlı ve sabit, Mintaka sosyal yolculuğunuzda bir işaret ışığı gibi parlar.",
			},
			"appstore_sku":   "mintaka_005",
			"googleplay_sku": "mintaka_005_gp",
			"web_sku":        "mintaka_web_005",
			"logo":           "/images/packages/package_5.png",
		},
		{
			"id": uuid.MustParse("e5a6b7c8-d9e0-4f1a-b2c3-d4e5f6a7b8c9"),
			"name": map[string]string{
				"en": "Rigel",
				"tr": "Rigel",
			},
			"priceUSD": 60,
			"description": map[string]string{
				"en": "Shine with the brilliance and leadership of Rigel, a star of unmatched glory.",
				"tr": "Eşsiz ihtişamıyla liderlik ve parlaklık sunan Rigel ile parlayın.",
			},
			"appstore_sku":   "rigel_006",
			"googleplay_sku": "rigel_006_gp",
			"web_sku":        "rigel_web_006",
			"logo":           "/images/packages/package_6.png",
		},
		{
			"id": uuid.MustParse("f7b8c9d0-e1f2-4a3b-b4c5-d6e7f8a9b0c1"),
			"name": map[string]string{
				"en": "Fomalhaut",
				"tr": "Fomalhaut",
			},
			"priceUSD": 70,
			"description": map[string]string{
				"en": "Mystical and captivating, Fomalhaut leads you towards new horizons.",
				"tr": "Mistik ve büyüleyici, Fomalhaut sizi yeni ufuklara götürür.",
			},
			"appstore_sku":   "fomalhaut_007",
			"googleplay_sku": "fomalhaut_007_gp",
			"web_sku":        "fomalhaut_web_007",
			"logo":           "/images/packages/package_7.png",
		},
		{
			"id": uuid.MustParse("a9b0c1d2-e3f4-5a6b-c7d8-e9f0a1b2c3d4"),
			"name": map[string]string{
				"en": "Antares",
				"tr": "Antares",
			},
			"priceUSD": 80,
			"description": map[string]string{
				"en": "Feel the intense passion and power of Antares, the heart of the scorpion.",
				"tr": "Akrep’in kalbi Antares’in yoğun tutkusunu ve gücünü hissedin.",
			},
			"appstore_sku":   "antares_008",
			"googleplay_sku": "antares_008_gp",
			"web_sku":        "antares_web_008",
			"logo":           "/images/packages/package_8.png",
		},
		{
			"id": uuid.MustParse("c1d2e3f4-a5b6-7c8d-9e0f-a1b2c3d4e5f6"),
			"name": map[string]string{
				"en": "Deneb",
				"tr": "Deneb",
			},
			"priceUSD": 90,
			"description": map[string]string{
				"en": "Reach new heights with Deneb, a luminous guide in the night sky.",
				"tr": "Gece gökyüzünde parlak bir rehber olan Deneb ile yeni zirvelere ulaşın.",
			},
			"appstore_sku":   "deneb_009",
			"googleplay_sku": "deneb_009_gp",
			"web_sku":        "deneb_web_009",
			"logo":           "/images/packages/package_9.png",
		},
		{
			"id": uuid.MustParse("d2e3f4a5-b6c7-8d9e-0fa1-b2c3d4e5f6a7"),
			"name": map[string]string{
				"en": "Sadalsuud",
				"tr": "Sadalsuud",
			},
			"priceUSD": 100,
			"description": map[string]string{
				"en": "Reach the pinnacle with Sadalsuud, a rare and mystical star of unparalleled prestige.",
				"tr": "Eşi benzeri olmayan prestije sahip, nadir ve mistik Sadalsuud ile zirveye ulaşın.",
			},
			"appstore_sku":   "sadalsuud_010",
			"googleplay_sku": "sadalsuud_010_gp",
			"web_sku":        "sadalsuud_web_010",
			"logo":           "/images/packages/package_10.png",
		},
	}

	ibanJSON, err := json.Marshal(ibans)
	if err != nil {
		return fmt.Errorf("failed to marshal IBANs: %w", err)
	}

	cryptoJSON, err := json.Marshal(cryptos)
	if err != nil {
		return fmt.Errorf("failed to marshal cryptos: %w", err)
	}

	googlePayJSON, err := json.Marshal(googlePays)
	if err != nil {
		return fmt.Errorf("failed to marshal google pay: %w", err)
	}

	packagesJSON, err := json.Marshal(packages)
	if err != nil {
		return fmt.Errorf("failed to marshal packages: %w", err)
	}

	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Tek kayıt oluşturuluyor:
	paymentMethod := payment.PaymentMethod{
		ID:                 uuid.New(),
		DefaultPaymentKind: payment.PaymentKind_GOOGLEPAY, // ya da PaymentKind tipi yoksa string olarak böyle bırakabilirsin
		IBANDetails:        ibanJSON,
		IsIBANEnabled:      true,
		CryptoDetails:      cryptoJSON,
		IsCryptoEnabled:    true,
		GooglePayDetails:   googlePayJSON, // Google Pay token yoksa nil bırak
		IsGooglePayEnabled: true,
		Packages:           packagesJSON,
		SecretKeys:         secretsJSON,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	var existing payment.PaymentMethod

	err = db.First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Yok → Insert
		if err := db.Create(paymentMethod).Error; err != nil {
			return fmt.Errorf("failed to insert payment method: %w", err)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("db query error: %w", err)
	}

	// Var → Update
	if err := db.Model(&existing).Updates(paymentMethod).Error; err != nil {
		return fmt.Errorf("failed to update payment method: %w", err)
	}

	return nil

}
