package main

import (
	"sync"
)

type Post struct {
	ID     string
	Title  string
	Link   string
	Author string
}

type PostUpdate struct {
	Title *string
	Link  *string
}

type DB struct {
	mu    sync.RWMutex
	posts map[string]Post
}

func NewDB() *DB {
	return &DB{posts: make(map[string]Post, 20)}
}

func (db *DB) GetAll() []Post {
	db.mu.RLock()
	defer db.mu.RUnlock()

	posts := make([]Post, 0, len(db.posts))
	for _, post := range db.posts {
		posts = append(posts, post)
	}

	return posts
}

func (db *DB) Get(id string) (Post, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	post, ok := db.posts[id]
	if ok {
		return post, true
	}

	return Post{}, false
}

func (db *DB) Create(post Post) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, ok := db.posts[post.ID]
	if ok {
		return false
	}

	db.posts[post.ID] = post

	return true
}

func (db *DB) Update(id string, update PostUpdate) bool {
	db.mu.Lock()
	defer db.mu.Unlock()

	post, ok := db.posts[id]
	if !ok {
		return false
	}

	if update.Title != nil {
		post.Title = *update.Title
	}

	if update.Link != nil {
		post.Link = *update.Link
	}

	db.posts[id] = post

	return true
}
