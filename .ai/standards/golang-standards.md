# Golang Standards

This is the short version: write idiomatic Go, keep the design simple, and optimize for readability and maintainability.

## Core Rules

- Prefer standard Go patterns over imports from other languages.
- Choose the simplest implementation that meets the requirement.
- Favor composition and package boundaries over deep abstraction.
- Keep code small, cohesive, and easy to review.

## Formatting

- Always run `gofmt`.
- Do not manually align columns, structs, or comments.
- Use normal Go brace placement.
- Do not worry about strict line length; favor readability.

## Packages

- Organize packages by behavior, not by generic buckets like `utils` or `helpers`.
- Keep packages small and single-purpose.
- Avoid circular dependencies.

## Naming

- Use short, lowercase package names.
- Use short variable names in short scopes.
- Name interfaces by behavior, usually with `-er` for single-method interfaces.
- Use `New` for constructors unless the semantics are different.

## Types and APIs

- Accept interfaces, return concrete types.
- Define interfaces at the consumer side, not the producer side.
- Keep interfaces small and only create them when they solve a real boundary.
- Keep structs focused and prefer useful zero values.
- Export only what must be public.

## Functions

- Keep functions small and focused on one job.
- Use guard clauses to reduce nesting.
- Put `context.Context` first in any function that uses it.

## Errors

- Check every returned error.
- Wrap errors with context when passing them upward.
- Return errors for normal failures; reserve `panic` for programmer mistakes or unrecoverable startup failures.

## Concurrency

- Prefer message passing when practical.
- Protect shared state with the right synchronization primitive.
- Make goroutine ownership, cancellation, and exit paths explicit.
- Use `context` for cancellation.

## Logging and Config

- Prefer structured logging.
- Do not both log and return the same error unless that is required.
- Pass configuration explicitly through constructors.
- Avoid hidden global state.

## Testing

- Prefer table-driven tests.
- Test public behavior, not implementation details.
- Avoid mock-first design; use real types unless an interface is justified.

## Comments and Imports

- Write comments that explain why, not what.
- Comment exported symbols.
- Group imports as standard library, external, then internal.
- Use side-effect imports only when necessary and justified.

## AI Agent Rules

### Required

- Use `gofmt`-compliant formatting.
- Check every returned error.
- Use `context.Context` for I/O and external operations.
- Prefer concrete types, composition, cohesive packages, and table-driven tests.
- Wrap errors with context.

### Forbidden

- Ignoring errors.
- Adding interfaces without demonstrated need.
- Inheritance-like structures.
- Global mutable state.
- Panics for expected failures.
- Utility packages without justification.
- Extra dependencies when the standard library is enough.
- Reflection when simpler code exists.

## Decision Order

When options are close, choose in this order:

1. Correctness
2. Simplicity
3. Readability
4. Maintainability
5. Performance
6. Flexibility

## Reference

Primary source: https://go.dev/doc/effective_go

Secondary sources: Go standard library conventions, `gofmt`, and Go package design practices.
