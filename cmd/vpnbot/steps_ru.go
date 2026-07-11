package main

// stepsRU holds the Russian per-device setup instructions, keyed by platform.
var stepsRU = map[string]string{
	"ios": `🍎 Настройка iOS

Рекомендуемое приложение: Streisand (бесплатно) или Hiddify, оба в App Store.

1. Установите Streisand из App Store.
2. Скопируйте ссылку на подписку (отправьте /start, если нужна снова).
3. Откройте Streisand, нажмите ＋ (справа вверху), затем "Add from Clipboard".
   Или нажмите ＋ → "Scan QR Code" и отсканируйте QR из сообщения /start.
4. Выберите конфигурацию и нажмите Connect.`,

	"android": `🤖 Настройка Android

Рекомендуемое приложение: Hiddify или v2rayNG (оба бесплатны).

1. Установите Hiddify из Google Play или v2rayNG с GitHub.
2. Скопируйте ссылку на подписку (отправьте /start, если нужна снова).
3. Откройте приложение, нажмите ＋, затем "Add from clipboard" / "Import from link".
   Или нажмите ＋ → "Scan QR code" и отсканируйте QR из сообщения /start.
4. Нажмите кнопку питания, чтобы подключиться.`,

	"windows": `🪟 Настройка Windows

Рекомендуемое приложение: Hiddify (hiddify.com) или v2rayN.

1. Скачайте и запустите Hiddify для Windows.
2. Скопируйте ссылку на подписку (отправьте /start, если нужна снова).
3. В Hiddify: New Profile → вставьте ссылку → Add.
4. Выберите профиль и нажмите Connect.`,

	"macos": `💻 Настройка macOS

Рекомендуемое приложение: Hiddify (hiddify.com), Apple Silicon и Intel.

1. Установите Hiddify для macOS.
2. Скопируйте ссылку на подписку (отправьте /start, если нужна снова).
3. В Hiddify: New Profile → вставьте ссылку → Add.
4. Выберите профиль и нажмите Connect.`,

	"linux": `🐧 Настройка Linux

Рекомендуемое приложение: Hiddify (AppImage с hiddify.com) или sing-box CLI.

Hiddify:
1. Скачайте AppImage Hiddify, сделайте его исполняемым и запустите.
2. New Profile → вставьте ссылку на подписку → Add → Connect.

sing-box (CLI): добавьте /sing-box к вашей ссылке и запустите:
  curl -L "YOUR_LINK/sing-box" -o config.json
  sing-box run -c config.json
Ссылка показана выше, или отправьте /start, чтобы получить её.`,
}
