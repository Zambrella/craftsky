import 'dart:async';

import 'package:craftsky_app/auth/data/auth_api_client.dart';
import 'package:craftsky_app/auth/data/oauth_handoff_mode.dart';
import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
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
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:dio/dio.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/recording_messenger.dart';

class _RecordingAuthController extends AuthController {
  int registrationCalls = 0;
  final List<String> signInCalls = [];

  @override
  FutureOr<void> build() => null;

  @override
  Future<void> startRegistration() async {
    registrationCalls++;
  }

  @override
  Future<void> signIn({required String handle}) async {
    signInCalls.add(handle);
  }
}

class _LoadingAuthController extends AuthController {
  @override
  Future<void> build() => Completer<void>().future;
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

  testWidgets('IR-004 HTTP 502 registration_incomplete is visible on Welcome', (
    tester,
  ) async {
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

    await tester.tap(find.text('Register'));
    await tester.pumpAndSettle();

    expect(
      tester.container().read(authControllerProvider).error,
      RegistrationFailure.registrationIncomplete,
    );
    expect(messenger.calls, [
      ('error', "We couldn't verify or complete account creation.", null),
    ]);
  });

  testWidgets('IR-004 browser-launch failure is visible on Welcome', (
    tester,
  ) async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
    DioAdapter(dio: dio).onPost(
      '/v1/auth/registrations',
      (server) =>
          server.reply(200, {'authUrl': 'https://provider.example/authorize'}),
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

    await tester.tap(find.text('Register'));
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
    expect(find.text('Join CraftSky'), findsOneWidget);
    expect(find.widgetWithText(ChunkyButton, 'Sign in'), findsOneWidget);
    expect(find.text('Register'), findsOneWidget);
    expect(
      find.text(
        "You'll create your account with Bluesky, then return to CraftSky.",
      ),
      findsOneWidget,
    );
    expect(find.byType(BrandTextField), findsOneWidget);

    await tester.tap(find.text('Register'));
    await tester.pump();

    final controller =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(controller.registrationCalls, 1);
    expect(find.byType(Dialog), findsNothing);
    expect(find.byType(WelcomePage), findsOneWidget);
  });

  testWidgets('AT-002 signs in with the entered handle from Welcome', (
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

    await tester.enterText(find.byType(BrandTextField), 'alice.bsky.social');
    final signInButton = find.widgetWithText(ChunkyButton, 'Sign in');
    await tester.ensureVisible(signInButton);
    await tester.tap(signInButton);
    await tester.pump();

    final controller =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(controller.signInCalls, ['alice.bsky.social']);
  });

  testWidgets('Welcome labels both loading actions as redirecting', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_LoadingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: const WelcomePage(),
        ),
      ),
    );

    expect(
      find.widgetWithText(ChunkyButton, 'Redirecting...'),
      findsNWidgets(2),
    );
    expect(find.byType(StitchProgressIndicator), findsNothing);
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

    await tester.tap(find.text('Register'));
    await tester.pump();
    tester.binding
      ..handleAppLifecycleStateChanged(AppLifecycleState.inactive)
      ..handleAppLifecycleStateChanged(AppLifecycleState.hidden)
      ..handleAppLifecycleStateChanged(AppLifecycleState.paused)
      ..handleAppLifecycleStateChanged(AppLifecycleState.hidden)
      ..handleAppLifecycleStateChanged(AppLifecycleState.inactive)
      ..handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();

    final action = tester.widget<ChunkyButton>(
      find.widgetWithText(ChunkyButton, 'Register'),
    );
    expect(action.onPressed, isNotNull);
    expect(find.byType(Dialog), findsNothing);

    await tester.tap(find.text('Register'));
    await tester.pump();

    final controller =
        tester.container().read(authControllerProvider.notifier)
            as _RecordingAuthController;
    expect(controller.registrationCalls, 2);
  });

  testWidgets('Welcome expands the Atmosphere account explainer', (
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

    expect(
      find.textContaining('CraftSky is built on the AT Protocol'),
      findsNothing,
    );
    await tester.ensureVisible(find.text('What is an Atmosphere account?'));
    await tester.tap(find.text('What is an Atmosphere account?'));
    await tester.pumpAndSettle();
    expect(
      find.textContaining('CraftSky is built on the AT Protocol'),
      findsOneWidget,
    );
  });

  testWidgets('Welcome opens Terms and Privacy links', (tester) async {
    final opened = <Uri>[];
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_RecordingAuthController.new),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: WelcomePage(
            linkLauncher: (uri) async {
              opened.add(uri);
              return true;
            },
          ),
        ),
      ),
    );

    final legalLinks = find.byKey(const Key('legal-links'));
    await tester.ensureVisible(legalLinks);
    final richText = tester.widget<Text>(legalLinks);
    final spans = (richText.textSpan! as TextSpan).children!;
    ((spans[1] as TextSpan).recognizer! as TapGestureRecognizer).onTap!();
    ((spans[3] as TextSpan).recognizer! as TapGestureRecognizer).onTap!();
    await tester.pump();

    expect(opened, [
      Uri.parse('https://craftsky.social/terms'),
      Uri.parse('https://craftsky.social/privacy'),
    ]);
  });

  testWidgets('Welcome remains usable on a narrow screen', (tester) async {
    tester.view
      ..physicalSize = const Size(320, 568)
      ..devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

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

    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.byKey(const Key('legal-links')));
    expect(find.byKey(const Key('legal-links')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
