import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/sharing/copy_link_menu_item.dart';
import 'package:craftsky_app/shared/sharing/record_links.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/recording_messenger.dart';

void main() {
  test('profile and event links resolve to stable public routes', () {
    final did = Did.parse('did:plc:alice');
    final profile = Uri.parse(RecordLinks.profile(did));
    expect(profile.origin, 'https://app.craftsky.social');
    expect(profile.pathSegments, ['profiles', did.value]);
    final event = Uri.parse(RecordLinks.event(did, RecordKey.parse('event-1')));
    expect(event.pathSegments, ['events', did.value, 'event-1']);
    expect(event.queryParameters, isEmpty);
  });

  for (final fail in [false, true]) {
    testWidgets('copy action reports ${fail ? 'failure' : 'success'}', (
      tester,
    ) async {
      final messenger = RecordingMessenger();
      String? copied;
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        (call) async {
          if (call.method == 'Clipboard.setData') {
            if (fail) throw PlatformException(code: 'clipboard-unavailable');
            copied = (call.arguments as Map)['text'] as String;
          }
          return null;
        },
      );
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          SystemChannels.platform,
          null,
        ),
      );
      const url = 'https://app.craftsky.social/profiles/did%3Aplc%3Aalice';
      await tester.pumpWidget(
        MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: Builder(
                builder: (context) => CraftskyContextMenuButton(
                  groups: [
                    CraftskyContextMenuGroup(
                      items: [copyLinkMenuItem(context, url)],
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byType(CraftskyContextMenuButton));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Copy link'));
      await tester.pumpAndSettle();
      expect(copied, fail ? null : url);
      expect(messenger.calls.single.$1, fail ? 'error' : 'info');
      expect(
        messenger.calls.single.$2,
        fail ? 'Could not copy link. Please try again.' : 'Link copied',
      );
    });
  }
}
