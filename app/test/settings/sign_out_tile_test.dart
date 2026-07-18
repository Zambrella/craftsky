import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';

class _FakeAuthController extends AuthController {
  int signOutCalls = 0;
  @override
  FutureOr<void> build() => null;
  @override
  Future<SignOutResult?> signOut() async {
    signOutCalls++;
    return const SignOutResult.signedOut();
  }
}

class _FallbackAuthController extends _FakeAuthController {
  @override
  Future<SignOutResult?> signOut() async {
    signOutCalls++;
    return const SignOutResult.switchedTo('bob.test');
  }
}

class _FailedAuthController extends _FakeAuthController {
  @override
  Future<SignOutResult?> signOut() async {
    signOutCalls++;
    return null;
  }
}

void main() {
  testWidgets('UT-019 exposes only account-scoped sign out', (tester) async {
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FakeAuthController.new),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Material(child: SignOutTile()),
          ),
        ),
      ),
    );
    expect(find.text('Sign out'), findsOneWidget);
    expect(find.textContaining('Sign out all'), findsNothing);
    expect(
      find.bySemanticsLabel(RegExp('all', caseSensitive: false)),
      findsNothing,
    );
    await tester.tap(find.byType(SignOutTile));
    await tester.pump();

    final fake =
        tester.container().read(authControllerProvider.notifier)
            as _FakeAuthController;
    expect(fake.signOutCalls, 1);
  });

  testWidgets('UT-023 last-account sign-out shows success', (tester) async {
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FakeAuthController.new),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Material(child: SignOutTile()),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(SignOutTile));
    await tester.pump();

    expect(
      messenger.calls,
      [('info', 'Signed out successfully.', null)],
    );
  });

  testWidgets('IT-014 fallback sign-out identifies the active account', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FallbackAuthController.new),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Material(child: SignOutTile()),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(SignOutTile));
    await tester.pump();

    expect(
      messenger.calls,
      [
        (
          'info',
          'Signed out successfully. Now signed in as @bob.test.',
          null,
        ),
      ],
    );
  });

  testWidgets('REG-011 failed sign-out does not show success', (tester) async {
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FailedAuthController.new),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: const MaterialApp(
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Material(child: SignOutTile()),
          ),
        ),
      ),
    );

    await tester.tap(find.byType(SignOutTile));
    await tester.pump();

    expect(messenger.calls, isEmpty);
  });
}
