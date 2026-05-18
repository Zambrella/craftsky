import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:flutter/widgets.dart';

/// Provides an [AppMessenger] to the widget subtree. Mirrors how Flutter's
/// own `Theme`, `MediaQuery`, and `ScaffoldMessenger` are provided.
///
/// Tests override the messenger by wrapping the widget under test in a
/// `MessengerScope` whose [messenger] is a recording fake.
class MessengerScope extends InheritedWidget {
  const MessengerScope({
    required this.messenger,
    required super.child,
    super.key,
  });

  final AppMessenger messenger;

  static AppMessenger of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<MessengerScope>();
    assert(
      scope != null,
      'MessengerScope.of() called with no MessengerScope ancestor.',
    );
    return scope!.messenger;
  }

  @override
  bool updateShouldNotify(MessengerScope old) => messenger != old.messenger;
}
