# CoolVibes Core

[VS Code eklentisini indir](https://marketplace.visualstudio.com/items?itemName=CoolVibes.coolvibes-vscode-extension)

CoolVibes Core; sosyal ağ, tanışma ve topluluk deneyimlerinin Go ile geliştirilen backend uygulamasıdır. Web ve mobil istemcilere profil, keşif, eşleşme, gönderi, sohbet, özel albüm, bildirim ve moderasyon servisleri sağlar. HTTP action API, Socket.IO ve MCP aynı uygulama servislerini kullanır.

Bu README mevcut kodun sunduğu özellikleri ve yerel/Docker kurulumunu anlatır. İstek alanları ve action sözleşmeleri için çalışan sunucunun [Swagger arayüzünü](http://localhost:3001/swagger) kullanabilirsiniz.

## İçindekiler

- [Ürün özellikleri](#ürün-özellikleri)
- [Teknoloji ve mimari](#teknoloji-ve-mimari)
- [Yerel kurulum](#yerel-kurulum)
- [Docker ile kurulum](#docker-ile-kurulum)
- [Yapılandırma](#yapılandırma)
- [HTTP API ve Swagger](#http-api-ve-swagger)
- [AI API ve MCP](#ai-api-ve-mcp)
- [Gerçek zamanlı iletişim](#gerçek-zamanlı-iletişim)
- [Medya ve yayınlama](#medya-ve-yayınlama)
- [Bakım ve geliştirme](#bakım-ve-geliştirme)
- [Sorun giderme](#sorun-giderme)

## Ürün özellikleri

### Hesaplar ve profiller

- Kullanıcı kaydı, kullanıcı adı/e-posta ile giriş ve JWT ile oturum doğrulama.
- Profil bilgilerini ve tercihleri güncelleme; avatar, kapak ve hikâye yükleme; hesabı silme.
- Profil görüntüleme, ziyaret ve etkileşim kayıtları; kullanıcıya ait gönderi, yanıt ve medya listeleri.
- Profil görünürlüğü, engelleme ve domain kurallarıyla erişim kontrolü. API yanıtlarında herkese açık profil verileri için ayrı projeksiyonlar kullanılır.
- Başlangıç senkronizasyonunda tercihler, diller, etkinlik ve şikâyet türleri gibi istemcinin ihtiyaç duyduğu referans veriler.

### Keşif, sosyal ilişkiler ve eşleşme

- Konuma göre yakındaki kullanıcıları ve mekânları keşfetme; PostGIS ile mesafe sorguları.
- Kullanıcı adı arama, genel arama, gönderi arama, trendler ve kategori filtreleri.
- Takip etme/takipten çıkma, takipçi ve takip edilen listeleri, kullanıcı beğenme/beğenmeme ve abonelik durumunu değiştirme.
- Arkadaşlık isteği gönderme, gelen/giden istekleri listeleme, kabul/ret, iptal ve arkadaşlıktan çıkarma. Karşılıklı istek gönderilmesi otomatik kabul oluşturmaz; alıcının onayı gerekir.
- Tanışma akışında adaylar, beğenilenler, geçilenler ve karşılıklı eşleşmeler; eşleşmeyi kaldırma.
- Aday sıralamasında ortak ilgi alanları (%60), diller (%20) ve yakınlık (%20). Yaş ve azami mesafe filtreleri desteklenir; koordinatlar ve özel tercih bayrakları aday yanıtlarına eklenmez.
- Kullanıcı engelleme, engeli kaldırma ve kullanıcı şikâyeti oluşturma.

### Gönderiler, içerik ve topluluk

- Gönderi, durum, video, hikâye, etkinlik ve check-in içerikleri; medya, hashtag, mention ve konum alanları.
- Zaman akışı, medya odaklı Vibes akışı, kullanıcı gönderileri, yanıtlar ve kategori taksonomisi.
- Gönderi beğenme/beğenmeme, banana etkileşimi, görüntüleme, yer imleri ve içerik şikâyetleri.
- Kullanıcının beğendiği ve kaydettiği gönderi koleksiyonları; kaydedilenler için sahiplik kontrolü.
- Anket seçeneklerine oy verme, etkinlik katılım durumu (RSVP), yetkili gönderi silme ve gönderi sahibine bakiyeden bahşiş gönderme.
- Mekân oluşturma, kategori ve yakındaki mekân listeleri; haber oluşturma/listeleme; ilan, iş teklifi ve iş arama içerikleri. Haber oluşturma başlangıçta yayımlanmamış içerik üretir.
- Genel bağlantılardan Open Graph, Twitter Card ve keşfedilen oEmbed verilerini alma.
- Sunucu tarafında HTML sayfaları ve gönderi, haber, kategori, görsel ve video sitemap çıktıları.

### Özel fotoğraf albümleri

- Profil fotoğrafları ile özel albümün ayrı listelenmesi, fotoğraf yükleme/silme ve albümler arasında taşıma.
- Erişim isteği oluşturma, istekleri listeleme, kabul/ret, sahibin doğrudan erişim vermesi, erişim durumu sorgulama ve izni geri alma.
- Engellemede özel albüm izinlerinin geri alınması ve değişikliklerin gerçek zamanlı iletilmesi.
- Ücretsiz kullanıcılar için 10 fotoğraf sınırı; premium sınırsız albüm davranışı `FF_PREMIUM_UNLIMITED_PRIVATE_PHOTOS` ile yönetilir.
- Özel medya bağlantıları her istekte yetkilendirilir. Albüm öğe sayısı kuralları dosya boyutu sınırı değildir.

### Birebir ve grup sohbetleri

- Birebir sohbet ve grup oluşturma, sohbet ve mesaj geçmişini sayfalama.
- Metin/medya mesajları, mesaj düzenleme, iletme, yazıyor durumu, mesajı açma ve okundu takibi.
- Grup katılımcılarını listeleme/ekleme/çıkarma ve gruptan ayrılma; yönetici ve kurucu yetki kuralları.
- Herkese açık grup keşfi ve önizlemesi, gruba katılma, davet bağlantısı üretme ve grup bilgilerini güncelleme.
- Sohbeti sessize alma, mesaj sabitleme/sabitlemeyi kaldırma.
- Mesajı veya sohbeti kendisi için silme, yetki dahilinde herkes için silme ve geçmişi temizleme.
- Sohbet medyası için aktif üyelik ve erişim kontrolü; katılımcı değişiklikleri için gerçek zamanlı yenileme olayları.

### Bildirimler ve moderasyon

- Kullanıcı bildirim listeleri, Socket.IO üzerinden bildirimler ve Web Push aboneliği/VAPID anahtarı işlemleri.
- Kalıcı bildirim outbox ve teslimat kayıtlarıyla arka plan işleme.
- Gönderi ve kullanıcı şikâyetleri, şikâyet türleri, moderasyon kuyruğu ve şikâyeti sonuçlandırma.
- İçerik gizleme/yeniden yayımlama; moderator, admin ve super_admin rolleriyle yetki kontrolü.
- Mevcut kullanılabilir hesaba CLI üzerinden admin/moderator rolü verme.

### Yayınlar, bakiye ve davet ödülleri

- Yapılandırılmış harici yayın servislerinde yayınları listeleme, yayın oluşturma/görüntüleme, konuk katılım isteği ve beğeni işlemleri.
- Yerel hediye oturumu açma, heartbeat ile canlı tutma, sorgulama ve kapatma. Heartbeat gelmezse oturum iki dakika sonra sona erer.
- Yerel yayın sahibine uygulama bakiyesinden hediye ve gönderi sahibine bahşiş transferi. Para işlemleri ondalık tutarlar ve tekrar denemelerde çift tahsilatı önleyen idempotency anahtarları kullanır.
- Hediye transferinde bakiyeler, makbuz, sayaçlar ve kalıcı bildirim aynı transaction içinde kaydedilir. Yerel hediye oturumu harici sağlayıcı yayını veya harici platform kredisi oluşturmaz; içe aktarılan bot hesapları yerel hediye alamaz.
- Kayıt sırasında davet kodu, bekleyen ödüller, kullanıcının ödül listesi ve yönetici inceleme kuyruğu.
- Davet ödülü bağımsız admin/super_admin doğrulaması ve onayından sonra harcanabilir olur. Normal moderatör para ödülü onaylayamaz; aynı doğrulama referansı birden fazla hesabı ödüllendiremez.
- Ödeme yöntemi/paket metaverisi bulunur. Stripe `thin` ve `snapshot` webhook rotaları şu an **501 Not Implemented** döndürür; tamamlanmış ödeme entegrasyonu olarak kullanılmamalıdır.

### AI ve entegrasyonlar

- Gemini, OpenAI ve Ollama sağlayıcıları üzerinden metin üretimi; sağlayıcı/model seçimi, sistem talimatı ve temperature alanı.
- MCP üzerinden AI, haber ve mekân araçları; HTTP JSON-RPC ve stdio bağlantısı.
- Swagger UI ve JSON/YAML OpenAPI çıktıları; HTTP action sözleşmelerini inceleme ve deneme.
- İsteğe bağlı Telegram webhook entegrasyonu. RSS/haber ve harici yayın işleyicileri kod tabanında bulunur; normal sunucu başlangıcında medya, sohbet ve bildirim işleyicileri başlatılır.

## Teknoloji ve mimari

| Bileşen | Kullanım |
| --- | --- |
| Go 1.26.3 | Uygulama ve bakım komutları |
| Fiber v3 | HTTP, middleware, statik dosyalar ve HTML |
| PostgreSQL + PostGIS | Kalıcı veri, transaction ve konumsal sorgular |
| GORM | Veritabanı adaptörleri ve şema migrasyonları |
| Socket.IO | Gerçek zamanlı sohbet ve bildirim |
| JWT HS256 | Kullanıcı kimliği ve oturum doğrulama |
| FFmpeg / ffprobe | Video çözünürlükleri, poster ve önizleme |
| MCP | AI istemcilerine araç erişimi |

```text
core/
├── adapters/inbound/http/   # HTTP router, middleware ve handler'lar
├── adapters/inbound/mcpserver/ # MCP araç adaptörleri
├── application/ports/       # Repository ve dış servis arayüzleri
├── application/types/       # Uygulama DTO'ları ve projeksiyonlar
├── application/usecases/    # Uygulama iş akışları
├── domain/                 # İş kuralları
├── infrastructure/bootstrap/ # Elle yapılan bağımlılık kurulumu
├── infrastructure/         # DB, auth, AI, socket ve repository adaptörleri
├── models/                 # GORM/legacy şema sınırı
├── mcp/                    # MCP protokol ve transport altyapısı
├── workers/                # Arka plan işleyicileri
├── seeders/                # Başlangıç ve referans veriler
├── docs/                   # OpenAPI üretimi ve mimari denetim
├── static/                 # Medya, GeoIP verisi ve statik dosyalar
├── views/                  # HTML şablonları
└── main.go                 # Süreç yaşam döngüsü ve CLI
```

Bağımlılık yönleri ve kalan dönüşüm işleri [DDD denetiminde](docs/ddd-audit.md) açıklanır.

## Yerel kurulum

### 1. Gereksinimler

- `go.mod` ile uyumlu Go **1.26.3**.
- PostgreSQL ve PostGIS; depodaki Compose ve CI yapılandırması PostgreSQL **17 / PostGIS 3.6** kullanır.
- C derleyicisi ve CGO desteği (WebP bağımlılığı için).
- Video işleme için PATH üzerinde `ffmpeg` ve `ffprobe`.
- Git; veritabanını konteynerde çalıştıracaksanız Docker ve Compose.

macOS üzerinde yardımcı araçları kurmak için:

```bash
xcode-select --install
brew install ffmpeg
```

Linux üzerinde C derleme araçları ve FFmpeg paketleri bulunmalıdır. PostgreSQL'i işletim sistemine kurmak yerine aşağıdaki Compose veritabanı kullanılabilir.

### 2. Ortam dosyasını hazırlayın

Repository'yi aldıktan sonra `core` dizininde çalışın:

```bash
cp env.sample .env
openssl rand -base64 48
```

Üretilen değeri `.env` içindeki `USER_JWT_SECRET` alanına yazın. En az 32 byte gerekir. Yerel Compose veritabanı için temel ayarlar:

```dotenv
DATABASE_URL="host=localhost port=5432 user=postgres password=password dbname=databasename sslmode=disable"
USER_JWT_SECRET="BURAYA_URETILEN_SECRET"
PORT=":3001"
SOCKET_PORT=":3002"
APP_BASE_URL="http://localhost:3001"
SITE_URI="http://localhost:3001"
MCP_ALLOWED_ORIGINS="http://localhost:3001"
```

Secret değerlerini repository'ye eklemeyin. Uygulama `.env` dosyasını yükler; önceden tanımlanmış süreç ortam değişkenleri önceliklidir.

### 3. Veritabanını başlatın

```bash
docker compose up -d db
```

Compose proje ortamını okuduğu için bu komuttan önce `USER_JWT_SECRET` ayarlanmış olmalıdır. Hazır olma durumunu kontrol edin:

```bash
docker compose ps db
docker compose exec db pg_isready -U postgres -d databasename
```

Kendi PostgreSQL sunucunuzu kullanıyorsanız veritabanını oluşturun, `DATABASE_URL` değerini değiştirin ve migrasyon kullanıcısının `uuid-ossp`, `postgis`, `pg_trgm` eklentilerini oluşturabildiğinden emin olun.

### 4. Dosyaları ve bağımlılıkları kontrol edin

```bash
go mod download
test -f static/data/GeoLite2-City.mmdb
ffmpeg -version
ffprobe -version
```

`static/data/GeoLite2-City.mmdb` GeoIP dosyası uygulamanın açılması için gereklidir. Checkout veya dağıtım paketinde yoksa bu konuma geçerli bir GeoLite2 City veritabanı yerleştirin. `views`, `static` ve `seeders/places/places.json` dosyalarını dağıtımda koruyun.

### 5. Şema ve başlangıç verilerini yükleyin

```bash
go run . -install
```

Bu komut migrasyonları ve indeksleri, ardından tercihleri, sistem kullanıcılarını, etkinlik/şikâyet türlerini, ödeme paketlerini ve mekân verilerini yükler. Mevcut veritabanını güncellemek için yalnızca migrasyon çalıştırabilirsiniz:

```bash
go run . -migrate
```

Migrasyon, şikâyet türlerini de yükler; şikâyet işlevi tam seed çalıştırılmasına bağlı değildir.

### 6. Sunucuyu başlatın

```bash
go run .
```

- HTTP: `http://localhost:3001`
- Swagger: `http://localhost:3001/swagger`
- Socket.IO: `http://localhost:3002/socket.io/`
- MCP HTTP: `http://localhost:3001/mcp`

İlk doğrulama isteği:

```bash
curl -fsS http://localhost:3001/api/actions/system.initial_sync
```

Mevcut `start.sh` yalnızca `go run .` çalıştırır ve argüman iletmez. Kurulum/bakım için yukarıdaki doğrudan komutları kullanın. Yerel `go run .` başlangıcı otomatik migrasyon yapmaz; `env.sample` içindeki `AUTO_MIGRATE` mevcut Go kodunda okunmaz.

## Docker ile kurulum

Önce `.env` dosyasını ve `USER_JWT_SECRET` değerini yukarıdaki gibi hazırlayın:

```bash
docker compose build app
docker compose up -d db
docker compose run --rm app ./core -install
docker compose up -d app
docker compose logs -f app
```

Dockerfile C derleme araçlarını build aşamasında, FFmpeg'i çalışma imajında kurar. Konteyner başlangıcı `./core -migrate` çalıştırır ve başarılı olursa sunucuyu başlatır. Veritabanı sağlık kontrolü uygulamanın bağımlılığıdır.

`postgres_data` veritabanını, `uploads_data` yüklenen medyayı kalıcı tutar. Konteyner içindeki DB adresi `db`, host üzerinde çalışan uygulamanın DB adresi `localhost` olmalıdır. Yerel Go sunucusu ile Compose `app` servisini aynı portlarda eşzamanlı başlatmayın.

Compose `app.environment` altında yalnızca tanımlanmış değişkenleri iletir; `.env` dosyasındaki her alan otomatik olarak konteynere geçmez. AI, Telegram, MCP origin veya harici yayın ayarları ekliyorsanız ilgili değişkenleri Compose ortamına da ekleyin. `SITE_URI` gibi sabit Compose değerlerini kendi kurulumunuza göre düzenleyin.

## Yapılandırma

Başlangıç şablonu: [env.sample](env.sample).

| Değişken | Açıklama |
| --- | --- |
| `DATABASE_URL` | PostgreSQL bağlantı DSN'i; zorunlu |
| `USER_JWT_SECRET` | En az 32 byte kullanıcı JWT sırrı; HTTP uygulaması için zorunlu |
| `PORT` | HTTP dinleme adresi; ör. `:3001` |
| `SOCKET_PORT` | Socket.IO adresi; boş bırakılırsa socket sunucusu başlatılmaz |
| `DB_CONNECT_TIMEOUT` | Bağlantı zaman aşımı; ör. `3s` |
| `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS` | DB bağlantı havuzu; örnekte 50/10 |
| `SITE_URI`, `APP_BASE_URL` | Servis/uygulama adresleri; MCP origin izinlerine de katılır |
| `TRUSTED_PROXIES` | Virgülle ayrılmış güvenilir proxy IP/CIDR listesi |
| `CAPTCHA_SECRET_KEY` | Kayıt akışı için reCAPTCHA doğrulaması |
| `FF_PREMIUM_UNLIMITED_PRIVATE_PHOTOS` | Premium sınırsız özel albüm özelliği; varsayılan `true` |
| `MCP_ALLOWED_ORIGINS` | Ek MCP origin izinleri; virgülle ayrılır |
| `TELEGRAM_BOT_TOKEN` | İsteğe bağlı Telegram bot erişimi |
| `TELEGRAM_WEBHOOK_URL`, `TELEGRAM_WEBHOOK_SECRET` | Telegram webhook adresi ve doğrulama sırrı |
| `BROADCAST_HORNET_*`, `BROADCAST_GROWLR_*` | Harici yayın adaptörleri; gerekli alanlar örnek ortam dosyasında açıklanır |

AI sağlayıcı ayarları aşağıda ayrıca açıklanır. `RIG_JWT_SECRET`, kullanıcı oturumları için gereken `USER_JWT_SECRET` yerine geçmez.

## HTTP API ve Swagger

Core action tabanlıdır. Action'lar [constants/commands.go](constants/commands.go), aktif kayıtları [routes.go](adapters/inbound/http/routes/routes.go) içinde tanımlanır.

| Adres | İşlev |
| --- | --- |
| `/packet`, `/api/` | Action dispatch; GET/POST |
| `/api/actions/:action` | Action'ı URL üzerinden seçme |
| `/swagger`, `/docs` | Swagger UI |
| `/swagger/openapi.json`, `/docs/openapi.json` | OpenAPI JSON |
| `/swagger/openapi.yaml`, `/docs/openapi.yaml` | OpenAPI YAML |
| `/mcp` | MCP JSON-RPC; POST/DELETE |
| `/webhook/bot/telegram/` | Telegram güncellemeleri |
| `/static/*` | Statik varlıklar ve erişim kontrollü medya |
| `/sitemap.xml` | Sitemap indeksi |

`/packet` action seçimi sırasıyla `X-Action` başlığı, `action` query parametresi ve POST gövdesindeki `action` alanından yapılır. JSON ile action seçilebilse de birçok handler parametreleri form alanlarından okur; her işlemin beklediği alanları Swagger'dan kontrol edin. Dosya yüklemelerinde `multipart/form-data` ve `X-Action` kullanın.

Giriş örneği:

```bash
curl -sS http://localhost:3001/api/actions/auth.login \
  --data-urlencode 'nickname=kullanici_adiniz' \
  --data-urlencode 'password=parolaniz'
```

Korumalı isteklerde giriş yanıtındaki token'ı kullanın:

```bash
curl -sS http://localhost:3001/packet \
  -H 'X-Action: auth.check' \
  -H "Authorization: Bearer $USER_TOKEN"
```

JWT doğrulaması HS256, beklenen issuer/subject/kimlik ve zaman claim'lerini gerektirir; saat sapması toleransı 30 saniyedir. Eski sürümlerde üretilmiş `iat`/`nbf` içermeyen token'larla yeniden giriş gerekir. Yetkiler ve hesap kullanılabilirliği oturum akışında kontrol edilir.

Swagger UI'da action yolları üzerinden parametreleri inceleyebilir ve `Authorize` ile bearer token kullanabilirsiniz. OpenAPI belgeleri [docs](docs) paketindeki Go tanımlarından çalışma anında oluşturulur.

## AI API ve MCP

### Sağlayıcıları yapılandırma

AI erişimi `ai.generate_text` MCP aracı üzerinden sunulur. Aynı araç Gemini, OpenAI ve Ollama sağlayıcılarını destekler. Sağlayıcı yapılandırılmamışsa sunucu AI olmadan açılır ve araç `tools/list` listesine eklenmez.

| Sağlayıcı | Ortam değişkenleri |
| --- | --- |
| Gemini (`gemini`) | `GOOGLE_API_KEY` veya `GEMINI_API_KEY`; model için `GOOGLE_GENAI_MODEL` |
| Vertex AI üzerinden Gemini | `GOOGLE_GENAI_USE_VERTEXAI=true`, `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION` veya `GOOGLE_CLOUD_REGION`; ortamın Google kimlik bilgileri |
| OpenAI (`openai`) | `OPENAI_API_KEY`, isteğe bağlı `OPENAI_BASE_URL`, `OPENAI_MODEL` |
| Ollama (`ollama`) | `OLLAMA_BASE_URL` veya `OLLAMA_HOST`, `OLLAMA_MODEL`, isteğe bağlı `OLLAMA_API_KEY` |

`AI_DEFAULT_PROVIDER` (alternatif `AI_PROVIDER`) varsayılan sağlayıcıyı, `AI_MODEL` ortak model fallback değerini belirler. Sağlayıcıya özel model ayarı önceliklidir. Varsayılan sağlayıcı açıkça seçilmişse o sağlayıcının yapılandırılmış olması gerekir. Yalnızca kullanacağınız sağlayıcının ayarlarını ekleyin; anahtarları istemciye göndermeyin.

Örneğin Gemini için:

```dotenv
AI_DEFAULT_PROVIDER=gemini
GOOGLE_API_KEY="kendi-api-anahtariniz"
GOOGLE_GENAI_MODEL="gemini-2.5-flash"
```

### Araçlar ve transport

| MCP aracı | İşlev |
| --- | --- |
| `ai.generate_text` | Yapılandırılmış sağlayıcıyla metin üretir |
| `news.fetch` | Ortak filtrelerle haberleri listeler |
| `news.get` | Haber kimliğiyle tek haber getirir |
| `news.categories` | Haber kategorilerini listeler |
| `place.fetch` | Ortak filtrelerle mekânları listeler |
| `place.categories` | Mekân kategorilerini listeler |

MCP istemcisi önce `initialize`, ardından `notifications/initialized`, sonra `tools/list` ve `tools/call` kullanır. Desteklenen protokol sürümleri `2025-11-25`, `2025-06-18` ve `2025-03-26`'dır. HTTP transport JSON yanıtı döndürür; `GET /mcp` **405** döndürür, SSE akışı sunmaz. JSON-RPC batch yalnızca initialize edilmiş `2025-03-26` oturumlarında desteklenir.

### HTTP üzerinden AI çağrısı

Önce oturum açın; yanıt başlıklarındaki `MCP-Session-Id` değerini alın:

```bash
curl -i http://localhost:3001/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"core-example","version":"1.0.0"}}}'
```

Dönen değeri `MCP_SESSION_ID` değişkenine yazın ve istemciyi hazır olarak bildirin:

```bash
export MCP_SESSION_ID='yanittaki-oturum-kimligi'
curl -i http://localhost:3001/mcp \
  -H 'Content-Type: application/json' \
  -H "MCP-Session-Id: $MCP_SESSION_ID" \
  -H 'MCP-Protocol-Version: 2025-11-25' \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'
```

Araç listesini kontrol edin:

```bash
curl -sS http://localhost:3001/mcp \
  -H 'Content-Type: application/json' \
  -H "MCP-Session-Id: $MCP_SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
```

Metin üretin:

```bash
curl -sS http://localhost:3001/mcp \
  -H 'Content-Type: application/json' \
  -H "MCP-Session-Id: $MCP_SESSION_ID" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ai.generate_text","arguments":{"provider":"gemini","prompt":"Bir topluluk etkinliği için kısa bir duyuru yaz.","system_instruction":"Türkçe ve samimi yaz.","temperature":0.7}}}'
```

`prompt` zorunludur; `provider`, `model`, `system_instruction` ve `temperature` isteğe bağlıdır. Araç sonucu `text`, `model` ve `provider` alanlarını içerir. Kullanılabilir sağlayıcılar `tools/list` içindeki araç şemasında görünür. Oturumu `DELETE /mcp` isteğinde `MCP-Session-Id` göndererek kapatabilirsiniz.

Origin başlığı olan HTTP istekleri aynı origin veya `APP_BASE_URL`, `SITE_URI`, `MCP_ALLOWED_ORIGINS` izinleriyle doğrulanır. Mevcut `/mcp` rotası kullanıcı JWT middleware'i kullanmaz; dış erişim politikasını dağıtım katmanında belirleyin.

### Stdio bağlantısı

```bash
go build -o build/core .
./build/core -mcp-stdio
```

MCP istemcisinde çalıştırılacak komut olarak binary'nin mutlak yolunu, argüman olarak `-mcp-stdio`, çalışma dizini olarak repository'nin `core` kökünü tanımlayın. Bu mod HTTP ve Socket.IO sunucularını başlatmaz; DB, GeoIP ve AI bağımlılıklarını kurar. Veritabanı migrasyonlarını önceden çalıştırın.

### Swagger ile MCP ilişkisi

Swagger/OpenAPI, `/mcp` POST ve DELETE transport sözleşmesini, oturum başlıklarını ve JSON-RPC istek gövdesini belgeler. AI araçlarının ayrıntılı çalışma zamanı şemaları MCP `tools/list` yanıtından keşfedilir; AI çağrısı `/mcp` üzerinden `tools/call` ile yapılır. `ai.generate_text` için ayrı bir HTTP action rotası kayıtlı değildir.

## Gerçek zamanlı iletişim

Socket.IO, `SOCKET_PORT` doluysa HTTP sunucusunun yanında açılır. İstemci bağlantı `auth` nesnesinde `token` gönderebilir veya bağlantı sonrası `auth` olayıyla kimliğini doğrulayabilir. Sonuç `auth:ack` ile bildirilir.

Sohbet, yazıyor/okundu durumları, bildirimler, grup katılımcı değişiklikleri ve özel albüm erişim değişiklikleri gerçek zamanlı olaylarla iletilir. Kullanıcı/sohbet odaları sunucunun kimlik ve üyelik denetimiyle belirlenir. Socket.IO istemcisi kullanın; ham WebSocket istemcisi aynı protokolü konuşmaz.

## Medya ve yayınlama

Yüklenen dosyalar `static/uploads` altında saklanır. Büyük multipart dosyaları geçici diske alınır; uygulamada ürün düzeyinde dosya byte sınırı uygulanmaz. Geçici dizin ve upload diskinde yeterli alan olmalıdır. Görseller ve videolar arka planda işlenir; video işleme poster, farklı çözünürlükler ve kısa önizleme üretir.

Özel albüm ve sohbet medyasını indiren istekler yetkili bearer token taşımalıdır. Bu alanları doğrudan herkese açık bir statik sunucuyla servis ederek uygulama erişim denetimini atlamayın.

Nginx API upstream örneği:

```nginx
location / {
    proxy_pass http://127.0.0.1:3001;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    client_max_body_size 0;
    proxy_request_buffering off;
}
```

Socket host'u veya ilgili location için:

```nginx
location /socket.io/ {
    proxy_pass http://127.0.0.1:3002;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
}
```

`TRUSTED_PROXIES` değerini gerçek proxy IP/CIDR listesiyle yapılandırın. HTTPS/TLS ve dış adresleri kendi dağıtımınızda tanımlayın. Uygulamada özel bir `/health` endpoint'i kayıtlı değildir; bir reverse proxy'nin sabit 200 yanıtı DB veya uygulama sağlığını ölçmez.

## Bakım ve geliştirme

Komutları repository kökünde çalıştırın:

```bash
# Şema ve indeksler
go run . -migrate

# Başlangıç verileri
go run . -seed

# İkisini sırayla çalıştır
go run . -install

# Mevcut hesabın public ID'siyle rol ver
go run . -grant-admin 123456789
go run . -grant-moderator 987654321

# Telegram webhook modunu belirten başlangıç
go run . -bot

# Derleme
go build -trimpath -ldflags="-s -w" -o build/core .

# Test ve lint
go test ./...
go test -race -cover ./...
golangci-lint run ./...
```

Bakım komutları işleri bitirip çıkar; ardından sunucuyu ayrıca başlatın. `-bot` ayrı bir polling sunucusu açmaz, mevcut HTTP webhook akışını kullanır. Normal başlangıçta yapılandırılmış Telegram webhook kaydı da denenir.

Veritabanı kullanan testler için ayrı bir test DB'si ve `DATABASE_URL` hazırlayın; testleri üretim veritabanında çalıştırmayın. CI akışı PostGIS servisi üzerinde migrasyon, seed, lint, build ve test çalıştırır; lint sürümü workflow'da `v2.12.2` olarak sabitlenmiştir.

## Sorun giderme

| Belirti | Kontrol |
| --- | --- |
| JWT secret hatası | `USER_JWT_SECRET` boş olmamalı ve en az 32 byte içermeli |
| DB bağlantı hatası | DB hazır mı, port doğru mu, host yerelde `localhost`/Compose içinde `db` mi? |
| PostGIS/extension izin hatası | Sunucuda eklenti paketleri ve migrasyon hesabının gerekli izinleri var mı? |
| GeoIP dosyası açılamıyor | Çalışma dizini `core` mu, `static/data/GeoLite2-City.mmdb` mevcut mu? |
| Video işlenmiyor | `ffmpeg`, `ffprobe`, desteklenen codec'ler ve yazılabilir disk alanı var mı? |
| `ai.generate_text` görünmüyor | Sağlayıcı değişkenleri sunucu sürecine/konteynere iletilmiş mi; yeniden başlatıldı mı? |
| MCP 400/404 | Initialize sonrası doğru `MCP-Session-Id` gönderiliyor mu, oturum hâlâ geçerli mi? |
| MCP 403 | İstek Origin'i MCP izin listesinde mi? |
| MCP GET 405 | HTTP transport için POST kullanılıyor mu? |
| Upload 413 | Reverse proxy/ingress request body sınırı kapatılmış mı? |
| Özel medya 401/403 | Bearer token ve albüm/sohbet erişimi geçerli mi? |
| Port kullanımda | Yerel süreç ile Compose aynı anda 3001/3002'yi kullanıyor mu? |
