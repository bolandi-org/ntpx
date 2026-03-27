# راهنمای کامل راه‌اندازی هسته nptx (مخصوص شبکه‌های دارای محدودیت، CGNAT و Starlink)

این ابزار با تکنولوژی‌های فوق‌پیشرفته شامل NTPv4 Header Mimicry، UDP Spraying (Rain Technique)، و App-Layer Fragmentation ساخته شده تا بر روی مخابراتی‌ترین و بی‌ثبات‌ترین اینترنت‌های جهان (مثل استارلینک یا CGNATهای پشت‌سرهم) با پایداری کامل و بدون افت سرعت اجرا شود.

---

## دانلود آخرین نسخه (برای سرور خارج یا لوکال)

اگر به اینترنت آزاد دسترسی دارید، می‌توانید مستقیماً آخرین نسخه باینری را از گیت‌هاب دانلود کنید:

```bash
# دانلود آخرین باینری برای لینوکس (AMD64)
export REPO="bolandi-org/ntpx"
export LATEST_URL=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep "browser_download_url.*nptx_core_linux_amd64" | cut -d '"' -f 4)
wget -O nptx_core_linux_amd64 "$LATEST_URL"
chmod +x nptx_core_linux_amd64
```

---

## نصب کاملاً آفلاین (بدون نیاز به اینترنت و دانلود)

برای سرورهای ایران که دسترسی به اینترنت آزاد ندارند، فایل اجرایی `nptx_core_linux_amd64` (که از بخش Releases گیت‌هاب دانلود کرده‌اید) و فایل‌های `nptx.service` و `config.json` را در یک پوشه روی سرور قرار دهید.

سپس کدهای زیر را در همان پوشه اجرا کنید تا برنامه نصب و سرویس فعال شود:

```bash
# 1. قرار دادن باینری در مسیر اجرایی لینوکس
chmod +x nptx_core_linux_amd64
sudo cp nptx_core_linux_amd64 /usr/local/bin/nptx_core

# 2. انتقال فایل کانفیگ به پوشه تنظیمات سیستم
sudo mkdir -p /etc/nptx
sudo cp config.json /etc/nptx/config.json

# 3. نصب و راه‌اندازی سرویس Systemd برای اجرای خودکار
sudo cp nptx.service /etc/systemd/system/nptx.service
sudo systemctl daemon-reload
sudo systemctl enable nptx.service
sudo systemctl start nptx.service
sudo systemctl status nptx.service
```

---

## تنظیمات کانفیگ (`config.json`)

برای ویرایش تنظیمات، فایل کانفیگ را باز کنید: `nano /etc/nptx/config.json`

### نمونه کانفیگ سرور خارج (Server)

**نکته مهم:** مقدار پورت `local` را روی `123` بگذارید، زیرا ترافیک NTP منحصراً مربوط به پورت ۱۲۳ است تا از مسدودسازهای DPI به راحتی عبور کند.

```json
{
  "mode": "server",
  "local": "0.0.0.0:123",
  "remote": "",
  "streams": 16,
  "password": "یک_پسورد_بسیار_سخت",
  "routes": ""
}
```

### نمونه کانفیگ سرور ایران / کلاینت (Client)

در بخش `routes`، مشخص می‌کنید که کدام پورتِ داخلی در سرور ایران، باید به چه پورتی در سرور خارج وصل شود! (مثال: `7305:25566` یعنی پورت 7305 ایران وصل شود به پورت 25566 خارج).

```json
{
  "mode": "client",
  "local": "",
  "remote": "IP_SERVER_KHAREJ:123",
  "streams": 16,
  "password": "یک_پسورد_بسیار_سخت",
  "routes": "7305:25566,7306:51820"
}
```

بعد از هر بار تغییر در فایل کانفیگ، فراموش نکنید سرویس را ری‌استارت کنید: `sudo systemctl restart nptx.service`

---

## تست یک سناریوی کامل با WireGuard

فرض کنید قصد دارید ترافیک وایرگارد را از ایران به خارج تونل بزنید.
۱- وایرگاردِ سرور خارج را تنظیم کنید تا روی پورت `51820` فعال شود.
۲- در سرور ایران، فایل کانفیگ کلاینت nptx را باز کرده و بخش `routes` را اینطور وارد کنید: `"51820:51820"`.
۳- کانکشن Wireguard کلایت‌ها (ویندوز/موبایل دوستانتان) را تنظیم کنید تا به Endpoint: `IP_IRAN:51820` متصل شوند.

**فرآیند جریان ترافیک جادویی:**
موبایل → (Wireguard 1420 MTU) → سرور ایران ۱ (پورت ۵۱۸۲۰) → (هسته nptx ایران) → تکه‌شدن پکت 1420 به سایز امن 1124 → دریافت هدر NTPv4 دروغین → اسپری شدن روی ۱۶ سوکت موازی → استارلینک / DPI → سرور خارج (پورت ۱۲۳) → جمع‌آوری سوکت‌ها → تایید هویت ChaCha20 → چیدن دوباره قطعات از هم‌گسسته با تحمل خطای ۲۰ ثانیه‌ای → تحویل پکت کامل ۱۴۲۰ بایتی به وایرگارد روی پورت ۵۱۸۲۰!
