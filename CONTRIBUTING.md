# Contributing

Lure is a single Go binary with no dependencies outside the standard library.

## Adding a behaviour

Behaviours are rules in `internal/classify`, grouped by what they read: HTTP
requests, login attempts, or the first bytes on the wire. Add the rule with a
tag, a severity and a one-line description, then add a test that shows it
matching the shape it is meant for and staying quiet on a near miss.

## Adding a sensor

A sensor implements `sensor.Sensor`. Keep it low-interaction: read a bounded
amount of input through the shared accept loop, never execute or forward
anything a client sends, record an event, and close the connection.

## Before a change

- `make lint` and `make test` must pass.
- `gofmt` is enforced; run `make fmt`.

## Commits

Conventional commits (`feat:`, `fix:`, `test:`, `docs:`, `build:`, `ci:`), one
logical change each.
