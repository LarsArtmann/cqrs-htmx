# [samber/do] do.InvokeAs resolves by scanning the service map — map-order nondeterminism with 2+ satisfying services

Target repo: github.com/samber/do (v2.1.0). Filing authorized by the cqrs-htmx
M01 decision batch (verdict 4); repro verified twice against v2.1.0
(cqrs-htmx round-2 review, 2026-10-30).

## Summary

`do.InvokeAs[T](injector)` resolves `T` by scanning the injector's service
map for the FIRST registered service whose instance satisfies `T`. With two
or more registered services satisfying the same interface, which one answers
depends on Go map iteration order — nondeterministic PER PROCESS RUN, in
single-goroutine code, with zero warning.

## Repro

```go
type Greeter interface{ Greet() string }

type prod struct{}
func (prod) Greet() string { return "prod" }

type override struct{}
func (override) Greet() string { return "override" }

func main() {
	i := do.New()
	do.Provide(i, func(do.Injector) (prod, error) { return prod{}, nil })
	do.Provide(i, func(do.Injector) (override, error) { return override{}, nil })
	_ = do.As[prod, Greeter](i)

	for range 20 {
		i2 := do.New()
		do.Provide(i2, func(do.Injector) (prod, error) { return prod{}, nil })
		do.Provide(i2, func(do.Injector) (override, error) { return override{}, nil })
		_ = do.As[prod, Greeter](i2)
		g, _ := do.InvokeAs[Greeter](i2)
		fmt.Println(g.Greet()) // flips between "prod" and "override"
	}
}
```

Observed: both answers occur across runs. Expected: either a documented
resolution order, or a panic/error on ambiguity ("multiple services satisfy
Greeter: example.prod, example.override").

## Why it matters

An `Override*`/`ProvideNamed` registration whose instance satisfies the same
interface as a `do.As` alias silently becomes a coin-flip alternative to that
alias. In test containers that override one production service, the override
can answer production resolutions — or not — depending on the run.

## Workaround (documented in cqrs-htmx gotcha 37)

Never let two services satisfy one interface when resolving via `InvokeAs`:
alias to a NAME (`do.AsNamed` + `InvokeNamed`) when more than one candidate
can exist. The repo's samber-do-demo resolves via the type-keyed form only
under an enforced single-satisfier rule.

## Ask

Ambiguity detection (deterministic error naming the satisfying services) or
a documented scan order — either closes the silent nondeterminism.
