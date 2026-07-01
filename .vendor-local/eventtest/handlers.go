package eventtest

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

func AppendEventsHandler(events *[]event.Event) event.Handler {
	return func(_ context.Context, evt event.Event) error {
		*events = append(*events, evt)

		return nil
	}
}

func NoopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

func NoopEventPublisher() event.Publisher {
	return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return nil
	})
}

func FailingEventPublisher(msg string) event.Publisher {
	return event.PublisherFunc(func(_ context.Context, _ ...event.Event) error {
		return event.NewRejection("eventtest.failing_publisher", msg)
	})
}

func FailingEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return event.NewRejection("eventtest.failing_handler", msg)
	}
}

func PanicEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		panic(msg)
	}
}

func CallbackEventHandler(called *bool) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		*called = true

		return nil
	}
}

func EventMiddleware(callOrder *[]string, name string) func(h event.Handler) event.Handler {
	return func(h event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, evt)
		}
	}
}
