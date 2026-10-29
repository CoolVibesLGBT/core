ÜLKELER ATLASI — ETKİLEŞİMLİ FOTOĞRAF

AÇILIŞ
index.html güncel bir tarayıcıda doğrudan açılabilir; mevcut AVIF görseller,
derinlik verileri ve çalışma kodu dosyanın içindedir. Paket kurulumu gerekmez.

Daha hafif atlas.html, kaynakları ayrı dosyalardan yükler. Proje klasöründe:
python3 -m http.server 8000
Sonra http://localhost:8000/atlas.html adresini açın.

KONTROLLER
Fareyi görsel üzerinde gezdirin. Telefonda sürükleyin; iki parmakla hafifçe
yakınlaşın. Çift tıklama, Home veya Esc görünümü ortalar. Otomatik gezinti
mevcut açıdan yumuşakça başlar. Telefon eğimi, destek ve hareket izni gerektirir.
Hareketi azalt sistem tercihi desteklenir.

Esneklik, Akışkanlık, Hareket hızı, Yakınlaşma ve Derinlik ayrı ayarlanır.
Sinematik / Canlı / Sakin profilleri tüm sürgülere birlikte uygulanır.
Ayarımı kaydet kişisel bir profili saklar; Kayıtlı ayarım geri çağırır.
Son kullanılan ayarlar tarayıcıda tutulur. Sıfırla varsayılanlara döndürür.

Görüntü kalitesi:
- Otomatik: Uzun süreli performans düşüşlerinde çizim çözünürlüğünü azaltır,
  kararlı performansta kademeli yükseltir. Hareketin zamanı değişmez.
- Yüksek: Cihaz piksel oranını ve bellek sınırını gözeten yüksek çözünürlük.
- Ekonomik: Daha düşük çizim çözünürlüğü.

GÖRSEL GEÇİŞİ VE VİDEO
Yeni görsel ve derinliği hazır olana kadar eski kare korunur. Sonra yumuşak
çapraz geçiş yapılır. Yükleme başarısızsa önceki görünüme dönülür.
Video kaydet, desteklenen tarayıcılarda en fazla 10 saniyelik bir video üretir.
Kaydı bitir ile erken durdurabilirsiniz. Çıktı en fazla 1280 piksel uzun kenar,
30 fps hedefiyle WebM veya tarayıcının desteklediği MP4 biçimindedir. Kayıt
sırasında ülke ve görsel seçimi kilitlenir. Sekme gizlenirse kayıt sonlandırılır.
Video indirmesi yeni bir dosyadır; kaynak AVIF dosyaları değiştirilmez.

AKIŞ MOTORU
Uygun WebGL float doku ve framebuffer desteğinde 96×64 GPU akış alanı kullanılır.
Destek gerçekten doğrulanamazsa 48×32 CPU motoruna otomatik dönülür. Nesne
derinliği, elastik kamera, yakınlaşma ve deformasyon ortak zaman adımlarıyla
ilerler. Akış alanı kübik örneklenir; güçlü esnemelerde kenar koruması uygulanır.
Bu, tek görselden ölçülmüş tam 3D model üretmez. Örtülü yüzeyleri yeniden
oluşturmaz; güçlü hareketlerde esneme görülebilir. AVIF indirmeleri sabittir.

KOD VE DERLEME
page.template.html — Arayüz ve derleme işaretçileri.
scripts/runtime/state.js — Ayarlar ve ortak durum.
scripts/runtime/flow.js — CPU/GPU deformasyon alanları.
scripts/runtime/renderer.js — Shader, dokular ve çizim.
scripts/runtime/motion.js — Kamera ve ortak animasyon zamanı.
scripts/runtime/features.js — Profiller, kalite, çapraz geçiş ve video.
scripts/runtime/viewer.js — Yükleme, ülke durumu ve çizim döngüsü.
scripts/runtime/controls.js — Fare, dokunma, klavye ve sensörler.
scripts/runtime/boot.js — Başlatma.

python3 scripts/build-page.py
Bu komut mevcut varlıklarla index.html ve atlas.html üretir; kaynak görsellere
ve derinlik dosyalarına yazmaz. Her iki HTML çalışma kodunu içinde taşır.
Üretilen HTML yerine şablonu ve runtime dosyalarını düzenleyin.
Katalog güncellemesi gerekirse ayrı olarak: python3 scripts/catalog.py

DOĞRULAMA
node scripts/qa-atlas.mjs
node scripts/qa-features.mjs
python3 scripts/qa-build-page.py
node scripts/qa-browser.mjs

İlk üç komut durum, fizik, zamanlama, kontroller, kayıt yaşam döngüsü,
GPU kaynak sahipliği ve derleme eşliğini sınar. Gerçek GPU shader çalışması
ve görsel sonuç için tarayıcı testi ayrıca gereklidir.
Tarayıcı testi kurulu Playwright/playwright-core kullanır; paket indirmez.
Gerekirse PLAYWRIGHT_MODULE ve ATLAS_BROWSER_PATH ortam değişkenlerini verin.
Rapor ve başarılı çalışmada ekran görüntüleri/video geçici atlas-browser-qa
klasörüne yazılır; ATLAS_QA_OUTPUT ile yer değiştirilebilir. Chrome'un sistem
izniyle engellenmesi testi başarılı saydırmaz; raporda blocked belirtilir.

VARLIKLAR VE KAPSAM
images/[ülke]_avatar.avif — 512×512
images/[ülke]_cover.avif — 1500×500
images/[ülke]_hero.avif — 1920×1280
depth/ — Yaklaşık gri tonlu derinlik verileri.
data/depth-regions/ — Sayısal derinlik tanımları.
data/catalog.json — Hazır/bekleyen varlıklar.
production/ — Üretim istemleri ve inceleme notları.

ISO 3166-1 kapsamındaki 249 ülke/bölge, ayrıca altı özel statülü yer ve
Birleşik Krallık’ın dört bileşeni için 259 kayıt ve toplam 777 görsel hazırdır.
Eklenen 64 kaydın üçer görseli incelendi; 777 yaklaşık derinlik haritası mevcuttur.
country-coverage-audit.html — Son kapsam raporu ve eklenen 64 kayıt.
production/expansion-final-prompts.json — Yerleşik Imagegen ile kullanılan son prompt seti.
Bu kapsam bağımsız devlet sayısı veya egemenlik tanıma listesi değildir.
XK, XN, XS, XA, XO ve XT özel uygulama kodlarıdır; resmî atanmış ISO kodları değildir.
Dosyaların hazır olması görsel içeriğin tamamının doğrulandığı anlamına gelmez.
Görseller temsili kültür/doğa seçkileridir; her il ve ilçenin eksiksiz miras
listesi veya gerçek bir kesintisiz coğrafi görünüm olarak değerlendirilmemelidir.

Referanslar:
https://research.un.org/en/unmembers/currentmembers
https://research.un.org/en/unmembers/observers
https://www.alanzucconi.com/2019/01/01/facebook-3d-photos-2/
https://developer.mozilla.org/en-US/docs/Web/API/OES_texture_float
https://developer.mozilla.org/en-US/docs/Web/API/MediaRecorder

360° PANORAMA
360° panorama düğmesi tek görseli sanal bir silindire sarar. Fare veya tek
parmakla sürükleyin; ok tuşlarıyla da dönebilirsiniz. Home / Esc kamerayı
başlangıca döndürür. Aynı düğme derinlikli fotoğraf moduna geri döner.
Kaynak dosya kırpılmaz veya değiştirilmez; kamera silindirin bir bölümüne bakar.
Bu mod yeni arka yüzler üretmez. Panorama dokusunda iki kenar yaklaşık %24 örtüştürülür. İçeriğe göre seçilen
geçiş yolu ve çok ölçekli harmanlama kullanılır; aynalı dönüş yoktur. Kaynak
görsel değiştirilmez; örtüşen kenar ayrıntıları panorama görünümünde birleşir.
Yeni coğrafi alan üretilmez; çok farklı kenarlarda kompozisyon farkı kalabilir.
