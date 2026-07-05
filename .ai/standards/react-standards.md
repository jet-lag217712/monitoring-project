# React Standards

These standards are based on React's official documentation and the current Vite + React 19 app in this repository.

## Core Principles

- Treat components as pure functions of props and state.
- Keep rendering predictable.
- Keep state local unless there is a clear reason to lift it.
- Compute derived data during render when possible.
- Use effects only to synchronize with external systems.

## Components

- Prefer function components.
- Keep components small and single-purpose.
- Split UI when a component starts mixing unrelated responsibilities.
- Avoid defining components inside other components unless there is a strong reason.
- Use composition instead of deep prop plumbing when it simplifies the tree.

## State

- Store the minimum state needed to represent the UI.
- Derive everything else from props, state, or memoized selectors only when necessary.
- Remember that state is tied to a component's position in the tree.
- Use keys intentionally when you want state to reset.
- Preserve state when the user expects continuity; reset it when a different entity is being shown.

## Hooks

- Call hooks only at the top level of a component or custom hook.
- Never call hooks conditionally or in loops.
- Keep hook dependencies honest and explicit.
- Move reusable logic into custom hooks instead of duplicating effect logic.
- Use `useEffect` for synchronization, not for ordinary data transformations.

## Effects

- Use effects when React must coordinate with something outside React, such as timers, subscriptions, browser APIs, or network listeners.
- Do not use effects to mirror props into state when the same result can be computed during render.
- Clean up subscriptions, timers, and listeners.
- Keep effect logic focused on one external concern.

## Lists And Keys

- Use stable keys that identify the underlying item.
- Do not use array indexes as keys when items can be reordered, inserted, or removed.
- Keep list rendering deterministic.

## Forms And Events

- Use controlled inputs when the UI should always reflect React state.
- Keep event handlers focused on user intent.
- Validate and normalize input before using it in stateful workflows.
- Reset forms explicitly when changing between entities or modes.

## Performance

- Start with correctness and readability.
- Avoid premature memoization.
- Use `useMemo` and `useCallback` only when there is a measured need or a clear referential-stability requirement.
- Prefer `useDeferredValue` or similar tools only when UI responsiveness is actually affected.
- Keep re-renders predictable by keeping state local and props stable.

## Accessibility And UX

- Use semantic HTML first.
- Ensure interactive controls are keyboard accessible.
- Keep labels, focus management, and visual state in sync.
- Make loading, empty, and error states explicit.

## Current Repo Notes

- The frontend currently uses Vite, React 19, and JavaScript.
- The app already uses `StrictMode`, which should remain enabled in development.
- The existing dashboard architecture is state-driven and should stay easy to reason about as it grows.

## References

- React docs: https://react.dev
- Rendering and commit: https://react.dev/learn/render-and-commit
- State and preserving state: https://react.dev/learn/preserving-and-resetting-state
- Avoid unnecessary effects: https://react.dev/learn/you-might-not-need-an-effect
- Rules of React: https://react.dev/reference/rules
