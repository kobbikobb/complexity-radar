---
name: codebase-design
description: Shared vocabulary for designing deep modules. Use when the user wants to design or improve a module's interface, find deepening opportunities, decide where a seam goes, make code more testable or AI-navigable, or when another skill needs the deep-module vocabulary.
---

# Codebase Design

Design **deep modules**: a lot of behaviour behind a small interface, placed at a clean seam, testable through that interface. Use this language and these principles wherever code is being designed or restructured. The aim is leverage for callers, locality for maintainers, and testability for everyone.

## Glossary

Use these terms exactly — don't substitute "component," "service," "API," or "boundary." Consistent language is the whole point.

**Module** — anything with an interface and an implementation. Deliberately scale-agnostic: a function, class, package, or tier-spanning slice. _Avoid_: unit, component, service.

**Interface** — everything a caller must know to use the module correctly: the type signature, but also invariants, ordering constraints, error modes, required configuration, and performance characteristics. _Avoid_: API, signature (too narrow — they refer only to the type-level surface).

**Implementation** — what's inside a module, its body of code. Distinct from **Adapter**: a thing can be a small adapter with a large implementation (a Postgres repo) or a large adapter with a small implementation (an in-memory fake). Reach for "adapter" when the seam is the topic; "implementation" otherwise.

**Depth** — leverage at the interface: the amount of behaviour a caller (or test) can exercise per unit of interface they have to learn. A module is **deep** when a large amount of behaviour sits behind a small interface, **shallow** when the interface is nearly as complex as the implementation.

**Seam** _(Michael Feathers)_ — a place where you can alter behaviour without editing in that place; the *location* at which a module's interface lives. Where to put the seam is its own design decision, distinct from what goes behind it. _Avoid_: boundary (overloaded with DDD's bounded context).

**Adapter** — a concrete thing that satisfies an interface at a seam. Describes *role* (what slot it fills), not substance (what's inside).

**Leverage** — what callers get from depth: more capability per unit of interface they learn. One implementation pays back across N call sites and M tests.

**Locality** — what maintainers get from depth: change, bugs, knowledge, and verification concentrate in one place rather than spreading across callers. Fix once, fixed everywhere.

## Deep vs shallow

**Deep module** = small interface + lots of implementation:

```
┌─────────────────────┐
│   Small Interface   │  ← Few methods, simple params
├─────────────────────┤
│                     │
│  Deep Implementation│  ← Complex logic hidden
│                     │
└─────────────────────┘
```

**Shallow module** = large interface + little implementation (avoid):

```
┌─────────────────────────────────┐
│       Large Interface           │  ← Many methods, complex params
├─────────────────────────────────┤
│  Thin Implementation            │  ← Just passes through
└─────────────────────────────────┘
```

When designing an interface, ask:

- Can I reduce the number of methods?
- Can I simplify the parameters?
- Can I hide more complexity inside?

## Principles

- **Depth is a property of the interface, not the implementation.** A deep module can be internally composed of small, mockable, swappable parts — they just aren't part of the interface. A module can have **internal seams** (private to its implementation, used by its own tests) as well as the **external seam** at its interface.
- **The deletion test.** Imagine deleting the module. If complexity vanishes, it was a pass-through. If complexity reappears across N callers, it was earning its keep.
- **The interface is the test surface.** Callers and tests cross the same seam. If you want to test *past* the interface, the module is probably the wrong shape.
- **One adapter means a hypothetical seam. Two adapters means a real one.** Don't introduce a seam unless something actually varies across it.

## Designing for testability

Good interfaces make testing natural:

1. **Accept dependencies, don't create them.**

   ```typescript
   // Testable
   function processOrder(order, paymentGateway) {}

   // Hard to test
   function processOrder(order) {
     const gateway = new StripeGateway();
   }
   ```

2. **Return results, don't produce side effects.**

   ```typescript
   // Testable
   function calculateDiscount(cart): Discount {}

   // Hard to test
   function applyDiscount(cart): void {
     cart.total -= discount;
   }
   ```

3. **Small surface area.** Fewer methods = fewer tests needed. Fewer params = simpler test setup.

## Relationships

- A **Module** has exactly one **Interface** (the surface it presents to callers and tests).
- **Depth** is a property of a **Module**, measured against its **Interface**.
- A **Seam** is where a **Module**'s **Interface** lives.
- An **Adapter** sits at a **Seam** and satisfies the **Interface**.
- **Depth** produces **Leverage** for callers and **Locality** for maintainers.

## Examples from this codebase

### Deep module: `Source` interface

```go
// Small interface — callers only need to know three things:
type Source interface {
    Name() string
    Collect(ctx context.Context, repo model.Repository) ([]SourceMetric, error)
    SupportedMetrics() []model.MetricTypeName
}
```

The GitHub implementation hides complex API calls, URL parsing, workflow run fetching, and metric normalization behind this small surface. A test can swap in a mock `APIClient` without touching any caller code — that's **depth-as-leverage**.

### Shallow module: `TerminalFormatter`

```go
type TerminalFormatter struct { UseColor bool }
func (f *TerminalFormatter) Format(report output.Report) string { ... }
```

This module is shallow — the implementation doesn't hide much complexity, and the interface is already minimal (one method). But that's acceptable here: the formatting logic has no hidden complexity worth hiding. Not every module needs to be deep — only those that would benefit from hiding complexity.

### Internal seam vs external seam

The `Source` interface is an **external seam** — it's where different adapters (GitHub, future Jira) plug in. The `APIClient` interface inside the GitHub source is an **internal seam** — it's private to the implementation and used only by its own tests:

```go
// External seam — where adapters plug in
type Source interface { ... }

// Internal seam — hidden inside GitHub source, used for testing
type APIClient interface {
    Get(ctx context.Context, endpoint string) (json.RawMessage, error)
    GetWithParams(ctx context.Context, endpoint string, params map[string]string) (json.RawMessage, error)
    GetPaginated(ctx context.Context, endpoint string, params map[string]string, maxPages int) (json.RawMessage, error)
    GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error)
}
```

Two adapters exist at the external seam (`Source`): the real GitHub implementation and any test mock. One adapter exists at the internal seam (`APIClient`): the real `gh` client and test fakes. Both seams earn their keep because there are two adapters at each.

### Deletion test applied

If you delete the `collector` module, complexity reappears across CLI commands — project creation, repository iteration, metric storage, scoring orchestration. The collector is earning its keep.

If you delete the `normalize` package inside `scorer`, the normalization logic would have to be duplicated across every dimension calculation. That's concentration — also earning its keep.

If you delete the `formatMetricName` function, callers would need to implement their own string formatting. But the function is a thin utility, not a deep module — it doesn't earn its keep by hiding complexity.

## Rejected framings

- **Depth as ratio of implementation-lines to interface-lines** (Ousterhout): rewards padding the implementation. We use depth-as-leverage instead.
- **"Interface" as the TypeScript `interface` keyword or a class's public methods**: too narrow — interface here includes every fact a caller must know.
- **"Boundary"**: overloaded with DDD's bounded context. Say **seam** or **interface**.

## Going deeper

- **Deepening a cluster given its dependencies** — dependency categories, seam discipline, and replace-don't-layer testing.
- **Exploring alternative interfaces** — spin up parallel sub-agents to design the interface several radically different ways, then compare on depth, locality, and seam placement.
