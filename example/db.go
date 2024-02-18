package main

import (
	"sync"
)

type User struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type DB struct {
	mu    sync.RWMutex
	users map[string]*User
}

func NewDB(users map[string]*User) *DB {
	return &DB{
		users: users,
	}
}

func (db *DB) Get(key string) (User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	user, ok := db.users[key]
	if ok {
		return *user, true
	}

	return User{}, false
}

func (db *DB) Update(key, name string) User {
	db.mu.Lock()
	defer db.mu.Unlock()

	user, ok := db.users[key]
	if !ok {
		user = new(User)
		db.users[key] = user
	}

	user.Key = key
	user.Name = name
	return *user
}
