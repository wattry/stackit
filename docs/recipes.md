# Common Change Recipes

Step-by-step file lists for cross-cutting changes that touch multiple layers.

## Add a New GitHub Client Method

Adding a method to the GitHub client interface requires updating 5 files:

| # | File | What to do |
|---|------|------------|
| 1 | `internal/github/client.go` | Add method to `Client` interface |
| 2 | `internal/github/client_real.go` | Implement on `StackitGitHubClient` |
| 3 | `testhelpers/github_mock_client.go` | Implement on `MockGitHubClient` (synthetic data) |
| 4 | `internal/demo/demo_github_client.go` | Implement on `GitHubClient` (fake data + `simulateDelay`) |
| 5 | `internal/app/context_test.go` | Add stub to `fakeGitHubClient` |

For GraphQL batch methods, follow the pattern in `status.go`:
- Build query with aliases via `strings.Builder`
- Execute via `executeGraphQLQuery()` (defined in `pr_operations.go`)
- Parse JSON response into typed results

## Add a New API Response Field (Backend to Frontend)

When adding a field that flows from Go through the API to the web app:

| # | File | What to do |
|---|------|------------|
| 1 | `internal/contracts/http/responses.go` | Add field to response struct |
| 2 | `internal/contracts/http/mappers.go` | Populate field in mapper function |
| 3 | `internal/api/handlers/view_assembler.go` | Fetch/compute data and pass to mapper |
| 4 | `internal/contracts/http/mappers_test.go` | Update existing test calls, add new test cases |
| 5 | `api/openapi/stackit.yaml` | Add field to OpenAPI schema |
| 6 | `apps/web/src/lib/api.ts` | Add field to TypeScript interface |
| 7 | `apps/web/src/components/...` | Use field in component |

When changing a mapper function signature, grep for all callers — typically `view_assembler.go` and `mappers_test.go`.

## Add a New CLI Command

| # | File | What to do |
|---|------|------------|
| 1 | `internal/cli/stack/<name>.go` | Cobra command definition (`Long` should include examples) |
| 2 | `internal/actions/<name>/` | Business logic package |
| 3 | `internal/cli/stack/root.go` | Register command in parent |
| 4 | Tests in respective packages | |

Follow patterns in existing commands (e.g., `internal/cli/stack/describe.go`).

## Frontend Testing Notes

- Component tests use **vitest + @testing-library/react**
- Tests exist for pure/self-contained components (e.g., `status-badge.test.tsx`)
- Components requiring `RepoProvider` (e.g., `RecentlyMerged`) don't have test wrappers yet — test pure logic helpers instead
- Run `mise run web:test` for web tests, `mise run check:web` for full web validation (tests + typecheck + build)
