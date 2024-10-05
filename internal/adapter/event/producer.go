package event

import (
	"context"

	"github.com/go-order/internal/core"
)

type EventNotifier interface {
	Producer(ctx context.Context, event core.Event) error
}