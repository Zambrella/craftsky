import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/messaging_inbox_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/widgets/chat_activity_scope.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  for (final bucket in ChatBucket.values) {
    testWidgets(
      'IT-001/UT-010 ${bucket.name} fits large text and preserves account isolation',
      (tester) async {
        await tester.binding.setSurfaceSize(const Size(320, 568));
        addTearDown(() => tester.binding.setSurfaceSize(null));
        final account = AccountKey('did:plc:alice');
        final registry = SessionRegistry.empty().upsertAndActivate(
          token: 'synthetic-token',
          did: account.did.value,
          handle: 'alice.test',
        );
        final repository = _Repository();
        final firstPage = await repository.conversations(bucket: bucket);
        final initialLoad = Completer<ChatConversationPage>();
        repository.pendingConversations = initialLoad;
        ChatConversationId? opened;
        var newMessage = 0;
        var requests = 0;
        await tester.pumpWidget(
          ProviderScope(
            overrides: [
              secureSessionRegistryStorageProvider.overrideWithValue(
                _Storage(registry),
              ),
              accountChatRepositoryProvider(account)
                  .overrideWith((ref) async => repository),
            ],
            child: MaterialApp(
              builder: (context, child) => MediaQuery(
                data: MediaQuery.of(context)
                    .copyWith(textScaler: const TextScaler.linear(2)),
                child: child!,
              ),
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              home: ChatActivityScope(
                account: account,
                view: ChatView.inbox,
                child: MessagingInboxPage(
                  account: account,
                  bucket: bucket,
                  onOpen: (id) => opened = id,
                  onNewMessage: () => newMessage++,
                  onRequests: () => requests++,
                  onSettings: () {},
                  onAuthorize: (_) {},
                ),
              ),
            ),
          ),
        );
        await tester.pump();
        await tester.pump(const Duration(milliseconds: 100));
        expect(find.byType(CraftskySkeletonList), findsOneWidget);
        expect(find.byType(AccountRowSkeleton), findsWidgets);
        expect(find.byType(StitchProgressIndicator), findsNothing);
        repository.pendingConversations = null;
        initialLoad.complete(firstPage);
        await tester.pumpAndSettle();
        final peer = bucket == ChatBucket.accepted
            ? 'external.test'
            : 'request.external.test';
        expect(find.text('@$peer'), findsOneWidget);
        expect(find.text('CraftSky peer'), findsOneWidget);
        expect(find.text('Conversation preview'), findsOneWidget);
        final timestamp = tester.widget<RelativeTimeText>(
          find.byType(RelativeTimeText),
        );
        expect(timestamp.timestamp, DateTime.utc(2026, 10, 21, 12));
        final tile = find.ancestor(
          of: find.byType(RelativeTimeText),
          matching: find.byType(ListTile),
        );
        expect(
          tester.getRect(find.byType(RelativeTimeText)).right,
          closeTo(
            tester
                .getRect(
                  find.descendant(
                    of: tile,
                    matching: find.byType(Badge),
                  ),
                )
                .right,
            1,
          ),
        );

        expect(find.byType(TabBar), findsNothing);
        expect(repository.heartbeats, 1);
        await tester.pump(const Duration(seconds: 30));
        await tester.pumpAndSettle();
        expect(repository.heartbeats, 2);
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.inactive,
        );
        tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
        tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
        await tester.pump(const Duration(seconds: 60));
        expect(repository.heartbeats, 2);
        tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.hidden);
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.inactive,
        );
        tester.binding.handleAppLifecycleStateChanged(
          AppLifecycleState.resumed,
        );
        await tester.pumpAndSettle();
        expect(repository.heartbeats, 3);

        final previousPage = await repository.conversations(bucket: bucket);
        final pending = Completer<ChatConversationPage>();
        repository.pendingConversations = pending;
        final calls = repository.listCalls;
        expect(find.byType(YarnRefreshScrollView), findsOneWidget);
        expect(find.byType(RefreshIndicator), findsNothing);
        await tester.drag(find.byType(CustomScrollView), const Offset(0, 400));
        await tester.pump(const Duration(milliseconds: 700));
        await tester.pump();
        expect(repository.listCalls, calls + 1);
        expect(find.text('@$peer'), findsOneWidget);
        final titleSpinner = find.descendant(
          of: find.byType(AppBar),
          matching: find.byType(StitchProgressIndicator),
        );
        expect(titleSpinner, findsNothing);
        pending.complete(previousPage);
        repository.pendingConversations = null;
        await tester.pumpAndSettle();
        expect(titleSpinner, findsNothing);

        await tester.tap(find.text('@$peer'));
        expect(
          opened?.value,
          bucket == ChatBucket.accepted
              ? 'external-conversation'
              : 'external-request',
        );
        if (bucket == ChatBucket.accepted) {
          expect(find.byTooltip('Direct messages'), findsOneWidget);
          await tester.tap(find.byTooltip('New message'));
          expect(newMessage, 1);
          await tester.tap(find.byTooltip('Requests'));
          expect(requests, 1);
          expect(find.text('3'), findsOneWidget);
        } else {
          expect(find.byTooltip('Direct messages'), findsNothing);
          expect(find.byIcon(CraftskyIcons.settings), findsNothing);
        }
        final container = ProviderScope.containerOf(
          tester.element(find.byType(MessagingInboxPage)),
        );
        final controller = container
            .read(chatInboxControllerProvider(account, bucket))
            .requireValue;
        repository.empty = true;
        await controller.refresh();
        await tester.pumpAndSettle();
        final emptyMessage = bucket == ChatBucket.accepted
            ? 'No messages yet.'
            : 'No message requests.';
        expect(find.text(emptyMessage), findsOneWidget);
        expect(
          find.descendant(
            of: find.byType(CraftskyEmptyState),
            matching: find.byIcon(
              bucket == ChatBucket.request
                  ? CraftskyIcons.email
                  : CraftskyIcons.comment,
            ),
          ),
          findsOneWidget,
        );

        final backgroundPage = Completer<ChatConversationPage>();
        repository.pendingConversations = backgroundPage;
        final backgroundRefresh = controller.refresh();
        await tester.pump();
        expect(find.text(emptyMessage), findsOneWidget);
        expect(titleSpinner, findsNothing);
        expect(find.byType(StitchProgressIndicator), findsNothing);
        repository
          ..empty = false
          ..pendingConversations = null;
        backgroundPage.complete(firstPage);
        await backgroundRefresh;
        await tester.pumpAndSettle();
        expect(find.text(emptyMessage), findsNothing);
        expect(find.text('@$peer'), findsOneWidget);
        expect(titleSpinner, findsNothing);

        await container
            .read(sessionRegistryProvider.notifier)
            .upsertAndActivate(
              token: 'bob-token',
              did: 'did:plc:bob',
              handle: 'bob.test',
            );
        await tester.pumpAndSettle();
        expect(find.text('request.external.test'), findsNothing);
        expect(find.text('external.test'), findsNothing);
        expect(find.text('CraftSky peer'), findsNothing);
        expect(tester.takeException(), isNull);
      },
    );
  }
}

class _Repository implements ChatRepository {
  Completer<ChatConversationPage>? pendingConversations;
  bool empty = false;
  int listCalls = 0;
  int heartbeats = 0;
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    heartbeats++;
    return ChatSyncSnapshot.fromMap({'revision': heartbeats, 'state': 'ready'});
  }

  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap(const {
    'generation': 2,
    'enabled': true,
    'state': 'ready',
    'pushEnabled': true,
    'capabilities': {
      'canRead': true,
      'canSend': true,
      'canReport': true,
      'canSetPolicy': true,
    },
  });
  @override
  Future<ChatUnreadCounts> unreadCounts() async =>
      ChatUnreadCounts.fromMap(const {'accepted': 100, 'requests': 3});
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    listCalls++;
    if (pendingConversations case final pending?) return await pending.future;
    if (empty) {
      return ChatConversationPage.fromMap(const {
        'items': <Map<String, Object?>>[],
      });
    }
    return ChatConversationPage.fromMap({
      'items': [
        {
          'id': bucket == ChatBucket.accepted
              ? 'external-conversation'
              : 'external-request',
          'revision': 'rev-1',
          'status': bucket.name,
          'muted': false,
          'unreadCount': 1,
          'canSend': bucket == ChatBucket.accepted,
          'lastMessage': const {
            'id': 'preview-message',
            'revision': 'r',
            'kind': 'text',
            'text': 'Conversation preview',
            'senderDid': 'did:plc:external',
            'sentAt': '2026-10-21T12:00:00Z',
          },
          'peer': {
            'did': 'did:plc:external',
            'displayName': 'CraftSky peer',
            'handle': bucket == ChatBucket.accepted
                ? 'external.test'
                : 'request.external.test',
          },
        },
      ],
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Storage implements SessionRegistryStorage {
  new(this.value);
  SessionRegistry value;
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry value) async {
    this.value = value;
  }
}
