import 'package:flutter_test/flutter_test.dart';

Future<void> pumpUntil(
  WidgetTester tester,
  bool Function() condition, {
  required String description,
  Duration timeout = const Duration(seconds: 3),
  Duration interval = const Duration(milliseconds: 20),
}) async {
  var elapsed = Duration.zero;
  while (!condition()) {
    if (elapsed >= timeout) {
      fail('Timed out waiting for $description after $timeout.');
    }
    await tester.pump(interval);
    elapsed += interval;
  }
}

Future<void> pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  required String description,
  Duration timeout = const Duration(seconds: 3),
}) => pumpUntil(
  tester,
  () => finder.evaluate().isNotEmpty,
  description: description,
  timeout: timeout,
);

Future<void> pumpUntilAbsent(
  WidgetTester tester,
  Finder finder, {
  required String description,
  Duration timeout = const Duration(seconds: 3),
}) => pumpUntil(
  tester,
  () => finder.evaluate().isEmpty,
  description: description,
  timeout: timeout,
);
