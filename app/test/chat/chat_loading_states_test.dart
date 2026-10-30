import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_conversation_page.dart';
import 'package:craftsky_app/chat/pages/messaging_inbox_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_conversation_controller.dart';
import 'package:craftsky_app/chat/services/chat_inbox_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_skeleton.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final conversation in [false, true]) {
    testWidgets('UT-010 skeleton content fills a tall viewport '
        'conversation=$conversation', (tester) async {
      await tester.binding.setSurfaceSize(const Size(390, 1100));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Scaffold(
            body: conversation
                ? const ChatConversationSkeleton()
                : const ChatListSkeleton(),
          ),
        ),
      );
      final scrollable = tester.state<ScrollableState>(
        find.byType(Scrollable),
      );
      expect(scrollable.position.maxScrollExtent, greaterThan(0));
      expect(tester.takeException(), isNull);
    });
  }
  for (final surface in ['chats', 'requests', 'conversation']) {
    for (final pendingStatus in [true, false]) {
      testWidgets(
        'UT-010 $surface uses skeletons while '
        '${pendingStatus ? 'status' : 'content'} loads',
        (tester) async {
          await tester.binding.setSurfaceSize(const Size(320, 568));
          addTearDown(() => tester.binding.setSurfaceSize(null));
          final account = AccountKey('did:plc:owner');
          final status = Completer<ChatStatus>();
          if (!pendingStatus) {
            status.complete(
              ChatStatus.fromMap(const {
                'enabled': true,
                'pushEnabled': true,
                'generation': 1,
                'state': 'ready',
                'capabilities': {
                  'canRead': true,
                  'canSend': true,
                  'canReport': true,
                  'canSetPolicy': true,
                },
              }),
            );
          }
          final bucket = surface == 'requests'
              ? ChatBucket.request
              : ChatBucket.accepted;
          final id = ChatConversationId.parse('conversation');
          await tester.pumpWidget(
            ProviderScope(
              overrides: [
                secureSessionRegistryStorageProvider.overrideWithValue(
                  _Storage(),
                ),
                chatStatusProvider(account).overrideWith(
                  () => _Status(status.future),
                ),
                chatUnreadCountsProvider(account).overrideWith(
                  (ref) async => ChatUnreadCounts.fromMap(const {
                    'accepted': 0,
                    'requests': 0,
                  }),
                ),
                chatInboxControllerProvider(account, bucket).overrideWith(
                  (ref) => Completer<ChatInboxController>().future,
                ),
                chatConversationControllerProvider(account, id).overrideWith(
                  (ref) => Completer<ChatConversationController>().future,
                ),
              ],
              child: MaterialApp(
                theme: AppTheme.lightThemeData,
                localizationsDelegates: appLocalizationsDelegates,
                supportedLocales: AppLocalizations.supportedLocales,
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(context).copyWith(
                    textScaler: const TextScaler.linear(2),
                    disableAnimations: true,
                  ),
                  child: child!,
                ),
                home: surface == 'conversation'
                    ? MessagingConversationPage(
                        account: account,
                        conversationId: id,
                        onClose: () {},
                        onReport: (_) {},
                        onAuthorize: (_) {},
                      )
                    : MessagingInboxPage(
                        account: account,
                        bucket: bucket,
                        onOpen: (_) {},
                        onNewMessage: () {},
                        onRequests: () {},
                        onSettings: () {},
                        onAuthorize: (_) {},
                      ),
              ),
            ),
          );
          await tester.pump();
          await tester.pump(const Duration(milliseconds: 100));
          expect(find.byType(CraftskySkeletonList), findsOneWidget);
          expect(find.byType(StitchProgressIndicator), findsNothing);
          final headerSkeleton = find.descendant(
            of: find.byType(AppBar),
            matching: find.byType(CraftskySkeleton),
          );
          if (surface == 'conversation') {
            expect(headerSkeleton, findsOneWidget);
            expect(
              tester.getBottomLeft(headerSkeleton).dy,
              lessThanOrEqualTo(tester.getBottomLeft(find.byType(AppBar)).dy),
            );
          } else {
            expect(headerSkeleton, findsNothing);
          }
          expect(
            find.byType(
              surface == 'conversation'
                  ? ChatConversationSkeleton
                  : ChatListSkeleton,
            ),
            findsOneWidget,
          );
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
}

class _Status extends MessagingStatus {
  new(this.pending);
  final Future<ChatStatus> pending;
  @override
  Future<ChatStatus> build(AccountKey account) => pending;
}

class _Storage implements SessionRegistryStorage {
  @override
  Future<SessionRegistry> read() async =>
      SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic',
        did: 'did:plc:owner',
        handle: 'owner.test',
      );
  @override
  Future<void> write(SessionRegistry registry) async {}
}
