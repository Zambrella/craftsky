import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_preview.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_message_preview.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/post_fixtures.dart';

void main() {
  for (final profile in [false, true]) {
    testWidgets(
      'IT-016 ${profile ? 'profile' : 'post'} embed matches quote styling '
      'without a type label',
      (tester) async {
        final account = AccountKey('did:plc:viewer');
        final id = ChatConversationId.parse('conversation');
        final post = postFixture(text: 'Shared post text');
        final message = ChatMessage.fromMap(const {
          'id': 'message',
          'revision': 'r',
          'kind': 'text',
          'text': 'CraftSky link',
          'senderDid': 'did:plc:sender',
          'sentAt': '2026-10-30T12:00:00Z',
        });
        final preview = ChatPreview(
          target: profile
              ? ChatLinkTarget.profile(did: post.author.did)
              : ChatLinkTarget.post(did: post.author.did, rkey: post.rkey),
          title: 'Author',
          excerpt: profile ? 'Profile bio' : post.text,
          postSummary: profile ? null : PostSummaryData.fromPost(post),
        );
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              chatPreviewProvider(
                account,
                id,
                message.id,
              ).overrideWith((ref) async => preview),
              profileRepositoryProvider.overrideWithValue(
                FakeProfileRepository(
                  onFetch: (did) async => Profile(
                    did: did,
                    handle: 'alice.test',
                    displayName: 'Author',
                    crafts: const [],
                  ),
                ),
              ),
            ],
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: Scaffold(
                body: SizedBox(
                  width: 360,
                  child: ChatMessagePreview(
                    account: account,
                    conversationId: id,
                    message: message,
                  ),
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(find.byType(CraftskyCard), findsNothing);
        expect(find.text('CraftSky post'), findsNothing);
        expect(find.text('CraftSky profile'), findsNothing);
        expect(find.text(preview.excerpt), findsOneWidget);
        if (!profile) {
          expect(find.byType(PostSummary), findsOneWidget);
          expect(find.text('alice.craftsky.social'), findsOneWidget);
        }
        final material = tester
            .widgetList<Material>(find.byType(Material))
            .where((widget) => widget.shape is RoundedRectangleBorder)
            .single;
        final shape = material.shape! as RoundedRectangleBorder;
        final theme = AppTheme.lightThemeData;
        expect(
          shape.side.color,
          theme.extension<BrandSwatchTheme>()!.borderHair,
        );
        expect(
          shape.borderRadius,
          BorderRadius.circular(theme.extension<RadiusTheme>()!.r2),
        );
        expect(tester.takeException(), isNull);
      },
    );
  }
}
