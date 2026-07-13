package main

var enMsg = msg{
	welcome:     "Welcome. Send the code your administrator gave you:\n/start <code>",
	codeInvalid: "That code isn't valid. Check it with your administrator.",
	codeClaimed: "That code has already been claimed.",
	helpUser: "Commands:\n" +
		"/start <code> - claim your subscription (a bare /start re-shows it)\n" +
		"/setup - how to connect on your device\n" +
		"/awg - get an AmneziaWG config for a location\n" +
		"/lang - change language (English / Русский)\n" +
		"/help - show this message\n",
	helpAdmin: "\nAdmin:\n" +
		"/add <username> - create a user, pick their locations, get a claim link\n" +
		"/list - show tracked users and their delivery status\n" +
		"/revoke <username> - rotate a user's key so their link stops working\n",
	notAuthorised:   "Not authorised.",
	panelDown:       "The panel is unavailable right now - please try again later.",
	panelShort:      "Panel unavailable",
	listEmpty:       "No users tracked yet.",
	listHeader:      "Tracked users:\n",
	listUnclaimed:   "unclaimed",
	listDelivered:   "delivered → chat %d",
	listLine:        "• %s - %s\n",
	revokeUsage:     "usage: /revoke <username>",
	revokeNoUser:    "No such user: %s",
	revokeFailed:    "Revoke failed: %s",
	revokeOK:        "Revoked %s. Their old subscription link no longer works.",
	accountNotFound: "Your account was not found. Please contact your administrator.",
	deliverCaption: "Your VPN subscription. Import this link into your client, or scan the QR:\n\n" +
		"%s\n\nNew here? Tap your device below for setup steps.",
	deliverAwgNote:  "\n\n🔐 Want WireGuard as well? Tap AmneziaWG below for a separate, DPI-resistant config.",
	setupChoose:     "Choose your device to see setup steps:",
	unknownPlatform: "Unknown platform",
	subLinkPrefix:   "Your subscription link:\n%s\n\n",
	multiServerNote: "ℹ️ Using several servers\n" +
		"We run more than one server, and they are all in your subscription. In your app pick \"Auto\" / \"Best Latency\" (a group, not a single server) to always use the fastest one. It switches over on its own if a server goes down. You can still pick a specific server by name.",
	steps:            stepsEN,
	addUsage:         "usage: /add <username>",
	addCreateFail:    "Could not create user: %s",
	addGrantPrompt:   "Grant locations for %s, then tap Done:",
	addBadRequest:    "Bad request",
	addNoUser:        "No such user",
	addUpdateFail:    "Update failed",
	addGranted:       "Granted",
	addRemoved:       "Removed",
	addPickFirst:     "Pick at least one location first",
	addDone:          "Created %s (%s).\nSend them this link:\n%s\n\n(or the code: %s)",
	btnDone:          "✅ Done",
	chooseLang:       "Choose your language:",
	langSet:          "Language set to English.",
	awgNotConfigured: "AmneziaWG is not available yet. Use /setup for your standard connection.",
	awgClaimFirst:    "Claim your subscription first with /start <code>, then use /awg.",
	awgNoLocations:   "You have no AmneziaWG locations available. Ask your administrator.",
	awgChoose:        "Choose a location for your AmneziaWG config:",
	awgProvisioning:  "Preparing your config...",
	awgFailed:        "Could not prepare the config right now - please try again later.",
	awgCaption: "AmneziaWG config for %s. Get the AmneziaVPN app (amnezia.org or your app " +
		"store), then tap + and import this file - or scan the QR below.",
	awgQRCaption: "Scan this in the AmneziaVPN app to import the config.",
	cmdStart:     "Claim or re-show your subscription",
	cmdSetup:     "How to connect on your device",
	cmdHelp:      "Show available commands",
	cmdLang:      "Change language (English / Русский)",
	cmdAwg:       "Get an AmneziaWG config for a location",
}
