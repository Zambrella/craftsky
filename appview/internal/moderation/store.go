package moderation

import "social.craftsky/appview/internal/dbx"

type Store struct {
	pool dbx.DB
}

func NewStore(pool dbx.DB) *Store {
	return &Store{pool: dbx.Normalize(pool)}
}
