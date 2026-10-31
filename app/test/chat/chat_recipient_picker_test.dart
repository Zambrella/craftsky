import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/chat/widgets/chat_recipient_picker.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/sharing/copy_link_menu_item.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/recording_error_reporter.dart';

void main() {
  for (final sharing in [false, true]) {
    for (final outcome in [
      'recent',
      'search',
      'unavailable',
      'owner changed',
      'dismissed',
      'inbox failed',
      'fullscreen',
      'initial recipient',
      'large text',
      'availability failed',
      'start failed',
      'held start',
    ]) {
      testWidgets('PICKER-001 shared picker sharing=$sharing: $outcome', (
        tester,
      ) async {
        FlutterSecureStorage.setMockInitialValues({});
        if (outcome == 'large text') {
          await tester.binding.setSurfaceSize(const Size(320, 568));
          addTearDown(() => tester.binding.setSurfaceSize(null));
        }
        final account = AccountKey('did:plc:owner');
        final repository = _Chat()
          ..available = outcome != 'unavailable'
          ..failInbox = outcome == 'inbox failed';
        if (outcome == 'owner changed' ||
            outcome == 'dismissed' ||
            outcome == 'availability failed' ||
            outcome == 'held start' ||
            outcome == 'start failed') {
          repository.pending = Completer<ChatAvailability>();
        }
        if (outcome == 'held start' || outcome == 'start failed') {
          repository.pendingStart = Completer<ChatConversation>();
        }
        final store = ChatDraftStore(const FlutterSecureStorage());
        await store.initialize({account});
        for (final id in ['existing', 'new-convo']) {
          await store.save(
            store.lease(account),
            ChatConversationId.parse(id),
            const ChatDraft.empty().edit('Keep existing draft'),
          );
        }
        final search = _Search();
        final messenger = RecordingMessenger();
        final reporter = RecordingErrorReporter();
        final container = ProviderContainer.test(
          overrides: [
            errorReporterProvider.overrideWithValue(reporter),
            secureSessionRegistryStorageProvider.overrideWithValue(_Storage()),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
            chatDraftStoreProvider.overrideWithValue(store),
            profileRepositoryProvider.overrideWithValue(
              FakeProfileRepository(
                onFetch: (did) async => Profile(
                  did: did,
                  handle: 'recent.test',
                  displayName: 'Recent peer',
                  crafts: const [],
                ),
              ),
            ),
            chatRecipientControllerProvider(account).overrideWith((ref) async {
              final controller = ChatRecipientController(
                account: account,
                repository: repository,
                searchRepository: search,
                operation: ActiveAccountOperation.fromLease(
                  ref,
                  ref.read(sessionRegistryProvider).value?.activeLease,
                ),
              );
              ref.onDispose(controller.dispose);
              return controller;
            }),
          ],
        );
        await container.read(sessionRegistryProvider.future);
        ChatConversationId? opened;
        await tester.pumpWidget(
          UncontrolledProviderScope(
            container: container,
            child: MaterialApp(
              theme: AppTheme.lightThemeData,
              localizationsDelegates: appLocalizationsDelegates,
              supportedLocales: AppLocalizations.supportedLocales,
              builder: (context, child) => MessengerScope(
                messenger: messenger,
                child: outcome == 'large text'
                    ? MediaQuery(
                        data: MediaQuery.of(context).copyWith(
                          textScaler: const TextScaler.linear(2),
                          viewInsets: const EdgeInsets.only(bottom: 250),
                        ),
                        child: child!,
                      )
                    : child!,
              ),
              home: outcome == 'fullscreen'
                  ? Scaffold(
                      body: Builder(
                        builder: (context) => TextButton(
                          onPressed: () => showSendViaChat(
                            context,
                            'https://example.test/link',
                          ),
                          child: const Text('Open picker'),
                        ),
                      ),
                    )
                  : Scaffold(
                      body: ChatRecipientPicker(
                        account: account,
                        initialRecipient: outcome == 'initial recipient'
                            ? Did.parse('did:plc:recent')
                            : null,
                        prefillLink: sharing
                            ? 'https://example.test/exact?x=1'
                            : null,
                        onSelected: (id) => opened = id,
                      ),
                    ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        if (outcome == 'fullscreen') {
          await tester.tap(find.text('Open picker'));
          await tester.pumpAndSettle();
          final sheetContext = tester.element(find.byType(ChatRecipientPicker));
          expect(
            ModalRoute.of(sheetContext),
            isA<FullscreenModalRoute<ChatConversationId>>(),
          );
          expect(find.byType(BottomSheet), findsNothing);
          final scaffold = find.descendant(
            of: find.byType(ChatRecipientPicker),
            matching: find.byType(Scaffold),
          );
          expect(
            tester.getSize(scaffold).height,
            tester.view.physicalSize.height / tester.view.devicePixelRatio,
          );
          expect(find.widgetWithText(AppBar, 'Send via chat'), findsOneWidget);
          await tester.tap(find.byType(CloseButton));
          await tester.pumpAndSettle();
          expect(find.text('Open picker'), findsOneWidget);
          expect(repository.sends, 0);
          return;
        }
        expect(find.text('Recent conversations'), findsOneWidget);
        if (outcome == 'inbox failed') {
          expect(messenger.calls.single.$1, 'error');
          expect(repository.sends, 0);
          await tester.pumpWidget(const SizedBox.shrink());
          return;
        }
        if (outcome == 'large text') {
          await tester.scrollUntilVisible(
            find.text('Recent peer'),
            80,
            scrollable: find
                .descendant(
                  of: find.byType(ListView),
                  matching: find.byType(Scrollable),
                )
                .first,
          );
        }
        expect(find.text('Recent peer'), findsOneWidget);
        final field = tester.widget<TextField>(find.byType(TextField));
        expect(field.autocorrect, false);
        final searching = [
          'search',
          'held start',
          'start failed',
        ].contains(outcome);
        if (searching) {
          await tester.enterText(find.byType(TextField), 'Search');
          await tester.pump(const Duration(milliseconds: 350));
          await tester.pumpAndSettle();
          expect(search.queries, ['Search']);
          expect(find.text('Recent conversations'), findsNothing);
          await tester.tap(find.text('Search peer'));
        } else if (outcome != 'initial recipient') {
          expect(reporter.errors, isEmpty);
          await tester.ensureVisible(find.text('Recent peer'));
          await tester.pumpAndSettle();
          await tester.tap(find.text('Recent peer'));
        }
        await tester.pump();
        if (repository.pending != null) {
          expect(repository.checks, 1);
          expect(repository.starts, 0);
          await tester.tap(
            find.text(searching ? 'Search peer' : 'Recent peer'),
            warnIfMissed: false,
          );
          expect(repository.checks, 1);
        }
        if (outcome == 'owner changed') {
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'other',
                did: 'did:plc:other',
                handle: 'other.test',
              );
          repository.pending!.complete(
            ChatAvailability.fromMap(const {
              'canChat': true,
              'conversationId': 'existing',
            }),
          );
        } else if (outcome == 'dismissed') {
          await tester.pumpWidget(const SizedBox.shrink());
          repository.pending!.complete(
            ChatAvailability.fromMap(const {
              'canChat': true,
              'conversationId': 'existing',
            }),
          );
        } else if (outcome == 'availability failed') {
          repository.pending!.completeError(
            StateError('synthetic availability failure'),
          );
        } else if (outcome == 'start failed' || outcome == 'held start') {
          repository.pending!.complete(
            ChatAvailability.fromMap(const {'canChat': true}),
          );
          await tester.pump();
          expect(repository.starts, 1);
          await tester.tap(find.text('Search peer'), warnIfMissed: false);
          expect(repository.starts, 1);
          if (outcome == 'start failed') {
            repository.pendingStart!.completeError(
              StateError('synthetic start failure'),
            );
          } else {
            repository.pendingStart!.complete(
              await repository.conversation(
                ChatConversationId.parse('new-convo'),
              ),
            );
          }
        }
        await tester.pumpAndSettle();
        if (outcome == 'unavailable' ||
            outcome == 'owner changed' ||
            outcome == 'dismissed' ||
            outcome.endsWith('failed')) {
          expect(opened, isNull);
          if (outcome == 'availability failed' || outcome == 'start failed') {
            expect(messenger.calls.single.$1, 'error');
          }
        } else {
          expect(reporter.errors, isEmpty);
          expect(opened?.value, searching ? 'new-convo' : 'existing');
          expect(
            (await store.load(store.lease(account), opened!)).composerText,
            sharing
                ? 'Keep existing draft\n\nhttps://example.test/exact?x=1'
                : 'Keep existing draft',
          );
        }
        expect(repository.sends, 0);
        if (outcome == 'unavailable') expect(messenger.calls.single.$1, 'info');
        await tester.pumpWidget(const SizedBox.shrink());
      });
    }
  }
  testWidgets(
    'SHARE-003 every shared link group contains chat and copy actions',
    (tester) async {
      late BuildContext context;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (value) {
              context = value;
              return const SizedBox();
            },
          ),
        ),
      );
      final items = linkSharingMenuItems(
        context,
        'https://app.craftsky.social/example',
      );
      expect(items.map((item) => item.text), ['Copy link', 'Send via chat']);
    },
  );
}

class _Storage implements SessionRegistryStorage {
  SessionRegistry value = SessionRegistry.empty().upsertAndActivate(
    token: 'synthetic',
    did: 'did:plc:owner',
    handle: 'owner.test',
  );
  @override
  Future<SessionRegistry> read() async => value;
  @override
  Future<void> write(SessionRegistry value) async {
    this.value = value;
  }
}

class _Search implements SearchRepository {
  final queries = <String>[];
  @override
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) async {
    queries.add(q);
    return ProfileSearchPage(
      items: [
        ProfileSearchResult(
          did: 'did:plc:search',
          handle: 'search.test',
          displayName: 'Search peer',
          isCraftskyProfile: true,
          viewerIsFollowing: false,
        ),
      ],
    );
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Chat implements ChatRepository {
  Completer<ChatConversation>? pendingStart;
  int checks = 0;
  int starts = 0;
  bool failInbox = false;
  bool available = true;
  int sends = 0;
  Completer<ChatAvailability>? pending;
  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap(const {
    'generation': 1,
    'enabled': true,
    'pushEnabled': true,
    'state': 'ready',
    'capabilities': {
      'canRead': true,
      'canSend': true,
      'canReport': true,
      'canSetPolicy': true,
    },
  });
  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    if (failInbox) throw StateError('synthetic inbox failure');
    return ChatConversationPage(
      items: [await conversation(ChatConversationId.parse('existing'))],
    );
  }

  @override
  Future<ChatAvailability> availability(Did recipient) async {
    checks++;
    return await (pending?.future ??
        Future.value(
          ChatAvailability.fromMap({
            'canChat': available,
            if (recipient.value == 'did:plc:recent')
              'conversationId': 'existing',
          }),
        ));
  }

  @override
  Future<ChatConversation> start(Did recipient) async {
    starts++;
    return await (pendingStart?.future ??
        conversation(ChatConversationId.parse('new-convo')));
  }

  @override
  Future<ChatConversation> conversation(ChatConversationId id) async =>
      ChatConversation.fromMap({
        'id': id.value,
        'revision': 'r',
        'status': 'accepted',
        'muted': false,
        'unreadCount': 0,
        'canSend': true,
        'peer': const {
          'did': 'did:plc:recent',
          'handle': 'recent.test',
          'displayName': 'Recent peer',
        },
      });
  @override
  Future<ChatConversationDetails> details(ChatConversationId id) async =>
      ChatConversationDetails(
        conversation: await conversation(id),
        messages: ChatMessagePage(items: const []),
      );
  @override
  Future<ChatSendResult> send(
    ChatConversationId id,
    ChatOperationId operation,
    String text,
  ) async {
    sends++;
    throw StateError('Must not send');
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
