package eventtest

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// FakeBus is a synchronous in-memory event.Bus for tests.
// It supports real Publish/Subscribe with middleware chains,
// replacing the deprecated memory.MemoryBus in test code.
type FakeBus struct {
	mu sync.Mutex

	Published  []event.Event
	PublishErr error

	subscribers    []fakeSub
	publishMW      []event.PublishMiddleware
	handlerMW      []event.Middleware
	publishChain   event.Publisher
	subscribeAllFn func(event.Handler) error
}

type fakeSub struct {
	eventType event.Type
	handler   event.Handler
	all       bool
}

// NewFakeBus creates a synchronous test bus with no subscribers or middleware.
func NewFakeBus() *FakeBus {
	b := &FakeBus{}
	b.rebuildPublishChain()

	return b
}

func (b *FakeBus) rebuildPublishChain() {
	var chain event.Publisher = event.PublisherFunc(b.dispatch)

	for i := len(b.publishMW) - 1; i >= 0; i-- {
		chain = b.publishMW[i](chain)
	}

	b.publishChain = chain
}

// dispatch is the innermost Publisher — appends to Published and calls subscribers.
func (b *FakeBus) dispatch(ctx context.Context, events ...event.Event) error {
	b.mu.Lock()

	if b.PublishErr != nil {
		b.mu.Unlock()
		return b.PublishErr
	}

	b.Published = append(b.Published, events...)
	subs := make([]fakeSub, len(b.subscribers))
	copy(subs, b.subscribers)
	hmw := make([]event.Middleware, len(b.handlerMW))
	copy(hmw, b.handlerMW)
	b.mu.Unlock()

	for _, evt := range events {
		for _, s := range subs {
			if s.all || s.eventType == evt.Type() {
				h := s.handler
				for i := len(hmw) - 1; i >= 0; i-- {
					h = hmw[i](h)
				}

				if err := h(ctx, evt); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (b *FakeBus) Publish(ctx context.Context, events ...event.Event) error {
	return b.publishChain.Publish(ctx, events...)
}

func (b *FakeBus) Subscribe(typ event.Type, handler event.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers = append(b.subscribers, fakeSub{eventType: typ, handler: handler})

	return nil
}

func (b *FakeBus) SubscribeAllFn(fn func(event.Handler) error) *FakeBus {
	b.subscribeAllFn = fn
	return b
}

func (b *FakeBus) SubscribeAll(handler event.Handler) error {
	if b.subscribeAllFn != nil {
		return b.subscribeAllFn(handler)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers = append(b.subscribers, fakeSub{all: true, handler: handler})

	return nil
}

func (b *FakeBus) Use(middleware ...event.Middleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlerMW = append(b.handlerMW, middleware...)
	return nil
}

func (b *FakeBus) UsePublish(middleware ...event.PublishMiddleware) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.publishMW = append(b.publishMW, middleware...)
	b.rebuildPublishChain()

	return nil
}

func (b *FakeBus) Close() error { return nil }

var _ event.Bus = (*FakeBus)(nil)
