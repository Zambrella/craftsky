import 'dart:async';

import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_api_client_provider.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/recording_messenger.dart';

class _RecordingAuthController extends AuthController {
  int registrationCalls = 0;

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> startRegistration() async {
    registrationCalls++;
  }
}

final class _RegistryStorage implements SessionRegistryStorage {
  SessionRegistry value = SessionRegistry.empty();

  @override
  Future<SessionRegistry> read() async => value;

  @override
  Future<void> write(SessionRegistry registry) async => value = registry;
}

void main() {
  setUpAll(initializeMappers);

  testWidgets(
    'IR-004 HTTP 502 registration_incomplete is visible on Welcome',
    (tester) async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
        ..interceptors.add(const ErrorMappingInterceptor());
      DioAdapter(dio: dio).onPost(
        '/v1/auth/registrations',
        (server) => server.reply(502, {
          'error': 'registration_incomplete',
          'message': 'safe bounded registration failure',
          'requestId': 'req_registration_incomplete',
        }),
        data: {'handoffMode': oauthHandoffModeForCurrentBuild()},
      );
      final messenger = RecordingMessenger();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authApiClientProvider.overrideWithValue(AuthApiClient(dio)),
            secureSessionRegistryStorageProvider.overrideWithValue(
              _RegistryStorage(),
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
              home: const WelcomePage(),
            ),
          ),
        ),
      );

      await tester.tap(find.text('Create an account'));
      await tester.pumpAndSettle();

      expect(
        tester.container().read(authControllerProvider).error,
        RegistrationFailure.registrationIncomplete,
      );
      expect(messenger.calls, [
        (
          'error',
          "We couldn't verify or complete account creation.",
          null,
        ),
      ]);
    },
  );

  testWidgets('IR-004 browser-launch failure is visible on Welcome', (
    tester,
  ) async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onPost(
      '/v1/auth/registrations',
      (server) => server.reply(200, {
        'authUrl': 'https://provider.example/authorize',
      }),
      data: {'handoffMode': oauthHandoffModeForCurrentBuild()},
    );
    final messenger = RecordingMessenger();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authApiClientProvider.overrideWithValue(AuthApiClient(dio)),
          secureSessionRegistryStorageProvider.overrideWithValue(
            _RegistryStorage(),
          ),
          sessionValidationLauncherProvider.overrideWithValue((_) async {}),
          authUrlLauncherProvider.overrideWithValue((_) async => false),
        ],
        child: MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const WelcomePage(),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Create an account'));
    await tester.pumpAndSettle();

    expect(
      tester.container().read(authControllerProvider).error,
      RegistrationFailure.registrationIncomplete,
    );
    expect(messenger.calls.single.$2, contains("couldn't verify or complete"));
  });

  testWidgets('AT-001 starts registration directly from Welcome', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const WelcomePage(),
        ),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(find.widgetWithText(ChunkyButton, 'Sign in'), findsOneWidget);
    const disclosure =
        'Bluesky hosts your portable account, which you can use with Craftsky.';
    expect(find.text(disclosure), findsOneWidget);
    expect(find.text('Create an account'), findsOneWidget);

    final disclosureTopLeft = tester.getTopLeft(find.text(disclosure));
    final actionTopLeft = tester.getTopLeft(find.text('Create an account'));
    expect(disclosureTopLeft.dy, lessThan(actionTopLeft.dy));

    await tester.tap(find.text('Create an account'));
    await tester.pump();

    final controller =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(controller.registrationCalls, 1);
    expect(find.byType(Dialog), findsNothing);
    expect(find.byType(WelcomePage), findsOneWidget);
  });

  testWidgets('AT-002 Sign in still opens the existing handle form', (
    tester,
  ) async {
    final router = GoRouter(
      initialLocation: '/welcome',
      routes: [
        GoRoute(path: '/welcome', builder: (_, _) => const WelcomePage()),
        GoRoute(path: '/sign-in', builder: (_, _) => const SignInPage()),
      ],
    );
    addTearDown(router.dispose);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp.router(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          routerConfig: router,
        ),
      ),
    );

    await tester.tap(find.widgetWithText(ChunkyButton, 'Sign in'));
    await tester.pumpAndSettle();

    expect(find.byType(SignInPage), findsOneWidget);
    expect(find.text('Handle'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Continue'), findsOneWidget);
  });

  testWidgets('AT-003 resume without a callback leaves Welcome retryable', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const WelcomePage(),
        ),
      ),
    );

    await tester.tap(find.text('Create an account'));
    await tester.pump();
    tester.binding
      ..handleAppLifecycleStateChanged(AppLifecycleState.paused)
      ..handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();

    final action = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Create an account'),
    );
    expect(action.onPressed, isNotNull);
    expect(find.byType(Dialog), findsNothing);

    await tester.tap(find.text('Create an account'));
    await tester.pump();

    final controller =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(controller.registrationCalls, 2);
  });

  testWidgets('UT-012 Welcome renders the exact localized disclosure first', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const WelcomePage(),
        ),
      ),
    );

    const disclosure =
        'Bluesky hosts your portable account, which you can use with Craftsky.';
    final disclosureFinder = find.text(disclosure);
    final actionFinder = find.text('Create an account');
    expect(disclosureFinder, findsOneWidget);
    expect(
      tester.getTopLeft(disclosureFinder).dy,
      lessThan(tester.getTopLeft(actionFinder).dy),
    );
  });
}
