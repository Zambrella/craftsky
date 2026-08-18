import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/pages/auth_complete_page.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  _FakeAuthController({required this.onComplete});
  final Future<void> Function(String code) onComplete;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> completeFromDeepLink(String code) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() => onComplete(code));
  }
}

void main() {
  testWidgets('calls completeFromDeepLink with the browser code on init', (
    tester,
  ) async {
    final seen = <String>[];
    final neverCompletes = Completer<void>();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (t) async {
                seen.add(t);
                await neverCompletes.future;
              },
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AuthCompletePage(code: 'code-123'),
        ),
      ),
    );
    await tester.pump(); // one frame for addPostFrameCallback
    expect(seen, ['code-123']);
  });

  testWidgets('renders spinner by default', (tester) async {
    final neverCompletes = Completer<void>();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (_) => neverCompletes.future,
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AuthCompletePage(code: 'code'),
        ),
      ),
    );
    await tester.pump();
    expect(find.byType(StitchProgressIndicator), findsOneWidget);
    expect(find.text('Signing in…'), findsOneWidget);
  });

  testWidgets('retries a transient handoff failure with the same code', (
    tester,
  ) async {
    final seen = <String>[];
    final retryInProgress = Completer<void>();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (code) async {
                seen.add(code);
                if (seen.length == 1) throw const ServerUnavailable();
                await retryInProgress.future;
              },
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AuthCompletePage(code: 'retry-code'),
        ),
      ),
    );
    await tester.pump();
    await tester.pump(); // allow AsyncError to propagate
    expect(find.text('Retry'), findsOneWidget);

    await tester.tap(find.text('Retry'));
    await tester.pump();
    await tester.pump();

    expect(seen, ['retry-code', 'retry-code']);
  });

  testWidgets('a callback without a code fails closed without exchanging', (
    tester,
  ) async {
    final seen = <String>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (code) async => seen.add(code),
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AuthCompletePage(),
        ),
      ),
    );

    await tester.pump();
    expect(seen, isEmpty);
    expect(
      find.textContaining('sign-in link expired', findRichText: true),
      findsOneWidget,
    );
  });

  testWidgets('renders a coarse pending-deletion outcome without signing in', (
    tester,
  ) async {
    final seen = <String>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(
            () => _FakeAuthController(
              onComplete: (token) async => seen.add(token),
            ),
          ),
        ],
        child: const MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: AuthCompletePage(error: 'account_deletion_pending'),
        ),
      ),
    );

    await tester.pump();
    expect(seen, isEmpty);
    expect(find.textContaining('already in progress'), findsOneWidget);
  });
}
