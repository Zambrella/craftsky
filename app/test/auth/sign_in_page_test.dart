import 'dart:async';

import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/auth/widgets/registration_action.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/recording_messenger.dart';

class _RecordingAuthController extends AuthController {
  final List<String> signInCalls = [];
  int registrationCalls = 0;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> signIn({required String handle}) async {
    signInCalls.add(handle);
  }

  @override
  Future<void> startRegistration() async {
    registrationCalls++;
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  _RegistryStorage(this.value);

  SessionRegistry value;

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

class _ErroringAuthController extends AuthController {
  @override
  FutureOr<void> build() => null;

  @override
  Future<void> signIn({required String handle}) async {
    state = const AsyncLoading();
    state = const AsyncError(InvalidHandle(), StackTrace.empty);
  }
}

class _AccountLimitAuthController extends AuthController {
  @override
  FutureOr<void> build() => null;

  @override
  Future<void> startRegistration() async {
    state = const AsyncLoading();
    state = const AsyncError(AccountLimitReached(), StackTrace.empty);
  }
}

void main() {
  SessionRegistry registryWith(int count) {
    var registry = SessionRegistry.empty();
    for (var index = 0; index < count; index++) {
      registry = registry.upsertAndActivate(
        token: 'token-$index',
        did: 'did:plc:a$index',
        handle: 'a$index.test',
      );
    }
    return registry;
  }

  testWidgets('renders a handle field and a Continue button', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const SignInPage(),
        ),
      ),
    );
    expect(find.byType(BrandTextField), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Continue'), findsOneWidget);
  });

  testWidgets(
    'tapping Continue dispatches AuthController.signIn with text',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_RecordingAuthController.new),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SignInPage(),
          ),
        ),
      );
      await tester.enterText(
        find.byType(BrandTextField),
        '  @alice.bsky.social ',
      );
      await tester.tap(find.widgetWithText(ChunkyButton, 'Continue'));
      await tester.pump();

      final fake =
          tester.container().read(authControllerProvider.notifier)
              as _RecordingAuthController;
      expect(fake.signInCalls, ['  @alice.bsky.social ']);
      // (Controller trims — that's unit-tested in auth_controller_test.dart.)
    },
  );

  testWidgets(
    'IR-004 HTTP 502 registration_provider_unavailable '
    'is visible on Add Account',
    (tester) async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio).onPost(
        '/v1/auth/registrations',
        (server) => server.reply(502, {
          'error': 'registration_provider_unavailable',
          'message': 'safe bounded registration failure',
          'requestId': 'req_registration_provider_unavailable',
        }),
        data: {'handoffMode': oauthHandoffModeForCurrentBuild()},
      );
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authApiClientProvider.overrideWithValue(AuthApiClient(dio)),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(SessionRegistry.empty()),
            ),
            sessionValidationLauncherProvider.overrideWithValue((_) async {}),
            authUrlLauncherProvider.overrideWithValue((_) async => true),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const SignInPage(mode: SignInMode.addAccount),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Create an account'));
      await tester.pumpAndSettle();

      expect(
        tester.container().read(authControllerProvider).error,
        RegistrationFailure.providerUnavailable,
      );
      expect(messenger.calls, [
        (
          'error',
          'Bluesky is temporarily unavailable. Please try again.',
          null,
        ),
      ]);
    },
  );

  testWidgets('IR-004 Add Account preserves distinct account-limit copy', (
    tester,
  ) async {
    final messenger = RecordingMessenger();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_AccountLimitAuthController.new),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(SessionRegistry.empty()),
          ),
          sessionValidationLauncherProvider.overrideWithValue((_) async {}),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SignInPage(mode: SignInMode.addAccount),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Create an account'));
    await tester.pump();

    expect(messenger.calls.single.$2, 'Maximum of 5 accounts');
    expect(messenger.calls.single.$2, isNot(contains('Something went wrong')));
  });

  testWidgets(
    'an InvalidHandle error dispatches showError with the localised message',
    (tester) async {
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_ErroringAuthController.new),
          ],
          child: MessengerScope(
            messenger: messenger,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: AppLocalizations.localizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: const SignInPage(),
            ),
          ),
        ),
      );

      await tester.enterText(find.byType(BrandTextField), 'whatever');
      await tester.tap(find.widgetWithText(ChunkyButton, 'Continue'));
      await tester.pump();

      expect(messenger.calls.length, 1);
      expect(messenger.calls.first.$1, 'error');
      expect(messenger.calls.first.$2, "We couldn't recognise that handle.");
    },
  );

  testWidgets('AT-002 Add Account keeps both paths subject to the limit', (
    tester,
  ) async {
    for (final count in [4, SessionRegistry.maxRetainedAccounts]) {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_RecordingAuthController.new),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registryWith(count)),
            ),
            sessionValidationLauncherProvider.overrideWithValue((_) async {}),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SignInPage(mode: SignInMode.addAccount),
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.byType(BrandTextField), findsOneWidget);
      expect(find.byType(RegistrationAction), findsOneWidget);
      expect(find.text('Create an account'), findsOneWidget);
      final continueButton = tester.widget<ChunkyButton>(
        find.widgetWithText(ChunkyButton, 'Continue'),
      );
      final registration = tester.widget<RegistrationAction>(
        find.byType(RegistrationAction),
      );

      if (count < SessionRegistry.maxRetainedAccounts) {
        expect(continueButton.onPressed, isNotNull);
        expect(registration.enabled, isTrue);
        await tester.enterText(find.byType(BrandTextField), 'alice.test');
        await tester.tap(find.widgetWithText(ChunkyButton, 'Continue'));
        await tester.tap(find.text('Create an account'));
        await tester.pump();
        final controller =
            tester.container().read(authControllerProvider.notifier)
                as _RecordingAuthController;
        expect(controller.signInCalls, ['alice.test']);
        expect(controller.registrationCalls, 1);
      } else {
        expect(continueButton.onPressed, isNull);
        expect(registration.enabled, isFalse);
      }
    }
  });

  testWidgets(
    'UT-012 Add Account renders the exact localized disclosure first',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_RecordingAuthController.new),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(registryWith(1)),
            ),
            sessionValidationLauncherProvider.overrideWithValue((_) async {}),
          ],
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const SignInPage(mode: SignInMode.addAccount),
          ),
        ),
      );
      await tester.pumpAndSettle();

      const disclosure =
          'Bluesky hosts your portable account, which you can use with '
          'Craftsky.';
      final disclosureFinder = find.text(disclosure);
      final actionFinder = find.text('Create an account');
      expect(disclosureFinder, findsOneWidget);
      expect(
        tester.getTopLeft(disclosureFinder).dy,
        lessThan(tester.getTopLeft(actionFinder).dy),
      );
    },
  );
}
