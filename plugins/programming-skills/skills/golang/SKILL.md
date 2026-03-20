---
name: golang-dev-guidelines
description: Use this skill when planning, researching, writing,
reviewing, refactoring, or testing Go code (including creating unit
tests, test files, and mocks). It provides comprehensive Go
development guidelines including proverbs, SOLID principles, and
testing standards. Apply these guidelines to ensure code quality,
maintainability, and consistency in any Go project.
---

# Go Development Guidelines

Use these guidelines when writing, reviewing, or refactoring Go code
to ensure idiomatic, maintainable, and high-quality code.

## Go Proverbs

These proverbs from Rob Pike capture the philosophy of Go programming:

1. **Don't communicate by sharing memory, share memory by
communicating** - Use channels to pass data between goroutines instead
of shared variables with locks.

2. **Concurrency is not parallelism** - Concurrency is about dealing
with many things at once; parallelism is about doing many things at
once.

3. **Channels orchestrate; mutexes serialize** - Use channels for
coordination and communication; use mutexes only when you need to
protect shared state.

4. **The bigger the interface, the weaker the abstraction** - Prefer
small, focused interfaces. The ideal interface has one method.

5. **Make the zero value useful** - Design types so their zero value
is immediately usable without initialization.

6. **interface{} says nothing** - Avoid empty interface when possible;
it provides no compile-time type safety.

7. **Gofmt's style is no one's favorite, yet gofmt is everyone's
favorite** - Always use gofmt. Consistency matters more than personal
preference.

8. **A little copying is better than a little dependency** - Don't
import a large package for a small function. Copy small amounts of
code when appropriate.

9. **Syscall must always be guarded with build tags** -
Platform-specific code should be properly isolated.

10. **Cgo must always be guarded with build tags** - CGo code should
be isolated and marked appropriately.

11. **Cgo is not Go** - Avoid CGo when possible; it complicates builds
and reduces portability.

12. **With the unsafe package there are no guarantees** - Avoid unsafe
unless absolutely necessary; it bypasses Go's type safety.

13. **Clear is better than clever** - Write straightforward code.
Readability trumps cleverness.

14. **Reflection is never clear** - Avoid reflection when possible; it
makes code harder to understand and maintain.

15. **Errors are values** - Errors are not exceptions. Handle them as
regular values in your control flow.

16. **Don't just check errors, handle them gracefully** - Provide
context when returning errors. Use error wrapping.

17. **Design the architecture, name the components, document the
details** - Good naming and documentation are part of good design.

18. **Documentation is for users** - Write documentation that helps
users understand how to use your code.

19. **Don't panic** - Reserve panic for truly unrecoverable
situations. Return errors instead.

## Effective Go Patterns

### Naming Conventions

- Use **MixedCaps** or **mixedCaps** (not underscores)
- Package names should be lowercase, single-word names
- Getters don't use "Get" prefix: `obj.Owner()` not `obj.GetOwner()`
- Interfaces with one method are named with -er suffix: `Reader`,
`Writer`, `Formatter`
- Acronyms should be all caps: `HTTPServer`, `XMLParser`, `ID`

### Error Handling

```go
// Good: Add context to errors
if err != nil {
return fmt.Errorf("failed to process user %s: %w", userID, err)
}

// Good: Define sentinel errors for comparison
var ErrNotFound = errors.New("not found")

// Good: Custom error types for rich error information
type ValidationError struct {
Field string
Message string
}

func (e *ValidationError) Error() string {
return fmt.Sprintf("validation failed on %s: %s", e.Field, e.Message)
}
```

### Concurrency Patterns

```go
// Good: Use context for cancellation
func process(ctx context.Context, items []Item) error {
for _, item := range items {
select {
case <-ctx.Done():
return ctx.Err()
default:
if err := processItem(item); err != nil {
return err
}
}
}
return nil
}

// Good: Use errgroup for concurrent operations
g, ctx := errgroup.WithContext(ctx)
for _, item := range items {
item := item // capture loop variable
g.Go(func() error {
return processItem(ctx, item)
})
}
if err := g.Wait(); err != nil {
return err
}
```

### Interface Design

```go
// Good: Small, focused interfaces
type Reader interface {
Read(p []byte) (n int, err error)
}

type Writer interface {
Write(p []byte) (n int, err error)
}

// Good: Accept interfaces, return structs
func ProcessData(r io.Reader) (*Result, error) {
// implementation
}
```

### Struct Design

```go
// Good: Make zero value useful
type Buffer struct {
buf []byte
off int
}

// Buffer is usable without explicit initialization
var b Buffer
b.Write([]byte("hello"))
```

## Go Testing Standards

Use the `testify/assert` package for cleaner, more readable assertions.

### Table-Driven Tests with Assert

```go
import (
"testing"
"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
tests := []struct {
name string
a, b int
expected int
}{
{"positive numbers", 2, 3, 5},
{"negative numbers", -1, -2, -3},
{"mixed numbers", -1, 5, 4},
{"zeros", 0, 0, 0},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
result := Add(tt.a, tt.b)
assert.Equal(t, tt.expected, result)
})
}
}
```

### Common Assert Functions

```go
import (
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestAssertions(t *testing.T) {
// Equality
assert.Equal(t, expected, actual)
assert.NotEqual(t, unexpected, actual)

// Nil checks
assert.Nil(t, err)
assert.NotNil(t, result)

// Boolean
assert.True(t, condition)
assert.False(t, condition)

// Error handling
assert.NoError(t, err)
assert.Error(t, err)
assert.ErrorIs(t, err, ErrNotFound)
assert.ErrorContains(t, err, "not found")

// Collections
assert.Len(t, slice, 3)
assert.Empty(t, slice)
assert.Contains(t, slice, element)

// Use require for fatal assertions (stops test on failure)
require.NoError(t, err) // Test stops here if err != nil
require.NotNil(t, result)
}
```

### Test Naming

- Test functions: `TestXxx` where Xxx describes what's being tested
- Subtests: Use descriptive names in `t.Run("name", ...)`
- Benchmarks: `BenchmarkXxx`
- Examples: `ExampleXxx` (appears in documentation)

### Test Organization

```go
// Good: Arrange-Act-Assert pattern with assert
func TestUserService_CreateUser(t *testing.T) {
// Arrange
svc := NewUserService(mockDB)
input := &CreateUserInput{Name: "Alice", Email: "alice@example.com"}

// Act
user, err := svc.CreateUser(context.Background(), input)

// Assert
require.NoError(t, err)
assert.Equal(t, input.Name, user.Name)
assert.Equal(t, input.Email, user.Email)
assert.NotZero(t, user.ID)
}
```

### Assert vs Require

- **assert**: Test continues on failure (use for multiple checks)
- **require**: Test stops immediately on failure (use for critical
preconditions)

```go
func TestWithRequireAndAssert(t *testing.T) {
result, err := DoSomething()

// Use require for errors - no point continuing if this fails
require.NoError(t, err)
require.NotNil(t, result)

// Use assert for checking multiple properties
assert.Equal(t, "expected", result.Name)
assert.True(t, result.IsActive)
assert.Len(t, result.Items, 3)
}
```

## Code Review Checklist

When reviewing Go code, check for:

- [ ] Proper error handling with context
- [ ] No ignored errors (unless explicitly documented why)
- [ ] Consistent naming following Go conventions
- [ ] Small, focused functions and interfaces
- [ ] Proper use of context for cancellation/timeouts
- [ ] No data races in concurrent code
- [ ] Useful zero values for custom types
- [ ] Documentation for exported symbols
- [ ] Table-driven tests for functions with multiple cases
- [ ] No unnecessary dependencies
