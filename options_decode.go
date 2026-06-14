package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// DecodeJSON decodes a JSON request body into a command using the mapper.
func DecodeJSON[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) {
			cfg.commandDecoder = dec
		})
}

// DecodeJSONQuery decodes a JSON request body into a query.
func DecodeJSONQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) {
			cfg.queryDecoder = dec
		})
}

// DecodeForm decodes form data into a command using the mapper.
func DecodeForm[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) {
			cfg.commandDecoder = dec
		})
}

// DecodeFormQuery decodes form data into a query using the mapper.
func DecodeFormQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) {
			cfg.queryDecoder = dec
		})
}
