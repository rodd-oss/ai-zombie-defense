# Agent Guide

## Agent Rules

- **NEVER ignore linting rules**: Always address lint warnings and errors. Use `bun task fix` to automatically fix issues where possible, and manually resolve any remaining violations before committing code.

## Build/Lint/Test Commands

The repository uses a custom task runner (`bun task`) defined in `package.json`. All commands should be run through `bun task <task>` unless a specific one-off command is needed.

### Task Runner Usage

```bash
bun task <taskname>          # Run a specific task from package.json
bun task --help              # List all available tasks
bun task check               # Run all checks (lint, format, typecheck) concurrently
bun task fix                 # Apply automatic fixes for lint and formatting
bun task test                # Run tests for all languages concurrently
bun task lint                # Run linters for all languages concurrently
```

### Language-Specific Commands

#### TypeScript/JavaScript (Bun)

**Single test execution**: Use `bun test --test-name-pattern "pattern"` to run specific tests. For example: `bun test --test-name-pattern "placeholder"`.

#### Rust

**Single test execution**: Use `cargo test --package <crate_name> --test <test_file>` for integration tests, or `cargo test test_function_name` for unit tests. Example: `cargo test placeholder`.

#### Python

**Single test execution**: Use `pytest path/to/test.py::test_function` when tests are added.

#### Go

**Single test execution**: Use `go test -run TestName ./path/to/package`. Example: `go test -run TestAdd ./packages/go/math`.

### Pre-commit Hooks

The repository uses Husky with lint-staged to run checks on staged files before commits. The pre-commit hook runs:

`bun task check` - Runs all language checks concurrently

## Code Style Guidelines

### TypeScript/JavaScript

- **TypeScript Configuration**: Strict mode enabled with `noUncheckedIndexedAccess` and `noImplicitOverride`. `noUnusedLocals` and `noUnusedParameters` are disabled to allow flexible development.
- **Imports**: Use ES modules (`import/export`). Prefer named imports over default imports when appropriate. Use workspace dependencies for internal packages.
- **Naming Conventions**:
  - Variables and functions: `camelCase`
  - Classes, types, interfaces, enums: `PascalCase`
  - Constants: `UPPER_SNAKE_CASE` for true constants, `camelCase` for `const` variables
  - Private members: prefix with underscore `_privateMethod` (optional)
- **Error Handling**: Use `try/catch` with typed errors where possible. Avoid swallowing errors; log or rethrow appropriately.
- **Async/Await**: Prefer `async/await` over raw promises for readability.
- **Formatting**: Prettier with default configuration (empty `.prettierrc`). Line length: 80 characters (default).
- **Type Annotations**: Provide explicit return types for functions and public methods. Use type inference for local variables when obvious.
- **Comments**: Use JSDoc for public APIs. Prefer self-documenting code over excessive comments.

### Rust

- **Edition**: 2021 edition (configured in workspace).
- **Formatting**: Use `rustfmt` with default settings. Run `cargo fmt --all` before committing.
- **Linting**: Follow all `clippy` warnings. The configuration uses `-D warnings` to treat warnings as errors.
- **Naming**: Follow Rust conventions (`snake_case` for variables/functions, `PascalCase` for types, `SCREAMING_SNAKE_CASE` for constants).
- **Error Handling**: Prefer `Result<T, E>` and `Option<T>` over panics. Use `?` operator for error propagation.
- **Imports**: Group imports (std, external, internal) with blank lines. Use `crate::` prefix for internal modules.
- **Documentation**: Use `///` doc comments for public items. Include examples where helpful.

### Python

- **Tooling**: Ruff for linting and formatting (replaces flake8, black, isort).
- **Python Version**: Requires Python ≥3.12 (configured in `pyproject.toml`).
- **Formatting**: Follow Ruff's default formatting rules (similar to black).
- **Linting**: Ruff with default rule set. Treat warnings as errors.
- **Naming**: Follow PEP 8: `snake_case` for functions/variables, `PascalCase` for classes, `UPPER_SNAKE_CASE` for constants.
- **Type Hints**: Use type annotations for function parameters and return values (enabled via `pyproject.toml`).
- **Imports**: Group imports (standard library, third-party, local) with blank lines. Use absolute imports.
- **Error Handling**: Use explicit exception handling. Avoid bare `except:` clauses.

### Go

- **Tooling**: Use `go fmt` for formatting and `go vet` for linting.
- **Formatting**: Standard `gofmt` style.
- **Naming**: Follow Go conventions: `CamelCase` for exported identifiers, `mixedCaps` or `camelCase` for unexported.
- **Error Handling**: Return errors as the last return value. Use `if err != nil` pattern.
- **Imports**: Group standard library, third-party, and internal imports with blank lines.
- **Testing**: Use `testing` package. Table-driven tests preferred for multiple test cases.

## Tooling Configuration

### Rust

- Workspace configuration in root `Cargo.toml`
- Members: `packages/rust/*`
- Edition: 2021
- Release profile: LTO enabled, codegen-units=1, opt-level=3, strip=true

### Python

- Uses UV workspace configuration in `pyproject.toml`
- Members: `apps/py-*`, `packages/python/*`
- Requires Python ≥3.12

### Go

- Workspace configuration in `go.work` (Go 1.24.1)

## Task Runner Implementation

The custom task runner (`scripts/task.ts`) provides:

- Concurrent and sequential task execution
- Circular dependency detection
- Colored output with task tagging
- Help system listing available tasks

Tasks are defined in `package.json` under the `"tasks"` key. Reference other tasks with `--concurrent task1 task2` for concurrent execution or space-separated for sequential.

## Troubleshooting

- **Pre-commit hooks failing**: Run `bun task check` to identify which language check is failing, then run the specific fix command (e.g., `bun task ts:fix`).
- **TypeScript errors**: Ensure `tsconfig.json` settings are respected. The configuration disables `noUnusedLocals` and `noUnusedParameters` for flexibility.
- **Rust compilation errors**: Check that all crates are members of the workspace in `Cargo.toml`.
- **Python linting errors**: Ensure UV is installed (`curl -LsSf https://astral.sh/uv/install.sh | sh`)
- **Task runner not found**: Ensure Bun is installed (`curl -fsSL https://bun.sh/install | bash`).

## Additional Notes

- Always run `bun task check` before committing to ensure code quality.
- When adding new packages, update the appropriate workspace configuration (Cargo.toml, pyproject.toml, go.work, package.json).
- **Ralph autonomous agent**: The `scripts/ralph/ralph.sh` script supports `amp`, `claude`, and `opencode` tools. Use `--tool opencode` to run with OpenCode. Ensure OpenCode is configured with appropriate permissions for autonomous operation.


# Ultracite Code Standards

This project uses **Ultracite**, a zero-config preset that enforces strict code quality standards through automated formatting and linting.

## Quick Reference

- **Format code**: `bun x ultracite fix`
- **Check for issues**: `bun x ultracite check`
- **Diagnose setup**: `bun x ultracite doctor`

Biome (the underlying engine) provides robust linting and formatting. Most issues are automatically fixable.

---

## Core Principles

Write code that is **accessible, performant, type-safe, and maintainable**. Focus on clarity and explicit intent over brevity.

### Type Safety & Explicitness

- Use explicit types for function parameters and return values when they enhance clarity
- Prefer `unknown` over `any` when the type is genuinely unknown
- Use const assertions (`as const`) for immutable values and literal types
- Leverage TypeScript's type narrowing instead of type assertions
- Use meaningful variable names instead of magic numbers - extract constants with descriptive names

### Modern JavaScript/TypeScript

- Use arrow functions for callbacks and short functions
- Prefer `for...of` loops over `.forEach()` and indexed `for` loops
- Use optional chaining (`?.`) and nullish coalescing (`??`) for safer property access
- Prefer template literals over string concatenation
- Use destructuring for object and array assignments
- Use `const` by default, `let` only when reassignment is needed, never `var`

### Async & Promises

- Always `await` promises in async functions - don't forget to use the return value
- Use `async/await` syntax instead of promise chains for better readability
- Handle errors appropriately in async code with try-catch blocks
- Don't use async functions as Promise executors

### React & JSX

- Use function components over class components
- Call hooks at the top level only, never conditionally
- Specify all dependencies in hook dependency arrays correctly
- Use the `key` prop for elements in iterables (prefer unique IDs over array indices)
- Nest children between opening and closing tags instead of passing as props
- Don't define components inside other components
- Use semantic HTML and ARIA attributes for accessibility:
  - Provide meaningful alt text for images
  - Use proper heading hierarchy
  - Add labels for form inputs
  - Include keyboard event handlers alongside mouse events
  - Use semantic elements (`<button>`, `<nav>`, etc.) instead of divs with roles

### Error Handling & Debugging

- Remove `console.log`, `debugger`, and `alert` statements from production code
- Throw `Error` objects with descriptive messages, not strings or other values
- Use `try-catch` blocks meaningfully - don't catch errors just to rethrow them
- Prefer early returns over nested conditionals for error cases

### Code Organization

- Keep functions focused and under reasonable cognitive complexity limits
- Extract complex conditions into well-named boolean variables
- Use early returns to reduce nesting
- Prefer simple conditionals over nested ternary operators
- Group related code together and separate concerns

### Security

- Add `rel="noopener"` when using `target="_blank"` on links
- Avoid `dangerouslySetInnerHTML` unless absolutely necessary
- Don't use `eval()` or assign directly to `document.cookie`
- Validate and sanitize user input

### Performance

- Avoid spread syntax in accumulators within loops
- Use top-level regex literals instead of creating them in loops
- Prefer specific imports over namespace imports
- Avoid barrel files (index files that re-export everything)
- Use proper image components (e.g., Next.js `<Image>`) over `<img>` tags

### Framework-Specific Guidance

**Next.js:**
- Use Next.js `<Image>` component for images
- Use `next/head` or App Router metadata API for head elements
- Use Server Components for async data fetching instead of async Client Components

**React 19+:**
- Use ref as a prop instead of `React.forwardRef`

**Solid/Svelte/Vue/Qwik:**
- Use `class` and `for` attributes (not `className` or `htmlFor`)

---

## Testing

- Write assertions inside `it()` or `test()` blocks
- Avoid done callbacks in async tests - use async/await instead
- Don't use `.only` or `.skip` in committed code
- Keep test suites reasonably flat - avoid excessive `describe` nesting

## When Biome Can't Help

Biome's linter will catch most issues automatically. Focus your attention on:

1. **Business logic correctness** - Biome can't validate your algorithms
2. **Meaningful naming** - Use descriptive names for functions, variables, and types
3. **Architecture decisions** - Component structure, data flow, and API design
4. **Edge cases** - Handle boundary conditions and error states
5. **User experience** - Accessibility, performance, and usability considerations
6. **Documentation** - Add comments for complex logic, but prefer self-documenting code

---

Most formatting and common issues are automatically fixed by Biome. Run `bun x ultracite fix` before committing to ensure compliance.
