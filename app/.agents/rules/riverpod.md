---
paths:
  - "**/*.dart"
---

# Riverpod Guidelines

Riverpod 3.x with code generation (`riverpod_annotation`/`riverpod_generator`). All providers use the `@riverpod` annotation pattern.

## Code Generation

After modifying provider files, run:
```bash
dart run build_runner build --delete-conflicting-outputs
```

Every provider file needs:
```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';
part 'my_provider.g.dart';
```

## Provider Patterns

### Read Family Providers Inline

When reading a parameterized/family provider, pass the provider call directly to `ref.read()` or `container.read()`. Do not store the provider instance in a local variable just to read it.

```dart
// Wrong - unnecessary provider instance local
final itemProvider = itemDetailProvider(id, locale);
final item = ref.read(itemProvider);

// Right - read the family provider inline
final item = ref.read(itemDetailProvider(id, locale));
```

This applies to notifier reads too:

```dart
// Wrong
final itemProvider = itemDetailProvider(id, locale);
ref.read(itemProvider.notifier).refresh();

// Right
ref.read(itemDetailProvider(id, locale).notifier).refresh();
```

### Simple State (Search, View Mode, Filters)

```dart
@riverpod
class SearchQuery extends _$SearchQuery {
  @override
  String build() => '';

  void setQuery(String query) => state = query;
  void clear() => state = '';
}
```

### Data Fetching (Read-Only)

```dart
@riverpod
Future<List<Item>> allItems(Ref ref) async {
  final api = ref.watch(apiClientProvider);  // Always watch in build/function body
  return await api.getItems();
}
```

### Parameterized Fetching

```dart
@riverpod
Future<Item> itemDetail(Ref ref, int itemId) async {
  final api = ref.watch(apiClientProvider);
  return await api.getItem(itemId);
}
// Usage: ref.watch(itemDetailProvider(itemId))
```

### Mutations (Create/Update/Delete)

**Use `FutureOr<T>` for providers that don't load initially:**

When a provider starts in an idle state (not loading), use `FutureOr<T> build() => null` instead of `Future<T> build() async {}`. An async build method causes Riverpod to transition from loading -> data on initialization, which can trigger `ref.listen` callbacks unexpectedly.

```dart
// Wrong - async causes loading -> data transition on init
@override
Future<void> build() async {}

// Right - synchronous return, no initial loading state
@override
FutureOr<void> build() => null;
```

**Prefer `AsyncValue.guard()` for cleaner error handling:**

```dart
@riverpod
class CreateItem extends _$CreateItem {
  @override
  FutureOr<Item?> build() => null;

  Future<void> create(String name, {String? description}) async {
    final api = ref.read(apiClientProvider);  // read() in methods, not watch()

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final newItem = await api.createItem(
        Item(name: name, description: description),
      );

      if (!ref.mounted) return null;  // guard after await

      // Explicit invalidation list - invalidate ALL related providers
      for (final provider in <ProviderOrFamily>[
        allItemsProvider,
        itemDetailProvider(newItem.id!),
      ]) {
        ref.invalidate(provider);
      }

      return newItem;
    });
  }

  void reset() => state = const AsyncData(null);
}
```

### Derived/Filtered Data (Synchronous)

```dart
@riverpod
Map<String, List<Item>> itemsByCategory(Ref ref) {
  final itemsAsync = ref.watch(allItemsProvider);

  return switch (itemsAsync) {
    AsyncData(:final value) => _groupItems(value),
    _ => {},  // Return empty on loading/error
  };
}
```

## Consuming AsyncValue in Widgets

**ALWAYS use switch statements with pattern matching - never use `.when()` method.**

### Preferred Pattern (Data First)

Put the data case first so data remains visible during refresh.

```dart
final itemAsync = ref.watch(itemDetailProvider(itemId));

return switch (itemAsync) {
  AsyncValue(:final value?) => ItemDetailContent(item: value),
  AsyncValue(:final error?) => Center(child: Text('Error: $error')),
  _ => const Center(child: CircularProgressIndicator()),
};
```

### Guard Clauses for Conditional Data States

Use `when` guard clauses instead of ternary operators inside case bodies:

```dart
// Wrong - ternary inside case body
AsyncData(:final value) => value.isEmpty ? const EmptyState() : ItemList(items: value),

// Right - guard clauses for each condition
AsyncData(:final value) when value.isEmpty => const EmptyState(),
AsyncData(:final value) => ItemList(items: value),
AsyncError(:final error) => ErrorState(error: error),
_ => const CircularProgressIndicator(),
```

The `?` postfix on a pattern (e.g., `final value?`) is a null-check pattern that matches only non-null and binds with the non-nullable type.

### Full-Screen Loading Pattern

Use when you want loading/error states without preserving previous data:

```dart
return switch (itemAsync) {
  AsyncLoading() => const Center(child: CircularProgressIndicator()),
  AsyncError(:final error) => Center(child: Text('Error: $error')),
  AsyncData(:final value) => ItemDetailContent(item: value),
};
```

## Listening for Side Effects

**Use `ref.listen` for navigation, snackbars, and other side effects - NOT try/catch around provider calls.**

### Transition-Based Pattern (Preferred)

Match on `(prev, state)` tuple to detect specific state transitions:

```dart
ref.listen(createItemProvider, (prev, state) {
  switch ((prev, state)) {
    case (AsyncLoading(), AsyncData(value: final item?)):
      ItemDetailRoute(itemId: item.id.toString()).go(context);
      ref.read(createItemProvider.notifier).reset();
    case (AsyncLoading(), AsyncError(:final error)):
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Error: $error')),
      );
    case _:
      break;
  }
});
```

### Simple Pattern

```dart
ref.listen(deleteItemProvider, (_, state) {
  switch (state) {
    case AsyncData(): context.pop();
    case AsyncError(:final error): showSnackbar(error);
    case _: break;
  }
});
```

## Calling Async Provider Methods from initState

When calling methods on an `AsyncNotifier` from `initState()`, **always wrap in `addPostFrameCallback`**:

```dart
// Wrong - state.value may be null
@override
void initState() {
  super.initState();
  ref.read(myProvider.notifier).runTask();
}

// Right - defers until provider is initialized
@override
void initState() {
  super.initState();
  WidgetsBinding.instance.addPostFrameCallback((_) {
    ref.read(myProvider.notifier).runTask();
  });
}
```

## Lifecycle & Safety

### Check `ref.mounted` After Async Gaps

After any `await`, the provider may have been disposed. Always check `ref.mounted` before using `ref` or `state`:

```dart
Future<void> doWork() async {
  final result = await someAsyncCall();

  // Wrong - provider may be disposed after await
  state = AsyncData(result);

  // Right - guard against disposed ref
  if (!ref.mounted) return;
  state = AsyncData(result);
}
```

### No Constructor Logic in Notifiers

Constructor logic runs before Riverpod initializes the notifier. All initialization must go in `build()`:

```dart
// Wrong - runs before Riverpod is ready
@riverpod
class MyNotifier extends _$MyNotifier {
  MyNotifier() {
    _init(); // ref and state are not available here
  }
}

// Right - use build() for initialization
@riverpod
class MyNotifier extends _$MyNotifier {
  @override
  FutureOr<MyState> build() {
    _init();
    return MyState.initial();
  }
}
```

### No Public Properties on Notifiers

Public properties bypass Riverpod's state management and notification system. All state must flow through the `state` property:

```dart
// Wrong - mutations to this field won't notify listeners
@riverpod
class MyNotifier extends _$MyNotifier {
  List<Item> cachedItems = [];  // public field, invisible to Riverpod
}

// Right - all state through the state property
@riverpod
class MyNotifier extends _$MyNotifier {
  @override
  List<Item> build() => [];

  void addItem(Item item) => state = [...state, item];
}
```

### Don't Store Notifier References in Variables

Stored references go stale after async gaps or provider disposal. Always fetch fresh:

```dart
// Wrong - notifier reference may be stale after await
final notifier = ref.read(myProvider.notifier);
await someOperation();
notifier.update(newValue);  // may reference a disposed notifier

// Right - fetch fresh reference each time
await someOperation();
if (!ref.mounted) return;
ref.read(myProvider.notifier).update(newValue);
```

### Don't Use `ref` Inside `dispose()`

By the time `dispose()` runs on a `ConsumerState`, the provider's ref is already invalid:

```dart
// Wrong - ref is invalid during dispose
@override
void dispose() {
  ref.read(myProvider.notifier).cleanup();  // will throw
  super.dispose();
}

// Right - use ref.onDispose() inside the provider itself
@override
FutureOr<MyState> build() {
  ref.onDispose(() => _controller.dispose());
  return MyState.initial();
}
```

### Clean Up Disposable Objects with `ref.onDispose()`

If a provider creates disposable resources, register cleanup to prevent memory leaks:

```dart
@riverpod
Stream<Event> eventStream(Ref ref) {
  final controller = StreamController<Event>();

  ref.onDispose(() => controller.close());  // always register cleanup

  return controller.stream;
}
```

## Testing

Reference: [Testing your providers](https://riverpod.dev/docs/how_to/testing).

### Container setup

**Use `ProviderContainer.test()` — it auto-disposes at test end.** Create one fresh container per `test(...)` body; never share across tests.

```dart
test('foo', () {
  final container = ProviderContainer.test(
    overrides: [
      apiClientProvider.overrideWithValue(fakeApi),
    ],
  );
  // no explicit dispose — .test() registers addTearDown for you.
});
```

Plain `ProviderContainer()` still works but forces manual `addTearDown(container.dispose)` — don't bother in tests.

### Overriding providers

| Shape | When |
|---|---|
| `provider.overrideWith((ref) => ...)` | Default for sync/function providers. |
| `provider.overrideWithValue(AsyncValue.data(x))` | `FutureProvider` / `StreamProvider` only (restored in 3.0). |
| `notifierProvider.overrideWith(MyNotifierMock.new)` | Replace an entire notifier implementation. Subclass the real notifier; pass the tear-off. |
| `notifierProvider.overrideWithBuild((ref) => seed)` | Mock only `build`; preserve the notifier's methods. Prefer this over a full subclass when you just need to seed state. |

**Family providers:** call the family, then override the returned instance — `onboardingStatusProvider('did:plc:a').overrideWith(...)`.

### Mocking notifiers

**Generally don't.** The docs say plainly: *"It is generally discouraged to mock Notifiers."* Instead, mock the repository / service / API client the notifier depends on, and override *that* provider.

If you must replace a notifier:
- **Subclass, don't `implements`.** Subclasses inherit the full surface; `implements` leaves you manually re-stubbing every method.
- Use `overrideWith(FakeNotifier.new)` — the class tear-off, not an instance.
- For the fake's `build()`, match the real signature and the `FutureOr<T> build() => null` idle-provider rule.

### Awaiting async providers

```dart
// For @riverpod async providers:
final value = await container.read(myFutureProvider.future);

// Or assert:
await expectLater(container.read(myFutureProvider.future), completion(42));
```

For "fire-and-forget" background work started inside a provider's `build` (e.g. an `unawaited(...)` validation), pump the event loop — don't guess at microtask counts:

```dart
// Flush the event loop until all chained awaits settle.
for (var i = 0; i < 5; i++) {
  await Future<void>.delayed(Duration.zero);
}
```

### Widget tests

Wrap in `ProviderScope` with overrides; grab the container via `tester.container()` when you need direct access:

```dart
await tester.pumpWidget(
  ProviderScope(
    overrides: [authControllerProvider.overrideWith(FakeAuthController.new)],
    child: const MyWidget(),
  ),
);
final container = tester.container();
```

### Key testing rules

| Do | Don't |
|---|---|
| `ProviderContainer.test()` | `ProviderContainer()` + manual `addTearDown` |
| One container per `test(...)` | Share containers across tests |
| `overrideWith(Notifier.new)` (tear-off) | `overrideWith(() => Notifier())` (allocates before Riverpod init) |
| `overrideWithBuild` to seed state | Full notifier subclass for a one-liner |
| Mock the repository/service | Mock the notifier directly |
| `subclass extends RealNotifier` | `class Fake implements RealNotifier` |
| `await container.read(p.future)` | `.then(...)` / synchronous reads on async providers |

## Key Rules

| Do | Don't |
|---|---|
| `ref.watch()` in build methods | `ref.read()` in build methods |
| `ref.read()` in callbacks/methods | `ref.watch()` in callbacks |
| `ref.read(provider(x, y))` | `final p = provider(x, y); ref.read(p)` |
| Check `ref.mounted` after every `await` | Use `ref`/`state` after async gaps unchecked |
| Initialize notifiers in `build()` | Put logic in notifier constructors |
| Keep notifier properties private | Expose public fields on notifiers |
| Fetch notifier refs fresh via `ref.read()` | Store notifier references in variables |
| Register `ref.onDispose()` for resources | Rely on GC for disposable objects |
| Clean up resources in provider `ref.onDispose()` | Use `ref` inside widget `dispose()` |
| Switch pattern matching for AsyncValue | `.when()` method |
| `ref.listen` for side effects | try/catch around provider calls |
| Explicit invalidation lists | `ref.invalidateSelf()` alone |
| `addPostFrameCallback` for async provider methods in `initState()` | Direct calls in `initState()` |

## File Organization

```
lib/{feature}/providers/
  {feature}_providers.dart      # Data fetching + simple state
  save_{feature}_provider.dart  # Create/Update mutations
  delete_{feature}_provider.dart # Delete mutations
```

## Naming Conventions

| Pattern | Example |
|---|---|
| Data fetching | `allItemsProvider`, `itemDetailProvider` |
| Search/filter state | `itemSearchQueryProvider` |
| View mode | `listViewModeProvider` |
| Create mutation | `CreateItem` class |
| Update mutation | `UpdateItem` class |
| Delete mutation | `DeleteItem` class |
