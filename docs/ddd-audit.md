# DDD Mimari Durum ve Geçiş Rehberi

## Durum özeti

Core bugün **katmanlı, port-adapter yönü belirginleşmiş fakat DDD dönüşümü devam eden hibrit bir sistemdir**. Seçili iş kuralları saf domain paketlerine çıkarılmış, inbound adapterler infrastructure'dan ayrılmış ve kritik yazma akışlarının bir bölümü açık transaction sınırlarına alınmıştır. Buna karşılık `models/` hâlâ GORM şeması, ilişki grafiği ve mevcut JSON/API uyumluluğunun merkezidir; application portları ile use case'lerin önemli bir kısmı da `models/*` tiplerini kullanmaya devam eder.

Bu nedenle mevcut durum “full DDD tamamlandı” olarak değerlendirilmemelidir. Özellikle bütün persistence entity'leri domain aggregate değildir, tüm bounded contextler bağımsız modele sahip değildir ve domain eventleri için kalıcı outbox/teslimat garantisi henüz yoktur.

## Güncel katmanlar ve bağımlılık yönü

### Inbound adapterler — `adapters/inbound/`

- `adapters/inbound/http` Fiber middleware, router, route ve handler'larını içerir. HTTP/form/JSON alanlarını okuyup application inputlarına çevirir; hata ve response sözleşmesini burada eşler.
- `adapters/inbound/mcpserver` MCP server ve tool girişlerini application use case'lerine bağlar. MCP şeması ve transport değerlerinin yorumlanması bu katmandadır.
- Legacy alan uyumluluğu burada kalır. Örneğin report isteğindeki eski `reason` alanının `kind` ve `description` alanlarına çevrilmesi domain'in değil inbound adapterin sorumluluğudur.
- Inbound paketler `core/infrastructure`, somut repository veya GORM import etmez. Use case ve application portları üzerinden içeriye çağrı yapar.

### Application — `application/`

- `application/usecases` iş akışını orkestre eder: yetki kontrolü yapar, domain value object/policy'lerini çalıştırır ve portları çağırır. Infrastructure, Fiber, HTTP, GORM ve legacy `core/helpers` bağımlılığı taşımaz.
- `application/ports` repository ve dış servis sınırlarını tanımlar: persistence, auth, dosya, AI, Telegram ve moderasyon portları burada bulunur.
- `application/types` cursor, filtre, actor ve küçük read-result DTO'larını içerir. Bu paketin bütün transitif bağımlılık grafiği `models/*`, inbound, infrastructure, Fiber ve GORM'dan arındırılmıştır; architecture testi bu sınırı kilitler.
- Public post/timeline/search sorguları `application/ports.PublicPostReader` üzerinden persistence-free `application/types.PublicPost` projection'ı döndürür. Bu sınır internal UUID, hesap/rol/tercih/broadcast alanları ile dosya saklama yolu ve orijinal adının HTTP/MCP çıktısına taşınmasını engeller; eski `id` anahtarları public ID, poll choice kimlikleri ise typed opaque token taşır.
- Model bileşimli eski timeline/post sonuçları geçici olarak `application/legacyviews` altında karantinadadır. Bu paket yeni sözleşmeler için örnek değildir; yalnız mevcut API JSON şeklini kırmadan bounded-context bazlı projection'lara geçebilmek için açık teknik borç sınırıdır.
- Geçiş tamamlanmış değildir: `application/ports/repositories.go`, `application/legacyviews` ve bazı use case imzaları hâlâ `models.User`, `models/post.Post`, `models.EngagementDetail` gibi persistence şekilli tipler kullanır. Yeni çalışmalar bu bağımlılığı büyütmemeli; bounded context bazında domain command/result veya application read model tiplerine geçmelidir.
- Moderasyon portu daha ileri bir örnektir: command/filter alanlarında `domain/moderation` tiplerini, dışarı dönen veride persistence entity yerine application read view'larını kullanır. Mevcut named-map view'lar legacy JSON şekli için ara uyumluluk projection'ıdır; yeni query sözleşmeleri mümkün olduğunda typed read model kullanmalıdır.

### Domain — `domain/`

Domain paketleri transport, application, infrastructure, GORM ve GORM-tag'li `models` tiplerinden bağımsızdır. Buradaki “saflık” persistence ve framework bağımsızlığını ifade eder; bütün paketlerin yalnız Go standart kütüphanesini kullandığı anlamına gelmez. Örneğin wallet kesin para hesabı için `decimal`, taxonomy slug üretimi için küçük bir domain utility bağımlılığı kullanır. `domain/moderation` için yalnız standart kütüphane kullanımı ayrıca mimari test ile korunur.

Mevcut domain çekirdekleri şunlardır:

- `domain/user`: kayıt ve credential normalizasyonu, domain/tenant sınıflandırması, profil ve gizlilik value object'leri, preference flags, self-interaction kuralları, reciprocal engagement eşlemesi, set/toggle ilişki intent'i, idempotent referral ödül talimatı ve user domain event tanımları.
- `domain/moderation`: report target, kind, description ve status value object'leri; `Report` aggregate'i; self-report, status transition ve resolution/visibility politikaları.
- `domain/media`: upload metadata invariantları ve dosya erişim politikası. Dosya sistemi, HTTP serving ve DB lookup bu paketin dışında kalır.
- `domain/wallet`: taraf, minimum tutar, self-transfer, bakiye yeterliliği ve `NUMERIC(38,18)` gösterim sınırlarını doğrulayan `Transfer`/money kuralları ile tekrar güvenli para komutları için `IdempotencyKey` value object'i.
- `domain/post`: geçerli post kind sözlüğü ve parse kuralı. Bu paket henüz bütün post davranışlarını içeren tam bir Post aggregate'i değildir.
- `domain/taxonomy`: slug normalizasyon kuralları. Taxonomy persistence grafiğinin tamamı henüz domain modeline taşınmış değildir.
- `domain/events`: domain eventleri için ortak, küçük event sözleşmesi; tek başına bounded context değildir.

Bu paketlerin olgunluğu aynı değildir. Şu anda açık aggregate davranışının en net örneği moderasyondaki `Report` aggregate'idir. User, media, wallet, post ve taxonomy paketlerinin çoğu value object/policy düzeyindedir; persistence model kümelerini sırf ilişkili tablolar oldukları için aggregate olarak kabul etmemek gerekir.

### Infrastructure ve composition — `infrastructure/`

- `infrastructure/repositories` application portlarının GORM implementasyonlarını, entity-domain/read-model eşlemelerini ve transaction/locking ayrıntılarını içerir.
- `infrastructure/db`, `auth`, `identity`, `media`, `ai`, `socket`, `push`, `bot` ve `geoip` DB veya dış sistem adapterleridir.
- Repository adapterleri inbound/Fiber/multipart/HTTP transport bağımlılığı taşımaz. Network kullanan outbound adapterler kendi infrastructure paketlerinde kalır.
- `infrastructure/bootstrap` composition root'tur: DB ve dış kaynakları açar, somut adapterleri kurar, use case'lere enjekte eder, HTTP/MCP inbound adapterlerini bağlar ve kapanış kaynaklarını toplar.
- Infrastructure ağacı içinde inbound adapter import etmesine izin verilen tek yer `infrastructure/bootstrap` ağacıdır. İş kodu bootstrap'a veya inbound katmana geri bağımlı olmamalıdır.
- Background worker'lar ihtiyaç duydukları dar port/dependency setini dışarıdan alır; bootstrap somut adapterleri bu portlara bağlar. Media processor serve/shutdown yaşam döngüsüne bağlıdır. News ve broadcast dependency setleri kompoze edilse de gizli ağ işiyle startup'ı yavaşlatmamak için normal server başlangıcında otomatik zamanlanmaz.

Bu yönler `architecture/layer_test.go` tarafından paket import grafiği üzerinden korunur. Guard'lar domain'in dış katmanlara, inbound'un infrastructure/repository/GORM'a, repository adapterlerinin inbound/Fiber/HTTP'a, application types'ın transport/infrastructure/GORM'a ve use case'lerin infrastructure/helpers'a bağlanmasını engeller.

## Legacy `models/` persistence uyumluluk sınırı

`models/` bugün bilinçli bir compatibility boundary'dir:

- GORM tablo/kolon tag'leri, association'lar, preload için gereken ilişki grafikleri ve migration uyumu burada korunur.
- Mevcut JSON alan adları ve eski API response şekillerinin önemli bölümü bu tiplerden gelir.
- Domain'e taşınmış kurallar için model tarafında geçici delegasyon shim'leri bulunabilir. Örneğin persistence-owned `models.ReportStatus` kendi tip kimliğini ve DB/JSON davranışını korurken transition kuralını `domain/moderation`a delege eder.
- Infrastructure repository'leri domain değerleri ile persistence entity'leri arasındaki mapping'i yapar. GORM entity'si domain paketine sokulmaz.

Bu sınır kalıcı domain tasarımı olarak görülmemelidir. Yeni invariant, state transition veya policy doğrudan `models/` içine eklenmemelidir. Aynı şekilde yeni bir application portu, yalnız kolay olduğu için GORM entity'sini dışarı döndürmemelidir. Eski API veya DB sözleşmesini korumak gerekiyorsa uyumluluk mapping'i inbound/application read model veya infrastructure adapterinde açıkça yapılmalıdır.

## Kritik transaction sınırları

Aşağıdaki liste kodda doğrulanmış, yüksek etkili sınırları özetler; bütün repository transactionlarının eksiksiz envanteri değildir.

| Akış | Atomik sınır | Concurrency/idempotency notu |
| --- | --- | --- |
| Contentable post oluşturma | Post, subtype/payload, ilişkili DB media kayıtları ve comment ise parent engagement detail/sayacı `PostRepository.CreateContentablePost` içinde tek DB transactionında yazılır. | Dosya sistemi DB transactionı değildir; commit olmazsa oluşturulan upload'lar compensation ile silinir. Yalnız dış notification teslimi commit sonrasında best-effort çalışır. |
| Comment silme | Comment soft-delete'i, comment'e özel parent engagement detail'i ve parent comment sayacı tek repository transactionında güncellenir. | Canonical parent aggregate lock önce alınır; herhangi bir detail/counter hatası soft-delete'i de rollback eder. Legacy NULL-key detail yalnız deterministik fallback ile eşlenir. |
| Reciprocal user ilişkisi | Follow/like/dislike/block/subscribe gibi iki yönlü engagement satırları ve sayaçları `ApplyReciprocalUserInteraction` içinde aynı transactionda güncellenir. | İki user aggregate'i deterministik biçimde kilitlenir; domain set/toggle intent'i kilit altında mevcut duruma uygulanır. Like ve dislike dedupe anahtarları engagement kind ile ayrılır. |
| Post bahşiş (tip) transferi | Payer/payee bakiyeleri ile tip engagement kaydı `PostRepository.Tip` içinde tek transactionda yazılır. | Post ile iki kullanıcı row lock alır; kullanıcılar sabit sırada kilitlenir. Tutar/bakiye invariantı `domain/wallet.Transfer` ile doğrulanır. Payer kapsamlı idempotency key aynı isteğin bakiyeyi ikinci kez düşürmesini engeller; farklı payload ile tekrar kullanım conflict olur. |
| Referral ödülü | Referrer bakiyesi ile referral engagement detayı `UserRepository.AddReferral` içinde tek transactionda yazılır. | Domain referral kimliği kararlı dedupe key üretir; advisory lock, sıralı `NO KEY UPDATE` user lock'ları ve unique detail anahtarı eşzamanlı tekrarları tek ödüle indirir. Bildirim yalnız commit sonrasında best-effort gönderilir. |
| Report oluşturma ve moderasyon | Pending report upsert'i, resolve, opsiyonel post visibility ve review metadata transaction içindedir. | Submit/resolve/hide/unhide aynı target advisory lock anahtarını kullanır; ters row-lock sırası engellenir. Aynı-status retry visibility ve resolution işini atlamaz; aynı reporter/target pending tekrarları idempotent update olur. |
| Event RSVP ve poll vote | RSVP dedupe/capacity/attendee güncellemesi event seviyesinde kilitli transactiondadır. Poll vote satırı ile choice sayacı birlikte commit/rollback olur. | PostgreSQL advisory/row lock kullanılır; RSVP'nin test/non-PostgreSQL yolu process lock ile serileştirilir. |
| Tekil view ve engagement sayaçları | Dedupe detail insert'i ile aggregate counter artışı aynı transactiondadır; detail silme ile counter azaltma da birlikte yürür. | Bütün engagement türleri aynı canonical owner lock anahtarını kullanır. `(contentable_type, contentable_id)` unique index aggregate tekilliğini DB'de de zorlar; migration eski duplicate aggregate'leri ve sayaçlarını kayıpsız birleştirir. |
| Media yaşam döngüsü | File metadata ile media row'u birlikte yazılır; worker claim `FOR UPDATE SKIP LOCKED` ile alınır; final metadata ve processing status birlikte güncellenir. | Fiziksel dosya DB dışında kaldığı için hata yolunda compensation gerekir. Claim/ready transactionları birden fazla worker'ın aynı işi sahiplenmesini önler. |
| Disappearing chat message | İlk açma/view/expiry başlangıcı ve batch expiry sırasında message, unread/pin/last-message güncellemeleri transaction içinde tutulur. | Row lock ve idempotent koşullu update tekrar açma/expire yarışlarını sınırlar; fiziksel attachment silinmez. |

Transaction sınırı repository adapterinde bulunabilir fakat hangi verilerin birlikte tutarlı olması gerektiği domain/application tarafından tanımlanmalıdır. Uzun network çağrıları DB transactionı içine alınmamalıdır. DB dışı zorunlu side effect için açık compensation veya güvenilir post-commit teslimat mekanizması gerekir.

Domain event interface'i ve user eventleri mevcuttur; ancak bootstrap şu anda `UserService` için kalıcı bir publisher bağlamadığında varsayılan publisher no-op'tur. Bu yapı durable event delivery veya outbox garantisi olarak yorumlanmamalıdır. Para, bildirim veya entegrasyon açısından kaybedilemez eventler kullanılacaksa aynı DB transactionında outbox kaydı ve retry/idempotent consumer tasarımı eklenmelidir.

## Yeni geliştirme kuralları

1. **İş kuralı önce domain'de tanımlanır.** Yeni invariant veya transition constructor, value object, entity/aggregate metodu ya da policy olarak ilgili `domain/<context>` paketine eklenir ve saf unit test ile korunur.
2. **Inbound yalnız çeviri yapar.** HTTP/MCP doğrulaması, legacy field mapping'i, principal çıkarımı ve response/error mapping'i adapterde kalır. Handler repository, GORM veya infrastructure servisi çağırmaz.
3. **Use case orkestrasyon yapar.** Yetki, akış sırası, domain çağrıları ve port koordinasyonu application katmanındadır. Use case içinde Fiber/HTTP, SQL/GORM, dosya sistemi veya `core/helpers` kullanılmaz.
4. **Port use case ihtiyacına göre tasarlanır.** Yeni port imzalarında persistence model yerine domain command/value veya application read model kullanılır. Portlar somut repository implementasyonunun mevcut metodlarını kopyalayan geniş interface'lere dönüşmemelidir.
5. **Persistence mapping infrastructure'dadır.** GORM tag'i, preload, SQL, row/advisory lock ve DB hata çevirisi repository adapterinde kalır. Domain tipi GORM'u; inbound tipi repository'yi bilmez.
6. **`models/` yalnız uyumluluk içindir.** Yeni davranış eklenmez. Geçici shim eklenirse hangi domain kuralına delege ettiği test edilir ve kaldırılma kriteri migration planına yazılır.
7. **Aggregate invariantı tek transactionda korunur.** Read-modify-write akışında gerekli row/advisory lock, deterministik lock sırası ve retry idempotency tanımlanır. Counter/detail veya iki yönlü ilişki gibi beraber doğru kalması gereken kayıtlar ayrı commit edilmez.
8. **DB dışı side effect açıkça sınıflandırılır.** Notification, event, dosya ve network çağrısının pre-commit, post-commit, compensation veya outbox davranışı kod ve testte görünür olmalıdır.
9. **Her sınır kendi seviyesinde test edilir.** Domain kuralı unit test; use case port fake'i; transaction/concurrency repository integration testi; eski API şekli handler contract testi; bağımlılık yönü architecture testi ile korunur.
10. **Composition yalnız bootstrap'ta yapılır.** Handler, worker veya repository kendi somut dependency'sini üretmez; yeni adapter `infrastructure/bootstrap` üzerinden bağlanır ve gerekiyorsa kapanış kaynağı `App.Close`/server shutdown akışına eklenir.

## Aşamalı model-to-domain migration ve çıkış kriterleri

Migration tablo bazında değil bounded context/use case dilimi bazında yapılmalıdır. Her dilim aşağıdaki aşamalardan geçer.

### 1. Davranışı sabitle

- Mevcut API/DB davranışı için contract ve repository integration testleri vardır.
- İş invariantları, geçerli state transitionları, concurrency riski ve idempotency anahtarları yazılıdır.
- Hangi `models/*` tiplerinin yalnız persistence, hangilerinin yanlışlıkla application/domain davranışı taşıdığı envanterlenmiştir.

**Aşama çıkışı:** Aynı davranışı koruyan otomatik test olmadan taşıma başlamaz.

### 2. Domain çekirdeğini çıkar

- Invariantlar `domain/<context>` altında framework/persistence bağımsız value object, policy veya aggregate davranışına taşınmıştır.
- Domain unit testleri başarı, sınır ve geçersiz transition vakalarını kapsar.
- Application aynı kurala birden fazla girişten ulaşıyorsa tek domain doğrulamasını kullanır; transport doğrulaması tek güven kaynağı değildir.

**Aşama çıkışı:** İlgili iş kararını veren `models`, handler veya repository kodu kalmaz; bu katmanlar yalnız delegasyon/mapping yapar.

### 3. Application sözleşmesini ayır

- Use case input/output ve repository portları ilgili context için `models/*`, GORM, Fiber ve HTTP tipi taşımaz.
- Command tarafı domain değerleri veya application command'ları, query tarafı açık read model/projection kullanır.
- Principal ve authorization bilgisi persistence user entity'si yerine use case'in ihtiyaç duyduğu minimum application tipiyle taşınır.

**Aşama çıkışı:** `go list`/architecture guard ve kod araması, ilgili application diliminden persistence model importunun kalktığını doğrular.

### 4. Persistence adapterini izole et

- Domain ↔ GORM entity mapping'i tek yönü açık fonksiyonlarla infrastructure repository'sindedir.
- Aggregate save/load ve transaction sınırı aynı invariant kapsamına uyar; concurrency ve rollback integration testleri vardır.
- API/DB uyumluluğu gerekiyorsa legacy kolon ve JSON şekli adapter/read-model mapper üzerinden korunur; domain tipi şemaya göre eğilip bükülmez.

**Aşama çıkışı:** İlgili GORM entity yalnız infrastructure/persistence compatibility alanında kullanılır; handler ve domain tarafından import edilmez.

### 5. Compatibility shim'ini emekliye ayır

- Eski ve yeni okuma/yazma yollarında veri parity'si ölçülmüş, gerekiyorsa backfill tamamlanmış ve rollback planı denenmiştir.
- Eski API alanı kaldırılacaksa version/deprecation süresi bitmiş; korunacaksa kalıcı inbound mapper/read model olarak açıkça sahiplenilmiştir.
- `models` üzerindeki domain delegasyon shim'i ve kullanılmayan association davranışları kaldırılmıştır.
- Kaybedilemez domain eventleri için no-op publisher yerine outbox/retry ve idempotent consumer devrededir.
- Context'e özel architecture guard, application/domain'e persistence tipinin geri girmesini engeller.

**Bounded context migration çıkışı:** Domain davranışı ve aggregate sınırı tek kaynak, application sözleşmesi persistence bağımsız, repository yalnız adapter, transaction/concurrency testleri yeşil, legacy shim ve çift yazma yolu kaldırılmış olmalıdır. Bu kriterlerin tamamı sağlanmadan context “DDD'ye taşındı” kabul edilmez.
