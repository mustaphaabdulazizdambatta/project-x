package database

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tidwall/buntdb"
)

const UserTable = "user"

type User struct {
	Id           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Token        string `json:"token"`
	CreatedAt    int64  `json:"created_at"`
}

func HashPassword(password string) string {
	h := sha256.Sum256([]byte(password))
	return hex.EncodeToString(h[:])
}

func (d *Database) userInit() {
	d.db.CreateIndex("user_id", UserTable+":*", buntdb.IndexJSON("id"))
}

func (d *Database) userCreate(username, passwordHash, token string) (*User, error) {
	if _, err := d.UserGetByUsername(username); err == nil {
		return nil, fmt.Errorf("username '%s' already exists", username)
	}
	id, _ := d.getNextId(UserTable)
	s := &User{
		Id:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Token:        token,
		CreatedAt:    time.Now().UTC().Unix(),
	}
	jf, _ := json.Marshal(s)
	err := d.db.Update(func(tx *buntdb.Tx) error {
		tx.Set(d.genIndex(UserTable, id), string(jf), nil)
		return nil
	})
	return s, err
}

func (d *Database) userList() ([]*User, error) {
	var list []*User
	err := d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("user_id", func(key, val string) bool {
			s := &User{}
			if err := json.Unmarshal([]byte(val), s); err == nil {
				list = append(list, s)
			}
			return true
		})
		return nil
	})
	return list, err
}

func (d *Database) userGetByUsername(username string) (*User, error) {
	var found *User
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("user_id", func(key, val string) bool {
			s := &User{}
			if err := json.Unmarshal([]byte(val), s); err == nil && s.Username == username {
				found = s
				return false
			}
			return true
		})
		return nil
	})
	if found == nil {
		return nil, fmt.Errorf("user '%s' not found", username)
	}
	return found, nil
}

func (d *Database) userGetByToken(token string) (*User, error) {
	var found *User
	d.db.View(func(tx *buntdb.Tx) error {
		tx.Ascend("user_id", func(key, val string) bool {
			s := &User{}
			if err := json.Unmarshal([]byte(val), s); err == nil && s.Token == token {
				found = s
				return false
			}
			return true
		})
		return nil
	})
	if found == nil {
		return nil, fmt.Errorf("user token not found")
	}
	return found, nil
}

func (d *Database) userDelete(id int) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(d.genIndex(UserTable, id))
		return err
	})
}

func (d *Database) userUpdatePassword(id int, passwordHash string) error {
	return d.db.Update(func(tx *buntdb.Tx) error {
		key := d.genIndex(UserTable, id)
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		var s User
		if err := json.Unmarshal([]byte(val), &s); err != nil {
			return err
		}
		s.PasswordHash = passwordHash
		b, _ := json.Marshal(s)
		_, _, err = tx.Set(key, string(b), nil)
		return err
	})
}

// --- Public API ---

func (d *Database) CreateUser(username, password, token string) (*User, error) {
	return d.userCreate(username, HashPassword(password), token)
}

func (d *Database) ListUsers() ([]*User, error) {
	return d.userList()
}

func (d *Database) UserGetByUsername(username string) (*User, error) {
	return d.userGetByUsername(username)
}

func (d *Database) UserGetByToken(token string) (*User, error) {
	return d.userGetByToken(token)
}

func (d *Database) DeleteUserById(id int) error {
	return d.userDelete(id)
}

func (d *Database) UpdateUserPassword(id int, newPassword string) error {
	return d.userUpdatePassword(id, HashPassword(newPassword))
}

func (d *Database) AuthUser(username, password string) (*User, error) {
	s, err := d.UserGetByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if s.PasswordHash != HashPassword(password) {
		return nil, fmt.Errorf("invalid credentials")
	}
	return s, nil
}
