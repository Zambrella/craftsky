import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  _FakeAuthController({required this.onComplete});
  final Future<void> Function(String token) onComplete;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> completeFromDeepLink(String token) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() => onComplete(token));
  }
}

void main() {
  testWidgets('calls completeFromDeepLink with the token on init',
      (tester) async {
    final seen = <String>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(onComplete: (t) async => seen.add(t)),
          ),
        ],
        child: const MaterialApp(
          home: AuthCompletePage(token: 'tok-123'),
        ),
      ),
    );
    await tester.pump(); // one frame for addPostFrameCallback
    expect(seen, ['tok-123']);
  });

  testWidgets('renders spinner by default', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(onComplete: (_) async {}),
          ),
        ],
        child: const MaterialApp(home: AuthCompletePage(token: 't')),
      ),
    );
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('Signing in…'), findsOneWidget);
  });

  testWidgets('renders retry text on AuthError', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (_) async => throw const NoPendingSignIn(),
            ),
          ),
        ],
        child: const MaterialApp(home: AuthCompletePage(token: 't')),
      ),
    );
    await tester.pump();
    await tester.pump(); // allow AsyncError to propagate
    expect(
      find.textContaining('sign in again', findRichText: true),
      findsOneWidget,
    );
  });
}
