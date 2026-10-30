import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/keys.dart';
import 'package:patrol/patrol.dart';

final class AuthModule {
  new(this.$);

  final PatrolIntegrationTester $;

  Future<void> signIn() async {
    const handle = String.fromEnvironment('E2E_HANDLE');
    const encodedPassword = String.fromEnvironment('E2E_PASSWORD_BASE64');
    final password = encodedPassword.isEmpty
        ? ''
        : utf8.decode(base64Decode(encodedPassword));
    if (handle.isEmpty || password.isEmpty) {
      throw StateError('Configure the dedicated account in app/.patrol.env.');
    }
    await $(keys.auth.handle).enterText(handle);
    await $(keys.auth.signIn).tap(settlePolicy: SettlePolicy.noSettle);
    if (Platform.isIOS) {
      await _signInIOS(password);
    } else {
      await _signInAndroid(password);
    }
  }

  Future<void> _signInAndroid(String password) async {
    final deadline = DateTime.now().add(const Duration(seconds: 90));
    var submittedPassword = false;
    do {
      if ($(keys.onboarding.bio).exists) return;
      final tree = await $.platform.android.getNativeViews(null);
      final views = tree.roots.expand(_flatten).toList();
      if (views.any((view) => view.text == 'Wrong identifier or password')) {
        throw StateError('Bluesky rejected the test account credentials.');
      }
      final browserUrl = views
          .where((view) => view.resourceName == 'com.android.chrome:id/url_bar')
          .firstOrNull
          ?.text;
      final isBlueskyLogin =
          browserUrl?.startsWith('bsky.social/oauth/authorize?') == true ||
          browserUrl?.startsWith('https://bsky.social/oauth/authorize?') ==
              true;
      final fields = views.where(
        (view) =>
            view.className == 'android.widget.EditText' &&
            view.applicationPackage == 'com.android.chrome' &&
            (view.resourceName?.isNotEmpty ?? false) &&
            view.resourceName != 'com.android.chrome:id/url_bar',
      );
      if (isBlueskyLogin && !submittedPassword && fields.length == 2) {
        await $.platform.android.enterText(
          AndroidSelector(resourceName: fields.last.resourceName),
          text: password,
          keyboardBehavior: KeyboardBehavior.alternative,
        );
        await $.platform.android.tap(
          const AndroidSelector(
            className: 'android.widget.Button',
            text: 'Sign in',
          ),
        );
        submittedPassword = true;
      } else {
        final approval = views
            .where(
              (view) =>
                  view.className == 'android.widget.Button' &&
                  const [
                    'Authorize',
                    'Accept',
                    'Continue',
                    'Open',
                    'Open CraftSky',
                  ].contains(view.text),
            )
            .firstOrNull;
        if (approval != null) {
          await $.platform.android.tap(
            AndroidSelector(className: approval.className, text: approval.text),
          );
        }
      }
      await Future<void>.delayed(const Duration(milliseconds: 500));
    } while (DateTime.now().isBefore(deadline));
    throw StateError(
      'OAuth did not return to onboarding. Inspect the emulator browser.',
    );
  }

  Future<void> _signInIOS(String password) async {
    const safari = 'com.apple.mobilesafari';
    final deadline = DateTime.now().add(const Duration(seconds: 90));
    var submittedPassword = false;
    do {
      if ($(keys.onboarding.bio).exists) return;
      final state = await _iosOAuthRequest('oauthState');
      final buttons = (state['buttons'] as List).cast<String>();
      if (buttons.contains('Not Now')) {
        await $.platform.ios.tap(
          const IOSSelector(
            elementType: IOSElementType.button,
            label: 'Not Now',
          ),
          appId: safari,
        );
        continue;
      }
      if (state['wrongCredentials'] == true) {
        throw StateError('Bluesky rejected the test account credentials.');
      }
      if (!submittedPassword && state['passwordVisible'] == true) {
        final result = await _iosOAuthRequest(
          'enterOAuthPassword',
          {'password': password},
        );
        if (result['error'] != null) {
          throw StateError(switch (result['error']) {
            'safariURLUnavailable' =>
              'Safari URL field did not appear after keyboard dismissal.',
            'unexpectedOrigin' => 'Expected the Bluesky OAuth origin.',
            'passwordFieldUnavailable' =>
              'Expected the Bluesky OAuth password field.',
            'keyboardUnavailable' =>
              'The Safari password keyboard did not appear.',
            'passwordKeyboardStillVisible' =>
              'The Safari password keyboard did not dismiss.',
            'addressKeyboardStillVisible' =>
              'The Safari address keyboard did not dismiss.',
            'safariIntroductionBlocked' =>
              'A Safari or keyboard introduction could not be dismissed.',
            'nativeIdleConfigurationUnavailable' =>
              'This XCTest version lacks the native idle timeout hook.',
            _ => 'Secure OAuth password entry failed.',
          });
        }
        await $.platform.ios.tap(
          const IOSSelector(
            elementType: IOSElementType.button,
            label: 'Sign in',
          ),
          appId: safari,
        );
        submittedPassword = true;
      } else {
        final approval = buttons
            .where(
              (label) => const [
                'Authorize',
                'Continue',
                'Open',
                'Open CraftSky',
              ].contains(label),
            )
            .firstOrNull;
        if (approval != null) {
          await $.platform.ios.tap(
            IOSSelector(
              elementType: IOSElementType.button,
              label: approval,
            ),
            appId: safari,
          );
        }
      }
      await Future<void>.delayed(const Duration(milliseconds: 500));
    } while (DateTime.now().isBefore(deadline));
    throw StateError('OAuth did not return to onboarding. Inspect Safari.');
  }

  Future<Map<String, dynamic>> _iosOAuthRequest(
    String action, [
    Map<String, Object?> payload = const {},
  ]) async {
    final client = HttpClient();
    try {
      final request = await client.postUrl(
        patrolNativeServerUri.resolve('/craftsky/$action'),
      );
      request.headers.contentType = ContentType.json;
      request.write(jsonEncode(payload));
      final response = await request.close();
      final body = await utf8.decoder.bind(response).join();
      if (response.statusCode != HttpStatus.ok) {
        throw StateError('The native OAuth helper failed.');
      }
      return jsonDecode(body) as Map<String, dynamic>;
    } finally {
      client.close(force: true);
    }
  }

  Iterable<AndroidNativeView> _flatten(AndroidNativeView view) sync* {
    yield view;
    for (final child in view.children) {
      yield* _flatten(child);
    }
  }
}
