# API Standards

This document defines standards for designing, implementing, and maintaining APIs within agentic coding systems. It is based on [Microsoft API Guidelines](https://github.com/microsoft/api-guidelines/blob/vNext/Guidelines.md?utm_source=chatgpt.com) and adapted for autonomous agents that perform coding tasks.

---

## General Principles
- **Consistency:** APIs must follow a uniform pattern across all modules to reduce cognitive load for agents and humans alike.
- **Predictability:** Avoid surprises; methods should do what their names imply.
- **Minimalism:** Expose only what is necessary; unnecessary complexity increases failure modes for agentic systems.
- **Extensibility:** Design APIs to accommodate future capabilities without breaking existing integrations.

---

## Naming Conventions
- **Classes:** PascalCase (e.g., `DataProcessor`, `AgentController`)
- **Methods:** VerbNoun (PascalCase or camelCase depending on language) (e.g., `SendMessage`, `fetchData`)
- **Properties:** PascalCase (public) or camelCase (private/internal)
- **Boolean Methods/Properties:** Use `Is` or `Has` prefixes (e.g., `IsConnected`, `HasAccess`)

> Avoid ambiguous verbs like `Do` or `Handle`—agents rely on precise semantics.

---

## Method Design
- Keep method responsibilities **single and clear**.
- **Parameters:** Explicit, required parameters first, optional parameters last.
- **Return Values:** Use clear and consistent data types.
- **Side Effects:** Avoid hidden side effects; agents struggle with non-deterministic behavior.

Example:
```csharp
public async Task<UserProfile> GetUserProfileAsync(string userId)