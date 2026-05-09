package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type fakeDB struct {
	hasSub    bool
	hasSubErr error
}

func (f *fakeDB) HasUserSubscription(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.hasSub, f.hasSubErr
}

type fakePolar struct {
	url   string
	err   error
	calls int
}

func (f *fakePolar) CreateCheckoutSession(_ context.Context, _, _ string) (string, error) {
	f.calls++
	return f.url, f.err
}

var errInternal = errors.New("internal db error")
