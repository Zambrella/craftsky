import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

const businessAccessibilityMatrix = <({Size size, double textScale})>[
  (size: Size(320, 568), textScale: 1),
  (size: Size(320, 568), textScale: 2),
  (size: Size(800, 600), textScale: 1),
  (size: Size(800, 600), textScale: 2),
];

String businessConstraintLabel(({Size size, double textScale}) constraint) =>
    '${constraint.size.width.toInt()}x${constraint.size.height.toInt()} '
    'at ${constraint.textScale.toStringAsFixed(1)}';

Future<void> setBusinessAccessibilityConstraint(
  WidgetTester tester,
  ({Size size, double textScale}) constraint,
) async {
  await tester.binding.setSurfaceSize(constraint.size);
  addTearDown(() => tester.binding.setSurfaceSize(null));
  tester.platformDispatcher.textScaleFactorTestValue = constraint.textScale;
  addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
}

void expectNoAccessibilityLayoutException(WidgetTester tester) {
  expect(tester.takeException(), isNull);
}

Future<void> expectKeyboardFocus(WidgetTester tester) async {
  await tester.sendKeyEvent(LogicalKeyboardKey.tab);
  expect(FocusManager.instance.primaryFocus, isNotNull);
}

void requestKeyboardFocus(WidgetTester tester, Finder target) {
  final node = Focus.maybeOf(
    tester.element(target),
    createDependency: false,
  )!;
  expect(node.canRequestFocus, isTrue);
  node.requestFocus();
}

void expectKeyboardFocusOn(Finder target) {
  final node = Focus.maybeOf(
    target.evaluate().single,
    createDependency: false,
  );
  expect(
    FocusManager.instance.primaryFocus,
    same(node),
    reason: 'Expected keyboard focus on $target',
  );
}

Future<void> pressTabAndExpectFocus(
  WidgetTester tester,
  Finder target,
) async {
  await tester.sendKeyEvent(LogicalKeyboardKey.tab);
  await tester.pump();
  expectKeyboardFocusOn(target);
}
