---
paths:
  - "**/*.dart"
---

# Flutter Development Guidelines

## Tooling

- Prefer **Dart MCP tools** over shell commands for analyzing code, formatting, running tests, pub commands, and launching apps.

## Widget Architecture

- **Always create new widget classes** instead of extracting build logic into helper methods within a widget.
- Prefer **small, composable widgets** that do one thing well.

```dart
// BAD: Helper method inside a widget
class MyScreen extends StatelessWidget {
  Widget _buildHeader() { ... }
  Widget _buildBody() { ... }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [_buildHeader(), _buildBody()],
    );
  }
}

// GOOD: Separate widget classes
class MyScreen extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return const Column(
      children: [MyScreenHeader(), MyScreenBody()],
    );
  }
}
```

## Theming & Styling

- Use `Theme.of(context)` for colors, text styles, and spacing. Never hardcode values that the theme provides.
- Use **theme extensions** when available (e.g., `appSpacing`, `semanticColors`) before falling back to base theme properties.
- **Avoid using `Opacity` or `.withOpacity()`** to create muted/less saturated colors. Instead, define explicit color variants in the theme or use `Color.alphaBlend` to blend with the background.

## Data Modeling

- Prefer **immutable data structures** using code generation packages (`freezed`, `dart_mappable`) over hand-written classes with mutable fields.

## Logging

- Use a **logging package** (e.g., `logger`, `logging`) instead of `print` or `debugPrint` for all diagnostic output.

## Modern Dart

- Use modern Dart syntax and language features where applicable:
  - **Switch expressions** with guard clauses (`when`)
  - **Sealed classes** for exhaustive pattern matching
  - **Records** and **destructuring** for lightweight data grouping
  - **If-case** and **switch-case** pattern matching
  - **Class modifiers** (`final`, `interface`, `base`, `mixin`) where appropriate

## Performance

- Use **`const` constructors** wherever possible to enable widget caching.
