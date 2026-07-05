# Coding Standards

These standards apply across the codebase. They are shaped by Google engineering practices, the TypeScript handbook, and the repository's existing Go and React code.

## Core Principles

- Prefer the smallest change that solves the problem.
- Optimize for code health over cleverness.
- Make code easy to read, review, and modify later.
- Choose explicit behavior over implicit behavior.
- Prefer correctness and clarity before performance tuning.

## Change Quality

- Keep changes focused and easy to review.
- Avoid mixing unrelated refactors with behavior changes.
- If a change is hard to explain briefly, split it up.
- Add tests when behavior changes or when a bug fix would otherwise be unprotected.
- Treat review comments as code health feedback, not personal preference.

## Design

- Prefer simple data flow and explicit dependencies.
- Avoid speculative abstractions and premature generalization.
- Use the narrowest useful interface or function signature.
- Keep modules cohesive and behavior-oriented.
- Remove dead code instead of preserving it "just in case".

## Naming And Readability

- Use names that describe intent, not implementation details.
- Prefer short names only when the scope is short and the meaning is obvious.
- Keep comments focused on why something exists or why it is non-obvious.
- Do not comment obvious code.
- Keep formatting consistent with the language's standard formatter.

## TypeScript Standards

- Enable `strict` mode and keep strictness on unless there is a strong reason not to.
- Prefer type inference when the inferred type is clear.
- Add explicit annotations where they improve API clarity or prevent ambiguity.
- Use `unknown` instead of `any` when the type is not yet known.
- Model object shapes with `type` aliases or `interface` declarations.
- Use unions and narrowing to represent distinct states instead of nullable ad hoc objects.
- Prefer optional properties only when the property is genuinely absent in some cases.
- Keep function inputs and outputs narrowly typed.
- Prefer readonly data structures when mutation is not required.
- Avoid unsafe casts unless the type system cannot express the real constraint.

## JavaScript Standards

- Use modern language features when they improve clarity.
- Avoid hidden globals and implicit mutation.
- Prefer immutable updates for application state.
- Use `const` by default and `let` only when reassignment is required.
- Keep side effects in clearly named functions or event handlers.

## Testing

- Test observable behavior, not internal implementation details.
- Prefer table-driven or data-driven tests where the language makes that natural.
- Add regression tests for bug fixes.
- Keep test names explicit about the behavior under test.
- Do not skip tests that can be made reliable with better setup.

## Code Review Expectations

- Reviews should focus on design, correctness, complexity, tests, naming, and consistency.
- Technical facts override preference.
- Style issues should follow the existing style guide or existing local conventions.
- Prefer useful suggestions over mandatory changes when a point is purely educational.
- Accept the author's choice when multiple reasonable approaches are equivalent.

## References

- Google engineering practices: https://google.github.io/eng-practices/
- TypeScript handbook: https://www.typescriptlang.org/docs/handbook/intro.html
- TypeScript basics and strictness: https://www.typescriptlang.org/docs/handbook/2/basic-types.html
