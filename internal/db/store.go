package db

import (
	"fmt"

	"github.com/dgraph-io/badger/v4"
)

type Store struct {
	db *badger.DB
}

func NewStore(path string) (*Store, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable Badger logging

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Set(key, value []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

func (s *Store) Get(key []byte) ([]byte, error) {
	var value []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})
	return value, err
}

func (s *Store) Delete(key []byte) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}

