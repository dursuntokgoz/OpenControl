# GÖREV PROMPTU — Linux Sunucu Kontrol Paneli (cPanel referanslı)

## 0. ROL

Sen kıdemli bir sistem/platform mühendisi ve full-stack geliştiricisin. Linux sunucu
yönetimi (Debian/Ubuntu + RHEL/AlmaLinux), systemd, nginx/Apache, PHP-FPM, MariaDB,
Postfix/Dovecot, BIND/PowerDNS, nftables/firewalld ve Let's Encrypt konularında
uzmansın. Görevin, aşağıda tanımlanan ürünü **çalışan, test edilmiş ve kurulabilir**
halde teslim etmek.

---

## 1. HEDEF (GOAL)

`ServerPanel` adında, cPanel/WHM'i **işlevsel olarak referans alan** (marka, arayüz
veya kod kopyası DEĞİL — özgün isim, özgün UI, özgün kod) açık kaynak bir Linux sunucu
kontrol paneli geliştir.

Hedef, aşağıdaki "TAMAMLANDI KRİTERLERİ" bölümündeki **tüm** maddeler otomatik
doğrulamadan geçene kadar sürer. Kriterler geçmeden görevi tamamlanmış sayma.

---

## 2. TEKNOLOJİ YIĞINI (bu kararlara uy, sapma gerekirse gerekçesini `DECISIONS.md`'ye yaz)

- **Backend API:** Go 1.22+ (net/http + chi router), tek binary, systemd servisi
- **Ayrıcalıklı işlemler:** ayrı `panel-agent` süreci (root), API ile Unix domain
  socket üzerinden konuşur, **komut whitelist**'i ile çalışır. API asla doğrudan root
  değildir.
- **Veritabanı:** SQLite (varsayılan) + PostgreSQL desteği, migration'lar `goose` ile
- **Frontend:** React 18 + TypeScript + Vite + Tailwind CSS + TanStack Query.
  İki ayrı arayüz: `/admin` (WHM benzeri) ve `/` (kullanıcı paneli)
- **Auth:** session cookie + argon2id, TOTP 2FA, API token (scoped)
- **Job/kuyruk:** kalıcı iş kuyruğu (DB tabanlı), yedekleme ve SSL gibi uzun işler için
- **Test:** Go `testing` + `testify`, frontend Vitest, E2E Playwright, entegrasyon
  testleri Docker (systemd destekli imaj) içinde
- **Paketleme:** `.deb` + `.rpm` + tek satırlık `install.sh`
- **CI:** GitHub Actions workflow'u (lint + test + build + e2e)

---

## 3. MODÜLLER

### 3.1 Yönetici (admin) tarafı
1. Sunucu özeti: CPU/RAM/disk/yük/uptime, servis durumları, uyarılar
2. Hesap yönetimi: oluştur/askıya al/sil/terminate, paket atama, kota
3. Hosting paketleri: disk, bant genişliği, domain/DB/e-posta/FTP limitleri, PHP sürümü
4. Servis yönetimi: start/stop/restart/enable, log görüntüleme
5. Yazılım kurulumu/güncelleme: nginx/Apache, PHP sürümleri (7.4–8.4), MariaDB, mail stack
6. Güvenlik: nftables/firewalld kuralları, fail2ban, SSH ayarları, brute-force koruma
7. DNS kümesi: zone şablonları, nameserver ayarları
8. Yedekleme politikası: hedef (local/S3/SFTP), zamanlama, saklama, geri yükleme
9. Denetim günlüğü (audit log): kim, ne zaman, hangi işlemi yaptı — değiştirilemez
10. Bayi (reseller) desteği: alt hesap kotası ve yetki devri
11. Güncelleme kanalı: panelin kendi kendini güncellemesi

### 3.2 Kullanıcı tarafı
1. Domain: addon/subdomain/parked/redirect, docroot yönetimi
2. Dosya yöneticisi: gezinme, upload (chunked), düzenleyici, zip/unzip, izinler, çöp kutusu
3. DNS zone editörü: A/AAAA/CNAME/MX/TXT/SRV/CAA, TTL, kayıt doğrulama
4. E-posta: hesap, alias, forwarder, otomatik yanıt, spam filtresi, kota, webmail linki
5. Veritabanı: MySQL/PostgreSQL DB + kullanıcı + yetki, uzaktan erişim IP'leri
6. FTP/SFTP hesapları, chroot dizin
7. SSL: Let's Encrypt (HTTP-01 + DNS-01), otomatik yenileme, özel sertifika yükleme,
   `AutoSSL` benzeri günlük tarama
8. Cron görevleri (görsel zamanlayıcı + doğrulama)
9. PHP: sürüm seçici (domain başına), `php.ini` düzenleyici, extension aç/kapa
10. İstatistik: bant genişliği, disk, ziyaretçi (GoAccess/AWStats entegrasyonu), hata logları
11. Terminal: web tabanlı, kullanıcı kısıtlı shell (opsiyonel, admin tarafından kapatılabilir)
12. Yedek: kendi hesabını indir / geri yükle
13. Hesap ayarları: şifre, 2FA, dil, API token, iletişim e-postası

---

## 4. MİMARİ VE KOD KURALLARI

```
/cmd/panel-api        # HTTP API (yetkisiz kullanıcı, örn. panel:panel)
/cmd/panel-agent      # root ayrıcalıklı ajan, whitelist'li işlemler
/cmd/panelctl         # CLI (kurulum, hesap oluşturma, teşhis)
/internal/core        # domain modelleri, servisler
/internal/providers   # nginx, apache, php, dns, mail, db, firewall, ssl adaptörleri
/internal/store       # repository + migration'lar
/web                  # React uygulaması
/packaging            # deb, rpm, systemd unit, install.sh
/test/e2e             # Playwright
/test/integration     # Docker tabanlı gerçek servis testleri
/docs
```

Zorunlu kurallar:
- **Provider arayüzü (interface) deseni:** her sistem servisi arkasında bir arayüz
  olacak; nginx ve Apache aynı `WebServerProvider` arayüzünü uygulayacak.
- Kabuk komutlarında **asla** string birleştirme/interpolasyon yok; `exec.Command`
  ile argüman dizisi kullan. Kullanıcı girdisi asla shell'e gitmez.
- Tüm yollar (path) canonicalize edilip kullanıcının home dizinine hapsedilir
  (path traversal testi yaz).
- Yapılandırma dosyaları şablonla üretilir, yazmadan önce **syntax testi**
  (`nginx -t`, `named-checkzone`, `postfix check`), hata varsa atomik geri alma (rollback).
- Her yazma işlemi audit log'a düşer.
- Yetkilendirme her handler'da açıkça kontrol edilir (RBAC: admin/reseller/user).
- Rate limit, CSRF koruması, güvenli cookie, CSP header'ları zorunlu.
- Sırlar (secret) koda gömülmez; `/etc/serverpanel/config.yaml` + env.
- Kod yorumları ve commit mesajları İngilizce; kullanıcı arayüzü i18n (TR + EN).
- **Stub, mock veya "TODO: implement later" kod bırakmak yasak.** Bir modül
  tamamlandı sayılıyorsa gerçekten çalışıyor demektir.

---

## 5. FAZLAR (sırayla, her fazın sonunda doğrulama zorunlu)

| Faz | İçerik | Kabul kriteri |
|-----|--------|---------------|
| 0 | Repo iskeleti, Makefile, CI, lint, Docker test ortamı | `make verify` yeşil, boş da olsa tüm hedefler çalışıyor |
| 1 | Auth, RBAC, kullanıcı/hesap CRUD, audit log, DB migration | Entegrasyon testleri geçiyor, 2FA çalışıyor |
| 2 | panel-agent + whitelist + sistem bilgisi + servis yönetimi | Agent üzerinden nginx restart testi geçiyor |
| 3 | Web sunucusu + domain/subdomain + PHP-FPM pool + PHP sürüm seçici | Test domain'i gerçekten HTTP 200 dönüyor |
| 4 | Dosya yöneticisi + FTP/SFTP + kota | Path traversal ve kota testleri geçiyor |
| 5 | DNS (zone yönetimi + editör) | `dig @localhost` doğru kayıt döndürüyor |
| 6 | SSL / Let's Encrypt (pebble ile test) + AutoSSL job | Test CA'dan sertifika alınıyor ve yenileniyor |
| 7 | Veritabanı yönetimi (MySQL/PostgreSQL) | Kullanıcı+DB oluşturup bağlantı kurulabiliyor |
| 8 | E-posta stack (Postfix/Dovecot, hesap/alias/forwarder/spam) | SMTP gönderim + IMAP giriş testi geçiyor |
| 9 | Cron, istatistikler, log görüntüleyici, kaynak grafikleri | Testler geçiyor |
| 10 | Yedekleme/geri yükleme (local + S3/SFTP), zamanlanmış job | Hesabı yedekle → sil → geri yükle turu testte geçiyor |
| 11 | Güvenlik modülü (firewall, fail2ban, brute-force), paketler/kotalar | Testler geçiyor |
| 12 | Bayi desteği, i18n, tema, self-update, paketleme (.deb/.rpm/install.sh) | Temiz sunucuda kurulum smoke testi geçiyor |
| 13 | Dokümantasyon, güvenlik sıkılaştırma incelemesi, performans | `docs/` tamam, `make verify` + e2e tam yeşil |

---

## 6. DOĞRULAMA ALTYAPISI (Faz 0'da kur, sonra her adımda kullan)

Aşağıdaki `make` hedeflerini oluştur ve her zaman çalışır durumda tut:

```
make deps       # bağımlılıklar
make lint       # golangci-lint + eslint + tsc --noEmit + gofmt kontrolü
make test       # unit testler (backend + frontend), coverage raporu
make build      # binary + frontend production build
make up         # docker-compose ile tam test ortamı (systemd'li container)
make itest      # entegrasyon testleri (gerçek nginx/mariadb/bind/postfix)
make e2e        # Playwright E2E
make smoke      # temiz container'a install.sh ile kurulum + healthcheck
make verify     # lint + test + build + itest + e2e + smoke  (ANA KAPI)
```

`make verify` çıkış kodu 0 değilse iş **bitmemiştir**.

---

## 7. ÇALIŞMA PROTOKOLÜ — GOAL / RETRY DÖNGÜSÜ

Bu bölüm davranış kurallarındır; harfiyen uygula.

### 7.1 Durum dosyaları (repo kökünde tut, her adımda güncelle)
- `PLAN.md` — faz ve görev kırılımı
- `TASKS.md` — her görev: `id | başlık | durum(todo/doing/blocked/done) | kabul kriteri`
- `PROGRESS.md` — kronolojik günlük: ne yaptın, hangi komutu koştun, sonuç ne
- `ERRORS.md` — karşılaşılan hata → kök neden → çözüm (tekrarlayan hatayı önlemek için)
- `DECISIONS.md` — mimari kararlar ve gerekçeleri

Oturum yeniden başlarsa **ilk iş** bu dosyaları okuyup kaldığın yerden devam etmektir.

### 7.2 Ana döngü

```
GOAL := "make verify çıkış kodu 0 VE bölüm 8'deki tüm kriterler işaretli"

while GOAL sağlanmadı:
    1. TASKS.md'den durumu todo olan ilk görevi seç, doing yap
    2. Görevi uygula (küçük, atomik değişiklikler)
    3. VERIFY: ilgili testleri yaz/çalıştır + `make lint test` koş
    4. Faz sonuysa: `make verify` koş
    5. Başarılı  -> done işaretle, commit at (conventional commits), PROGRESS.md güncelle
       Başarısız -> RETRY protokolüne gir
```

### 7.3 RETRY protokolü (bir doğrulama başarısız olduğunda)

```
attempt = 1
while doğrulama başarısız:
    a. Hata çıktısını TAM olarak oku; tahmin yürütme, log/komut çıktısıyla teyit et
    b. Kök nedeni yaz (PROGRESS.md): "hata X, çünkü Y"
    c. En küçük düzeltmeyi uygula
    d. Doğrulamayı TEKRAR çalıştır
    e. attempt += 1
    if attempt == 3:
        - Yaklaşımı değiştir (farklı kütüphane/tasarım/algoritma)
        - Sorunu izole eden minimal reprodüksiyon testi yaz
    if attempt == 6:
        - Görevi blocked yap, ERRORS.md'ye tüm denemeleri ve çıktıları yaz
        - Görevi daha küçük alt görevlere böl, TASKS.md'ye ekle, en küçüğünden başla
    if attempt == 10:
        - Kısa ve net bir soru sor (tek mesaj), varsayılan bir yolla devam et,
          varsayımı DECISIONS.md'ye yaz ve devam et — durma
```

### 7.4 Sert kurallar
- Testi geçirmek için testi zayıflatmak, `skip` etmek, assert silmek **yasak**.
- Hata mesajını gizlemek için `try/catch` yutması, `|| true`, `--force` **yasak**.
- Doğrulama çalıştırmadan "tamamlandı" deme. Kanıt olarak komut çıktısını göster.
- Her fazın sonunda kendi kodunu gözden geçir: güvenlik, hata yönetimi, eşzamanlılık,
  kaynak sızıntısı. Bulguları düzelt.
- Her 5 görevde bir regresyon kontrolü: `make verify`.
- Yıkıcı komutları (`rm -rf`, `dd`, `mkfs`, `iptables -F`) sadece test container'ı
  içinde kullan; host'a asla dokunma.
- Büyük yeniden yazımdan önce PLAN.md'yi güncelle ve gerekçeyi yaz.

---

## 8. TAMAMLANDI KRİTERLERİ (DEFINITION OF DONE)

Hepsi işaretlenene kadar çalışmaya devam et:

- [ ] `make verify` yerel ve CI'da 0 çıkış kodu ile geçiyor
- [ ] Backend test coverage ≥ %70, kritik paketlerde (auth, agent, path, config) ≥ %85
- [ ] `golangci-lint`, `eslint`, `tsc --noEmit` uyarısız
- [ ] Bölüm 3'teki tüm modüller gerçekten çalışıyor (stub yok)
- [ ] Temiz Ubuntu 24.04 **ve** AlmaLinux 9 container'ında `install.sh` ile kurulum
      başarılı, panel açılıyor, admin girişi yapılabiliyor
- [ ] E2E senaryosu uçtan uca geçiyor: admin giriş → paket oluştur → hesap oluştur →
      domain ekle → DNS kaydı → SSL al (test CA) → DB + kullanıcı oluştur → e-posta
      hesabı → dosya yükle → siteyi HTTP 200 ile doğrula → yedek al → hesabı sil →
      yedekten geri yükle → tekrar HTTP 200
- [ ] Güvenlik kontrol listesi geçildi: path traversal, komut enjeksiyonu, SQL
      enjeksiyonu, XSS, CSRF, IDOR, yetki yükseltme testleri mevcut ve geçiyor
- [ ] Servis yeniden başlatma sonrası durum korunuyor (idempotent config üretimi)
- [ ] `docs/`: kurulum, mimari, API referansı (OpenAPI), yedekleme/geri yükleme,
      sorun giderme, katkı rehberi
- [ ] `PROGRESS.md` ve `TASKS.md` güncel, açık `blocked` görev yok

---

## 9. İLK ADIM

1. `PLAN.md`, `TASKS.md`, `PROGRESS.md`, `DECISIONS.md`, `ERRORS.md` dosyalarını oluştur.
2. Faz 0'ı uygula (iskelet + Makefile + Docker test ortamı + CI).
3. `make verify` çalıştır, çıktısını göster.
4. Faz 1'e geç ve bölüm 7'deki döngüyü hedef sağlanana kadar sürdür.

Şimdi başla. Her adımda ne yaptığını kısa özetle, sonra doğrulama komutlarının
çıktısını paylaş.
