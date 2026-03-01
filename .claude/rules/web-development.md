# Web Development Standards

Rules for working on the web app (`apps/web/`). See `docs/web.md` for architecture details.

## TypeScript

- Strict mode enabled — no `any` types
- Prefer `interface` over `type` for component props (better error messages, extensibility)
- Early returns over deep nesting
- Use `@/` path alias for all imports (configured in `tsconfig.json`)

## React Patterns

- Functional components only (no class components)
- Custom hooks for shared logic (in `src/hooks/`)
- Context for cross-cutting state (RepoProvider, ThemeProvider)
- No external state management libraries — React Context + local state is sufficient

## Component Organization

- **shadcn/ui primitives**: `src/components/ui/` — generated via `shadcn` CLI
- **Project components**: `src/components/` — organized by feature
- **Tests**: co-located in `__tests__/` directories alongside source
- **Hooks**: `src/hooks/` with tests in `src/hooks/__tests__/`

## Styling

- Use Tailwind utility classes — avoid inline styles and CSS modules
- Use `cn()` helper from `src/lib/utils.ts` for conditional classes
- Colors use OKLch color space via CSS custom properties in `globals.css`
- Respect `prefers-reduced-motion` for all animations
- Use CSS animations over JS animations where possible
- Theme support: use `dark:` variant for dark mode styles

## State Management

- **RepoProvider**: API data, SSE events, loading/error state — use `useRepo()` hook
- **ThemeProvider**: Light/dark/system theme — use `useTheme()` hook
- **Local state**: UI-only state (selection, collapse, hover) stays in components
- No Redux, Zustand, or other external state libraries

## API Client

- All fetch functions in `src/lib/api.ts` — never call `fetch()` directly from components
- Types must mirror Go contracts in `internal/contracts/http/responses.go`
- SSE subscription via `useSSE()` hook in `src/lib/use-sse.ts`

## Testing

- **Framework**: Vitest + Testing Library
- **Environment**: jsdom
- Test behavior, not implementation — query by role/text, not by class/id
- Co-locate test files in `__tests__/` directories

### Validation Commands

| Change | Command | Time |
|--------|---------|------|
| Single component | `mise run web:test` | ~10s |
| Web + API contract | `mise run check:web` | ~30s |

## Performance

- Avoid unnecessary re-renders — lift state up only when needed
- Use `useMemo`/`useCallback` for expensive computations and stable references
- Prefer CSS animations over JS-driven animations
- Keep the EventFeed capped (100 events max)
