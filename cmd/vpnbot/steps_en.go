package main

// stepsEN holds the English per-device setup instructions, keyed by platform.
var stepsEN = map[string]string{
	"ios": `🍎 iOS setup

Recommended app: Streisand (free) or Hiddify, both on the App Store.

1. Install Streisand from the App Store.
2. Copy your subscription link (send /start if you need it again).
3. Open Streisand, tap ＋ (top-right), then "Add from Clipboard".
   Or tap ＋ → "Scan QR Code" and scan the QR from your /start message.
4. Select the config and tap Connect.`,

	"android": `🤖 Android setup

Recommended app: Hiddify or v2rayNG (both free).

1. Install Hiddify from Google Play, or v2rayNG from GitHub.
2. Copy your subscription link (send /start if you need it again).
3. Open the app, tap ＋, then "Add from clipboard" / "Import from link".
   Or tap ＋ → "Scan QR code" and scan the QR from your /start message.
4. Tap the power button to connect.`,

	"windows": `🪟 Windows setup

Recommended app: Hiddify (hiddify.com) or v2rayN.

1. Download and run Hiddify for Windows.
2. Copy your subscription link (send /start if you need it again).
3. In Hiddify: New Profile → paste the link → Add.
4. Select the profile and click Connect.`,

	"macos": `💻 macOS setup

Recommended app: Hiddify (hiddify.com), Apple Silicon and Intel.

1. Install Hiddify for macOS.
2. Copy your subscription link (send /start if you need it again).
3. In Hiddify: New Profile → paste the link → Add.
4. Select the profile and click Connect.`,

	"linux": `🐧 Linux setup

Recommended app: Hiddify (AppImage from hiddify.com), or the sing-box CLI.

Hiddify:
1. Download the Hiddify AppImage, make it executable, and run it.
2. New Profile → paste your subscription link → Add → Connect.

sing-box (CLI): append /sing-box to your link and run:
  curl -L "YOUR_LINK/sing-box" -o config.json
  sing-box run -c config.json
Your link is shown above, or send /start to get it.`,
}
