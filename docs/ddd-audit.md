# DDD Audit

## Katmanlar

- Domain: `domain/`
  - Framework, HTTP, DB, ORM import etmez.
  - Kullanıcı value object, interaction rule ve domain eventleri burada: `domain/user`, `domain/events`.
- Application: `application/`
  - Port arayüzleri `application/ports` altında.
  - Use case orkestrasyonu `application/usecases` altında.
  - MCP application adapterleri `application/mcpserver` altında.
- Infrastructure: `infrastructure/`
  - DB, captcha, token, hash, id üretimi, socket, external API ve composition root burada.
- Interface: `routes/`, `routes/handlers`, `middleware`, `mcp`
  - HTTP/Fiber request okuma, response yazma ve transport adapterleri burada kalır.

## Aggregate Sınırları

- User aggregate: `models.User`, `Story`, `Wallet`, user preference flags, user domain kind, registration event.
- Post aggregate: `models/post.Post`, poll payloads, post engagement operations.
- Media aggregate: `models/media.Media` ve file metadata.
- Engagement aggregate: `models.Engagement`, `EngagementDetail`.
- Chat aggregate: `models/chat.Chat`, participant ve message payloadları.
- Payment aggregate: `models/payment.PaymentMethod`.
- Taxonomy aggregate: pillar, cluster, entity, synonym, intent.
- Notification aggregate: notification payload ve delivery state.

## Düzeltilen İhlaller

- Wire kaldırıldı; dependency composition artık `infrastructure/bootstrap` altında manuel.
- `domain/` altında framework/DB/HTTP/ORM bağımlılığı yok.
- User registration/login akışında value object doğrulaması domain tarafına taşındı.
- Captcha, password hash, token ve public id üretimi application portları üzerinden infrastructure adapterlarına bağlandı.
- Repository arayüzleri application portlarına alındı; somut implementasyonlar `infrastructure/repositories` altında kaldı.
- Registration için `user.registered` domain eventi eklendi.
- Controller register/login akışında request okur, typed input oluşturur ve use case metodunu çağırır.
- Repository içindeki reCAPTCHA HTTP çağrısı infrastructure adapterına taşındı.
- Application use case paketleri `services/user` altından `application/usecases` altına taşındı.
- DB, socket ve Telegram adapterları `services/*` altından `infrastructure/*` altına taşındı.
- AI, push ve repository adapterları kök paketlerden `infrastructure/*` altına taşındı.
- AI use case somut provider registry yerine `ports.TextGenerator` arayüzüne bağlandı.
- Telegram webhook handler somut bot servisi yerine application portuna bağlandı.
- Application use case paketlerinin somut repository, GORM, HTTP ve Fiber bağımlılığı kaldırıldı.
- System/VAPID/payment initial-sync handlerlarının GORM erişimi system use case ve repository portuna taşındı.
- Follow, like, block, subscribe ve chat self-interaction iş kuralları domain interaction rule olarak eklendi.
- Follow/reaction/block/subscribe domain eventleri eklendi.
- Mimari sınırlar `architecture/layer_test.go` ile test altına alındı.
- Üretim akışındaki ani durdurmalar DB, notification ve seeder tarafında error dönüşüne çevrildi.

## Sınır Notu

- `models/` mevcut API ve GORM migration uyumluluğu için persistence model olarak korunur.
- Saf domain davranışı yeni `domain/` paketlerinde tutulur; yeni iş kuralı eklenirken `models/` içine framework bağımlı domain davranışı eklenmemelidir.
