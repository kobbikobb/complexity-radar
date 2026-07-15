package main

import (
	"fmt"

	"github.com/kobbikobb/complexity-radar/internal/store"
)

func openStore(dbPath string) (*store.Store, error) {
	s, err := store.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return s, nil
}
