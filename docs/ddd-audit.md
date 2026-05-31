# DDD Audit

## Katmanlar

- Domain: `domain/`
  - Framework, HTTP, DB, ORM import etmez.
  - Şu an kullanıcı domain değerleri ve eventleri burada: `domain/user`, `domain/events`.
- Application: `application/`
  - Port arayüzleri `application/ports` altında.
  - MCP use case adapterleri `application/mcpserver` altında.
- Infrastructure: `infrastructure/`, `repositories/`, `services/db`, `services/socket`, `ai`, `push`
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
- User repository arayüzü application portuna alındı; somut implementasyon `repositories` tarafında kaldı.
- Registration için `user.registered` domain eventi eklendi.
- Controller register/login akışında request okur, typed input oluşturur ve use case metodunu çağırır.
- Repository içindeki reCAPTCHA HTTP çağrısı infrastructure adapterına taşındı.
- Üretim akışındaki gereksiz `panic` kullanımları DB, notification ve seeder tarafında error dönüşüne çevrildi.

## Kalan Kontrollü Borç

- `models/` halen GORM persistence modelidir; saf domain entity olarak kullanılmamalıdır.
- Bazı service/repository metodlarında post/media/chat iş kuralları halen karışık durur.
- Büyük taşıma riski nedeniyle mevcut davranışı bozmamak için refactor register/login ve user value object hattında başlatıldı.
