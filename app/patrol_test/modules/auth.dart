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
      final tree = await $.platform.ios.getNativeViews(
        null,
        appId: safari,
        iosInstalledApps: [safari],
      );
      final views = tree.roots.expand(_flattenIOS).toList();
      if (views.any((view) => view.label == 'Wrong identifier or password')) {
        throw StateError('Bluesky rejected the test account credentials.');
      }
      final passwordField = views
          .where((view) => view.elementType == IOSElementType.secureTextField)
          .firstOrNull;
      if (!submittedPassword && passwordField != null) {
        final client = HttpClient();
        try {
          final request = await client.postUrl(
            patrolNativeServerUri.resolve('/craftsky/enterOAuthPassword'),
          );
          request.headers.contentType = ContentType.json;
          request.write(jsonEncode({'password': password}));
          final response = await request.close();
          await response.drain<void>();
          if (response.statusCode != HttpStatus.ok) {
            throw StateError('Secure OAuth password entry failed.');
          }
        } finally {
          client.close(force: true);
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
        final approval = views
            .where(
              (view) =>
                  view.elementType == IOSElementType.button &&
                  const [
                    'Authorize',
                    'Continue',
                    'Open',
                    'Open CraftSky',
                  ].contains(view.label),
            )
            .firstOrNull;
        if (approval != null) {
          await $.platform.ios.tap(
            IOSSelector(
              elementType: IOSElementType.button,
              label: approval.label,
            ),
            appId: safari,
          );
        }
      }
      await Future<void>.delayed(const Duration(milliseconds: 500));
    } while (DateTime.now().isBefore(deadline));
    throw StateError('OAuth did not return to onboarding. Inspect Safari.');
  }

  Iterable<IOSNativeView> _flattenIOS(IOSNativeView view) sync* {
    yield view;
    for (final child in view.children) {
      yield* _flattenIOS(child);
    }
  }

  Iterable<AndroidNativeView> _flatten(AndroidNativeView view) sync* {
    yield view;
    for (final child in view.children) {
      yield* _flatten(child);
    }
  }
}
