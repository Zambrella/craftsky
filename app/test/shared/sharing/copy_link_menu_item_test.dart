import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/sharing/copy_link_menu_item.dart';
import 'package:craftsky_app/shared/sharing/record_links.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

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

  for (final outcome in ['success', 'dismissed', 'unavailable', 'failure']) {
    testWidgets('native sharing handles $outcome and preserves reply focus', (
      tester,
    ) async {
      const channel = MethodChannel('dev.fluttercommunity.plus/share');
      const url =
          'https://app.craftsky.social/posts/did:plc:alice/root'
          '?focus=at%3A%2F%2Fdid%3Aplc%3Aalice%2F'
          'social.craftsky.feed.post%2Freply';
      final messenger = RecordingMessenger();
      MethodCall? shared;
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
        call,
      ) async {
        shared = call;
        if (outcome == 'failure') {
          throw PlatformException(code: 'share-unavailable');
        }
        return outcome == 'success'
            ? 'com.example.recipient'
            : 'dev.fluttercommunity.plus/share/$outcome';
      });
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          channel,
          null,
        ),
      );
      await tester.pumpWidget(
        MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData.copyWith(
              platform: outcome == 'success'
                  ? TargetPlatform.iOS
                  : TargetPlatform.android,
            ),
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: Scaffold(
              body: Center(
                child: Builder(
                  builder: (context) => CraftskyContextMenuButton(
                    groups: [
                      CraftskyContextMenuGroup(
                        items: linkSharingMenuItems(
                          context,
                          url,
                          canSendViaChat: false,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byType(CraftskyContextMenuButton));
      await tester.pumpAndSettle();
      expect(find.text('Copy link'), findsOneWidget);
      expect(find.text('Share'), findsOneWidget);
      expect(
        find.byIcon(
          outcome == 'success' ? CupertinoIcons.share : CraftskyIconsBold.share,
        ),
        findsOneWidget,
      );
      expect(find.text('Send via chat'), findsNothing);
      await tester.tap(find.text('Share'));
      await tester.pumpAndSettle();
      expect(shared?.method, 'share');
      final arguments = shared!.arguments as Map;
      expect(arguments['uri'], url);
      expect(arguments.containsKey('text'), isFalse);
      expect(arguments.containsKey('paths'), isFalse);
      final origin = Rect.fromLTWH(
        arguments['originX'] as double,
        arguments['originY'] as double,
        arguments['originWidth'] as double,
        arguments['originHeight'] as double,
      );
      expect(origin.isEmpty, isFalse);
      expect(
        const Rect.fromLTWH(0, 0, 800, 600).contains(origin.center),
        isTrue,
      );
      expect(find.byType(BottomSheet), findsNothing);
      if (outcome == 'failure') {
        expect(messenger.calls.single, (
          'error',
          'Could not open sharing options. Please try again.',
          null,
        ));
      } else {
        expect(messenger.calls, isEmpty);
      }
      expect(tester.takeException(), isNull);
    });
  }

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
            localizationsDelegates: appLocalizationsDelegates,
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
