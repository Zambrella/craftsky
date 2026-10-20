import 'dart:convert';

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

  Iterable<AndroidNativeView> _flatten(AndroidNativeView view) sync* {
    yield view;
    for (final child in view.children) {
      yield* _flatten(child);
    }
  }
}
