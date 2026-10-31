import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/chat_conversation_page.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_conversation_controller.dart';
import 'package:craftsky_app/chat/services/chat_preview_resolver.dart';
import 'package:craftsky_app/chat/widgets/chat_time_header.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_chat_ui/flutter_chat_ui.dart' as ui;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';
import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/post_fixtures.dart';

void main() {
  for (final outcome in [
    'confirmed',
    'cancel',
    'failed',
    'message changed',
    'owner changed',
  ]) {
    testWidgets('DEL-003 delete confirmation and feedback: $outcome', (
      tester,
    ) async {
      final pending = Completer<void>();
      final repository = _Repository()
        ..accepted = true
        ..seedMessage = true
        ..seedSender = 'did:plc:owner'
        ..pendingDelete = pending.future;
      final view = await _pumpRequest(tester, repository, onClose: () {});
      view.controller.edit('Unsent draft');
      await tester.pump();
      await tester.longPress(find.text('private-peer'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete for me'));
      await tester.pumpAndSettle();
      expect(find.text('Delete message for me?'), findsOneWidget);
      expect(find.textContaining('other person will still'), findsOneWidget);
      expect(repository.deletes, 0);
      if (outcome == 'message changed') {
        repository.seedText = 'Changed message';
        await view.controller.refresh();
        await tester.pumpAndSettle();
      } else if (outcome == 'owner changed') {
        final container = ProviderScope.containerOf(
          tester.element(find.byType(MessagingConversationPage)),
        );
        await container
            .read(sessionRegistryProvider.notifier)
            .upsertAndActivate(
              token: 'other',
              did: 'did:plc:other',
              handle: 'other.test',
            );
        await tester.pumpAndSettle();
      }
      final action = outcome == 'cancel'
          ? find.text('Cancel')
          : find.widgetWithText(ChunkyButton, 'Delete for me');
      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 600));
      if (['cancel', 'message changed', 'owner changed'].contains(outcome)) {
        expect(repository.deletes, 0);
        expect(
          view.messenger.calls.where((call) => call.$1 == 'info'),
          isEmpty,
        );
      } else {
        expect(repository.deletes, 1);
        expect(find.text('private-peer'), findsOneWidget);
        expect(_sendButton(tester).onPressed, isNull);
        if (outcome == 'failed') {
          pending.completeError(const ApiNetworkError('offline'));
        } else {
          pending.complete();
        }
        await tester.pumpAndSettle();
        expect(
          find.text('private-peer'),
          outcome == 'confirmed' ? findsNothing : findsOneWidget,
        );
        expect(
          view.messenger.calls.single.$1,
          outcome == 'confirmed' ? 'info' : 'error',
        );
        expect(
          tester.widget<TextField>(find.byType(TextField)).controller!.text,
          'Unsent draft',
        );
      }
      expect(repository.leaves, 0);
      expect(repository.sentText, isNull);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('UT-013 conversation blocks peer after explicit confirmation', (
    tester,
  ) async {
    final repository = _Repository()..accepted = true;
    var blocks = 0;
    final profiles = FakeProfileRepository(
      onFetch: (did) async =>
          Profile(did: did, handle: 'peer.test', crafts: const []),
      onBlock: (did, operationKey) async {
        expect(did, 'did:plc:peer');
        expect(operationKey, isNotEmpty);
        blocks++;
        return const ProfileRelationship(blocking: true);
      },
    );
    var closes = 0;
    final view = await _pumpRequest(
      tester,
      repository,
      profiles: profiles,
      onClose: () => closes++,
    );
    view.controller.edit('Unsent draft');
    await tester.pump();
    await tester.tap(find.byTooltip('Conversation actions'));
    await tester.pumpAndSettle();
    expect(
      tester.widget<Text>(find.text('Block account')).style?.color,
      AppTheme.lightThemeData.extension<SemanticColorsTheme>()!.error,
    );
    await tester.tap(find.text('Block account'));
    await tester.pumpAndSettle();
    expect(find.text('Block this account?'), findsOneWidget);
    expect(find.textContaining('public on the AT Protocol'), findsOneWidget);
    expect(blocks, 0);
    final confirm = find.widgetWithText(ChunkyButton, 'Block account');
    await tester.ensureVisible(confirm);
    await tester.tap(confirm);
    await tester.pumpAndSettle();
    expect(blocks, 1);
    expect(view.messenger.calls.single.$2, 'Account blocked.');
    expect(repository.leaves, 0);
    expect(repository.sentText, isNull);
    expect(closes, 0);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      'Unsent draft',
    );
    expect(
      tester
          .widget<ChunkyIconButton>(
            find.byWidgetPredicate(
              (widget) =>
                  widget is ChunkyIconButton &&
                  widget.tooltip == 'Send message',
            ),
          )
          .onPressed,
      isNull,
    );
    expect(tester.takeException(), isNull);
  });
  for (final accepted in [false, true]) {
    for (final fails in [false, true]) {
      testWidgets(
        'UT-013 pending block guards actions accepted=$accepted fails=$fails',
        (tester) async {
          final pending = Completer<ProfileRelationship>();
          final repository = _Repository()..accepted = accepted;
          final profiles = FakeProfileRepository(
            onFetch: (did) async =>
                Profile(did: did, handle: 'peer.test', crafts: const []),
            onBlock: (_, _) => pending.future,
          );
          final view = await _pumpRequest(
            tester,
            repository,
            profiles: profiles,
            onClose: () {},
          );
          await tester.tap(find.byTooltip('Conversation actions'));
          await tester.pumpAndSettle();
          await tester.tap(find.text('Block account'));
          await tester.pumpAndSettle();
          final confirm = find.widgetWithText(ChunkyButton, 'Block account');
          await tester.ensureVisible(confirm);
          await tester.tap(confirm);
          await tester.pump();
          await tester.pump(const Duration(milliseconds: 600));
          expect(profiles.blockOperationKeys, hasLength(1));
          if (accepted) {
            expect(_sendButton(tester).onPressed, isNull);
          } else {
            expect(
              tester
                  .widget<ChunkyButton>(
                    find.widgetWithText(ChunkyButton, 'Accept request'),
                  )
                  .onPressed,
              isNull,
            );
          }
          await tester.tap(
            find.byTooltip('Conversation actions'),
            warnIfMissed: false,
          );
          await tester.pump();
          expect(find.text('Block account'), findsNothing);
          expect(profiles.blockOperationKeys, hasLength(1));
          if (fails) {
            pending.completeError(StateError('synthetic-block-failure'));
          } else {
            pending.complete(const ProfileRelationship(blocking: true));
          }
          await tester.pumpAndSettle();
          expect(repository.leaves, 0);
          expect(repository.acceptCalls, 0);
          expect(profiles.blockOperationKeys, hasLength(1));
          if (fails) {
            expect(view.messenger.calls.single.$1, 'error');
            await tester.tap(find.byTooltip('Conversation actions'));
            await tester.pumpAndSettle();
            expect(find.text('Block account'), findsOneWidget);
          } else if (!accepted) {
            expect(
              tester
                  .widget<ChunkyButton>(
                    find.widgetWithText(ChunkyButton, 'Accept request'),
                  )
                  .onPressed,
              isNull,
            );
            expect(
              tester
                  .widget<ChunkyButton>(
                    find.widgetWithText(ChunkyButton, 'Ignore'),
                  )
                  .onPressed,
              isNotNull,
            );
          }
          expect(tester.takeException(), isNull);
        },
      );
    }
  }
  for (final outcome in [
    'cancel',
    'owner changed',
    'reactivated',
    'capability loss',
  ]) {
    testWidgets('UT-013 block confirmation is guarded: $outcome', (
      tester,
    ) async {
      final repository = _Repository();
      final profiles = FakeProfileRepository(
        onFetch: (did) async =>
            Profile(did: did, handle: 'peer.test', crafts: const []),
        onBlock: (_, _) async => const ProfileRelationship(blocking: true),
      );
      final view = await _pumpRequest(
        tester,
        repository,
        profiles: profiles,
        onClose: () {},
      );
      final container = ProviderScope.containerOf(
        tester.element(find.byType(MessagingConversationPage)),
      );
      await tester.tap(find.byTooltip('Conversation actions'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Block account'));
      await tester.pumpAndSettle();
      if (outcome == 'owner changed' || outcome == 'reactivated') {
        await container
            .read(sessionRegistryProvider.notifier)
            .upsertAndActivate(
              token: 'other',
              did: 'did:plc:other',
              handle: 'other.test',
            );
        if (outcome == 'reactivated') {
          await container
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'synthetic',
                did: 'did:plc:owner',
                handle: 'owner.test',
              );
        }
        await tester.pumpAndSettle();
      } else if (outcome == 'capability loss') {
        repository.connectionState = 'unavailable';
        container.invalidate(chatStatusProvider(AccountKey('did:plc:owner')));
        await tester.pumpAndSettle();
      }
      final action = outcome == 'cancel'
          ? find.text('Cancel')
          : find.widgetWithText(ChunkyButton, 'Block account');
      await tester.ensureVisible(action);
      await tester.tap(action);
      await tester.pumpAndSettle();
      expect(profiles.blockOperationKeys, isEmpty);
      expect(repository.leaves, 0);
      expect(view.messenger.calls.where((call) => call.$1 == 'info'), isEmpty);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('UT-010 details title shows name and handle at large text', (
    tester,
  ) async {
    final repository = _Repository()
      ..displayName = 'CraftSky peer with a long display name';
    await _pumpRequest(tester, repository, onClose: () {});
    final title = find.descendant(
      of: find.byType(AppBar),
      matching: find.text(repository.displayName!),
    );
    final handle = find.descendant(
      of: find.byType(AppBar),
      matching: find.text('@peer.test'),
    );
    expect(title, findsOneWidget);
    expect(handle, findsOneWidget);
    expect(
      tester.getTopLeft(handle).dy,
      greaterThan(tester.getTopLeft(title).dy),
    );
    expect(
      tester.getBottomLeft(handle).dy,
      lessThanOrEqualTo(tester.getBottomLeft(find.byType(AppBar)).dy),
    );
    expect(tester.takeException(), isNull);
  });
  testWidgets('AT-009 Ignore leaves a request without accepting it', (
    tester,
  ) async {
    final pending = Completer<void>();
    final repository = _Repository()..pendingLeave = pending.future;
    var closes = 0;
    await _pumpRequest(tester, repository, onClose: () => closes++);
    expect(find.text('Ignore'), findsOneWidget);
    expect(
      find.text(
        'Accept this shared message request before sending a reply. '
        'Reading it does not accept it.',
      ),
      findsNothing,
    );
    expect(find.byType(TextField), findsNothing);
    await tester.tap(find.text('Ignore'));
    await tester.pump();
    await tester.tap(find.text('Ignore'), warnIfMissed: false);
    await tester.tap(find.text('Accept request'), warnIfMissed: false);
    await tester.pump();
    expect(repository.leaves, 1);
    expect(repository.acceptCalls, 0);
    expect(repository.sentText, isNull);
    expect(closes, 0);
    expect(find.byType(Dialog), findsNothing);
    for (final button in tester.widgetList<ChunkyButton>(
      find.byType(ChunkyButton),
    )) {
      expect(button.onPressed, isNull);
    }
    pending.complete();
    await tester.pumpAndSettle();
    expect(closes, 1);
    expect(repository.acceptCalls, 0);
    expect(tester.takeException(), isNull);
  });
  for (final outcome in [
    'failure',
    'owner changed',
    'covered',
    'disposed',
    'capability loss',
  ]) {
    testWidgets('AT-009 Ignore completion is guarded: $outcome', (
      tester,
    ) async {
      final pending = Completer<void>();
      final repository = _Repository()..pendingLeave = pending.future;
      var closes = 0;
      var current = true;
      final view = await _pumpRequest(
        tester,
        repository,
        onClose: () => closes++,
        isCurrent: () => current,
      );
      await tester.tap(find.text('Ignore'));
      await tester.pump();
      expect(repository.leaves, 1);
      switch (outcome) {
        case 'owner changed':
          current = false;
          await ProviderScope.containerOf(
                tester.element(find.byType(MessagingConversationPage)),
              )
              .read(sessionRegistryProvider.notifier)
              .upsertAndActivate(
                token: 'other',
                did: 'did:plc:other',
                handle: 'other.test',
              );
          await tester.pump();
        case 'covered':
          Navigator.of(tester.element(find.text('Ignore'))).push<void>(
            MaterialPageRoute(
              builder: (_) => const Scaffold(body: Text('Covered')),
            ),
          );
          await tester.pumpAndSettle();
        case 'disposed':
          await tester.pumpWidget(const SizedBox.shrink());
        case 'capability loss':
          repository.connectionState = 'unavailable';
          ProviderScope.containerOf(
            tester.element(find.byType(MessagingConversationPage)),
          ).invalidate(chatStatusProvider(AccountKey('did:plc:owner')));
          await tester.pump();
          await tester.pump();
        case _:
          break;
      }
      if (outcome == 'failure') {
        pending.completeError(const ApiNetworkError('offline'));
      } else {
        pending.complete();
      }
      await tester.pumpAndSettle();
      expect(closes, 0);
      expect(repository.acceptCalls, 0);
      expect(repository.sentText, isNull);
      if (outcome == 'failure') {
        expect(view.controller.left, false);
        expect(find.text('Ignore'), findsOneWidget);
        expect(find.text('Accept request'), findsOneWidget);
        expect(view.messenger.calls.single.$1, 'error');
        expect(
          view.messenger.calls.single.$2,
          'Messaging is temporarily unavailable. Try again.',
        );
      }
      expect(tester.takeException(), isNull);
    });
  }
  for (final outcome in ['owner changed', 'covered', 'disposed']) {
    testWidgets('UT-013 block completion is guarded: $outcome', (
      tester,
    ) async {
      final pending = Completer<ProfileRelationship>();
      final repository = _Repository();
      final profiles = FakeProfileRepository(
        onFetch: (did) async =>
            Profile(did: did, handle: 'peer.test', crafts: const []),
        onBlock: (_, _) => pending.future,
      );
      final view = await _pumpRequest(
        tester,
        repository,
        profiles: profiles,
        onClose: () {},
      );
      final context = tester.element(find.byType(MessagingConversationPage));
      final container = ProviderScope.containerOf(context);
      await tester.tap(find.byTooltip('Conversation actions'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Block account'));
      await tester.pumpAndSettle();
      final confirm = find.widgetWithText(ChunkyButton, 'Block account');
      await tester.ensureVisible(confirm);
      await tester.tap(confirm);
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 600));
      expect(profiles.blockOperationKeys, hasLength(1));
      if (outcome == 'owner changed') {
        await container
            .read(sessionRegistryProvider.notifier)
            .upsertAndActivate(
              token: 'other',
              did: 'did:plc:other',
              handle: 'other.test',
            );
      } else if (outcome == 'covered') {
        Navigator.of(context).push<void>(
          MaterialPageRoute(
            builder: (_) => const Scaffold(body: Text('Covered')),
          ),
        );
      } else {
        await tester.pumpWidget(const SizedBox.shrink());
      }
      await tester.pump();
      pending.complete(const ProfileRelationship(blocking: true));
      await tester.pumpAndSettle();
      expect(view.messenger.calls.where((call) => call.$1 == 'info'), isEmpty);
      expect(repository.leaves, 0);
      expect(repository.acceptCalls, 0);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets('UT-010 large-text uncertain composer fits above the keyboard', (
    tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(320, 568));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation');
    final repository = _Repository()..accepted = true;
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    await store.save(
      store.lease(account),
      id,
      const ChatDraft.empty()
          .edit('first\nsecond\nthird\nfourth\nfifth')
          .beginSend(
            ChatOperationId.parse('00000000-0000-4000-8000-000000000123'),
          )
          .markUncertain(),
    );
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: repository,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await controller.load();
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'token',
      did: account.did.value,
      handle: 'owner.test',
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountChatRepositoryProvider(account)
              .overrideWith((ref) async => repository),
          chatConversationControllerProvider(
            account,
            id,
          ).overrideWith((ref) async => controller),
        ],
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: const TextScaler.linear(2),
              viewInsets: const EdgeInsets.only(bottom: 250),
            ),
            child: MessengerScope(
              messenger: RecordingMessenger(),
              child: child!,
            ),
          ),
          home: MessagingConversationPage(
            account: account,
            conversationId: id,
            onClose: () {},
            onReport: (_) {},
            onAuthorize: (_) {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.text('Check status'));
    await tester.pumpAndSettle();
    expect(find.text('Check status').hitTestable(), findsOneWidget);
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      contains('fifth'),
    );
  });

  testWidgets(
    'IT-016 request and composer perform no preview fetch; accept enables '
    'one bubble card',
    (tester) async {
      FlutterSecureStorage.setMockInitialValues({});
      final messenger = RecordingMessenger();
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation-1');
      final repository = _Repository()
        ..seedMessage = true
        ..seedText = 'https://app.craftsky.social/posts/did:plc:alice/3lf2abc';
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final operation = ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      );
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: repository,
        store: store,
        draftLease: store.lease(account),
        operation: operation,
      );
      addTearDown(controller.dispose);
      await controller.load();
      var fetches = 0;
      final resolver = ChatPreviewResolver(
        account: account,
        operation: operation,
        posts: FakePostRepository(
          onFetch: (did, rkey) async {
            fetches++;
            return postFixture(text: 'Permitted preview excerpt');
          },
        ),
        profiles: FakeProfileRepository(),
        events: _Events(),
      );
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic-token',
        did: account.did.value,
        handle: 'owner.test',
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
            chatConversationControllerProvider(
              account,
              id,
            ).overrideWith((ref) async => controller),
            accountChatPreviewResolverProvider(account)
                .overrideWith((ref) async => resolver),
          ],
          child: MaterialApp(
            builder: (context, child) =>
                MessengerScope(messenger: messenger, child: child!),
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MessagingConversationPage(
              account: account,
              conversationId: id,
              onClose: () {},
              onReport: (_) {},
              onAuthorize: (_) {},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(fetches, 0);
      expect(find.text('Permitted preview excerpt'), findsNothing);
      await tester.tap(find.text('Accept request'));
      await tester.pumpAndSettle();
      expect(fetches, 1);
      expect(find.text('Permitted preview excerpt'), findsOneWidget);
      await tester.enterText(
        find.byType(TextField),
        'https://app.craftsky.social/profiles/did:plc:later',
      );
      await tester.pumpAndSettle();
      expect(fetches, 1);
      expect(tester.takeException(), isNull);
      final editor = tester
          .widget<TextField>(find.byType(TextField))
          .controller;
      final container = ProviderScope.containerOf(
        tester.element(find.byType(MessagingConversationPage)),
      );
      repository.connectionState = 'unavailable';
      container.invalidate(chatStatusProvider(account));
      await tester.pumpAndSettle();
      expect(
        messenger.calls.any(
          (call) => call.$1 == 'warning' && call.$2.contains('out of date'),
        ),
        true,
      );
      expect(find.text('Permitted preview excerpt'), findsOneWidget);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller,
        same(editor),
      );
      expect(
        tester
            .widget<ChunkyIconButton>(
              find.byType(ChunkyIconButton),
            )
            .onPressed,
        isNull,
      );
      repository.connectionState = 'ready';
      container.invalidate(chatStatusProvider(account));
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller,
        same(editor),
      );
    },
  );

  testWidgets(
    'IT-007/UT-013 conversation offers shared mute, confirmed leave '
    'and message report',
    (tester) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation-1');
      final repository = _Repository()
        ..accepted = true
        ..seedMessage = true;
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: repository,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic-token',
        did: account.did.value,
        handle: 'owner.test',
      );
      ChatMessageId? reported;
      var closed = false;
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
            chatConversationControllerProvider(
              account,
              id,
            ).overrideWith((ref) async => controller),
          ],
          child: MaterialApp(
            builder: (context, child) =>
                MessengerScope(messenger: RecordingMessenger(), child: child!),
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MessagingConversationPage(
              account: account,
              conversationId: id,
              onClose: () => closed = true,
              onReport: (id) => reported = id,
              onAuthorize: (_) {},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField)).decoration?.hintText,
        'Message',
      );
      expect(
        tester.getCenter(find.byType(TextField)).dy,
        closeTo(tester.getCenter(find.byType(ChunkyIconButton)).dy, 4),
      );
      final sendButton = find.descendant(
        of: find.byType(ChunkyIconButton),
        matching: find.byType(ChunkyButton),
      );
      expect(tester.widget<ChunkyButton>(sendButton).onPressed, isNull);
      await tester.enterText(find.byType(TextField), 'Unsent draft');
      await tester.pump();
      expect(tester.widget<ChunkyButton>(sendButton).onPressed, isNotNull);
      expect(
        tester.widget<ChunkyButton>(sendButton).variant,
        ChunkyButtonVariant.primary,
      );
      final previousPage = await repository.history(id);
      final pendingHistory = Completer<ChatMessagePage>();
      repository.pendingHistory = pendingHistory;
      final refreshed = controller.refresh();
      await tester.pump();
      expect(find.text('private-peer'), findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(AppBar),
          matching: find.byType(StitchProgressIndicator),
        ),
        findsNothing,
      );
      pendingHistory.complete(previousPage);
      await refreshed;
      repository.pendingHistory = null;
      await tester.pumpAndSettle();
      expect(
        find.descendant(
          of: find.byType(AppBar),
          matching: find.byType(StitchProgressIndicator),
        ),
        findsNothing,
      );
      controller.edit('private-retained');
      final container = ProviderScope.containerOf(
        tester.element(find.byType(MessagingConversationPage)),
      );
      expect(
        await container
            .read(unsavedWorkGuardProvider)
            .confirmLeave(registry.activeLease!.session),
        true,
      );
      expect(
        (await store.load(store.lease(account), id)).text,
        'private-retained',
      );
      await tester.pump();
      expect(find.byTooltip('Report message'), findsNothing);
      await tester.longPress(find.text('private-peer'));
      await tester.pumpAndSettle();
      expect(reported, isNull);
      await tester.tap(find.text('Report message'));
      await tester.pumpAndSettle();
      expect(reported?.value, 'peer-message');
      await tester.tap(find.byTooltip('Conversation actions'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Mute conversation'));
      await tester.pumpAndSettle();
      expect(repository.muted, true);
      await tester.tap(find.byTooltip('Conversation actions'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Leave conversation'));
      await tester.pumpAndSettle();
      expect(find.textContaining('other apps'), findsOneWidget);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(repository.leaves, 0);
      await tester.tap(find.byTooltip('Conversation actions'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Leave conversation'));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(ChunkyButton, 'Leave'));
      await tester.pumpAndSettle();
      expect(repository.leaves, 1);
      expect(closed, true);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets('UT-009 Send again warns before repeating the earlier message', (
    tester,
  ) async {
    FlutterSecureStorage.setMockInitialValues({});
    final account = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation-1');
    final original = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    final repository = _Repository()..accepted = true;
    final store = ChatDraftStore(const FlutterSecureStorage());
    await store.initialize({account});
    await store.save(
      store.lease(account),
      id,
      const ChatDraft.empty()
          .edit('private-original')
          .beginSend(original)
          .markUncertain(),
    );
    final controller = ChatConversationController(
      account: account,
      id: id,
      repository: repository,
      store: store,
      draftLease: store.lease(account),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await controller.load();
    controller.edit('private-newer');
    final registry = SessionRegistry.empty().upsertAndActivate(
      token: 'synthetic-token',
      did: account.did.value,
      handle: 'owner.test',
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          secureSessionRegistryStorageProvider.overrideWithValue(
            _Storage(registry),
          ),
          accountChatRepositoryProvider(account)
              .overrideWith((ref) async => repository),
          chatConversationControllerProvider(
            account,
            id,
          ).overrideWith((ref) async => controller),
        ],
        child: MaterialApp(
          builder: (context, child) =>
              MessengerScope(messenger: RecordingMessenger(), child: child!),
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: MessagingConversationPage(
            account: account,
            conversationId: id,
            onClose: () {},
            onReport: (_) {},
            onAuthorize: (_) {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Send again'));
    await tester.pumpAndSettle();
    expect(find.textContaining('duplicate'), findsOneWidget);
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(repository.sentText, isNull);
    await tester.tap(find.text('Send again'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(ChunkyButton, 'Send again'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(repository.sentText, 'private-original');
    expect(repository.operation, isNot(original));
    expect(
      tester.widget<TextField>(find.byType(TextField)).controller!.text,
      'private-newer',
    );
    repository.result.complete(
      ChatSendResult.fromMap({
        'operationId': repository.operation!.value,
        'kind': 'send',
        'state': 'confirmed',
        'messageId': 'retried-message',
      }),
    );
    await tester.pumpAndSettle();
    expect(controller.draft.text, 'private-newer');
    expect(tester.takeException(), isNull);
  });
  for (final state in ['failed', 'uncertain']) {
    testWidgets('UT-009 $state send stops spinning and keeps recovery', (
      tester,
    ) async {
      final repository = _Repository()..accepted = true;
      final view = await _pumpRequest(tester, repository, onClose: () {});
      await tester.enterText(find.byType(TextField), 'Outgoing attempt');
      await tester.pump();
      await tester.tap(find.byTooltip('Send message'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.byType(ui.SimpleTextMessage), findsOneWidget);
      repository.result.complete(
        ChatSendResult.fromMap({
          'operationId': repository.operation!.value,
          'kind': 'send',
          'state': state,
        }),
      );
      await tester.pumpAndSettle();
      expect(find.byType(CircularProgressIndicator), findsNothing);
      final bubble = tester.widget<ui.SimpleTextMessage>(
        find.byType(ui.SimpleTextMessage),
      );
      expect(bubble.message.text, 'Outgoing attempt');
      expect(bubble.showTime, false);
      expect(find.text('Send again'), findsOneWidget);
      expect(find.text('Check status'), findsOneWidget);
      expect(view.controller.draft.text, 'Outgoing attempt');
      expect(view.controller.draft.pending!.state.name, state);
      expect(
        view.messenger.calls.any(
          (call) => call.$1 == (state == 'failed' ? 'error' : 'warning'),
        ),
        true,
      );
      expect(repository.sendCalls, 1);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets(
    'AT-003/AT-006 request requires accept and confirmed send keeps later text',
    (tester) async {
      FlutterSecureStorage.setMockInitialValues({});
      final account = AccountKey('did:plc:owner');
      final id = ChatConversationId.parse('conversation-1');
      final repository = _Repository();
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({account});
      final controller = ChatConversationController(
        account: account,
        id: id,
        repository: repository,
        store: store,
        draftLease: store.lease(account),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      await controller.load();
      final registry = SessionRegistry.empty().upsertAndActivate(
        token: 'synthetic-token',
        did: account.did.value,
        handle: 'owner.test',
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            secureSessionRegistryStorageProvider.overrideWithValue(
              _Storage(registry),
            ),
            accountChatRepositoryProvider(account)
                .overrideWith((ref) async => repository),
            chatConversationControllerProvider(
              account,
              id,
            ).overrideWith((ref) async => controller),
          ],
          child: MaterialApp(
            builder: (context, child) =>
                MessengerScope(messenger: RecordingMessenger(), child: child!),
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: MessagingConversationPage(
              account: account,
              conversationId: id,
              onClose: () {},
              onReport: (_) {},
              onAuthorize: (_) {},
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(TextField), findsNothing);
      expect(repository.acceptCalls, 0);
      expect(find.text('Ignore'), findsOneWidget);
      await tester.tap(find.text('Accept request'));
      await tester.pumpAndSettle();
      expect(repository.acceptCalls, 1);
      expect(find.text('Ignore'), findsNothing);
      expect(find.byType(TextField), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'private-original');
      await tester.pump();
      await tester.tap(find.byTooltip('Send message'));
      await tester.pump();
      expect(repository.sentText, 'private-original');
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        isEmpty,
      );
      await tester.pump(const Duration(milliseconds: 300));
      final pendingBubble = tester.widget<ui.SimpleTextMessage>(
        find.byType(ui.SimpleTextMessage),
      );
      expect(pendingBubble.message.text, 'private-original');
      expect(pendingBubble.showTime, isFalse);
      expect(pendingBubble.showStatus, isTrue);
      final localId = pendingBubble.message.id;
      expect(
        find.descendant(
          of: find.byType(ui.SimpleTextMessage),
          matching: find.byType(CircularProgressIndicator),
        ),
        findsWidgets,
      );
      await tester.enterText(find.byType(TextField), 'private-newer');
      repository.result.complete(
        ChatSendResult.fromMap({
          'operationId': repository.operation!.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'sent-message',
          'message': {
            'id': 'sent-message',
            'revision': 'rev-2',
            'kind': 'text',
            'text': 'private-original',
            'senderDid': account.did.value,
            'sentAt': '2026-10-19T12:01:00Z',
          },
        }),
      );
      await tester.pumpAndSettle();
      expect(find.text('private-original'), findsOneWidget);
      final confirmedBubble = tester.widget<ui.SimpleTextMessage>(
        find.byType(ui.SimpleTextMessage),
      );
      expect(confirmedBubble.message.id, localId);
      expect(confirmedBubble.showTime, isFalse);
      expect(find.byType(ChatTimeHeader), findsOneWidget);
      expect(confirmedBubble.showStatus, isFalse);
      expect(find.byType(CircularProgressIndicator), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        'private-newer',
      );
      expect(tester.takeException(), isNull);
    },
  );
}

ChunkyIconButton _sendButton(WidgetTester tester) =>
    tester.widget<ChunkyIconButton>(
      find.byWidgetPredicate(
        (widget) =>
            widget is ChunkyIconButton && widget.tooltip == 'Send message',
      ),
    );

Future<({ChatConversationController controller, RecordingMessenger messenger})>
_pumpRequest(
  WidgetTester tester,
  _Repository repository, {
  required VoidCallback onClose,
  bool Function()? isCurrent,
  FakeProfileRepository? profiles,
}) async {
  await tester.binding.setSurfaceSize(const Size(320, 568));
  addTearDown(() => tester.binding.setSurfaceSize(null));
  FlutterSecureStorage.setMockInitialValues({});
  final account = AccountKey('did:plc:owner');
  final id = ChatConversationId.parse('request');
  final store = ChatDraftStore(const FlutterSecureStorage());
  await store.initialize({account});
  final controller = ChatConversationController(
    account: account,
    id: id,
    repository: repository,
    store: store,
    draftLease: store.lease(account),
    operation: ActiveAccountOperation.withGuard(
      ownership: null,
      isCurrent: isCurrent ?? () => true,
    ),
  );
  addTearDown(controller.dispose);
  await controller.load();
  final messenger = RecordingMessenger();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        if (profiles != null)
          profileRepositoryProvider.overrideWithValue(profiles),
        pdsRecordOperationControllerProvider.overrideWithValue(
          PdsRecordOperationController(schedule: (_, _) {}),
        ),
        secureSessionRegistryStorageProvider.overrideWithValue(
          _Storage(
            SessionRegistry.empty().upsertAndActivate(
              token: 'synthetic',
              did: account.did.value,
              handle: 'owner.test',
            ),
          ),
        ),
        accountChatRepositoryProvider(account)
            .overrideWith((ref) async => repository),
        chatConversationControllerProvider(
          account,
          id,
        ).overrideWith((ref) async => controller),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            textScaler: const TextScaler.linear(2),
          ),
          child: MessengerScope(messenger: messenger, child: child!),
        ),
        home: MessagingConversationPage(
          account: account,
          conversationId: id,
          onClose: onClose,
          onReport: (_) {},
          onAuthorize: (_) {},
        ),
      ),
    ),
  );
  await tester.pumpAndSettle();
  return (controller: controller, messenger: messenger);
}

class _Repository implements ChatRepository {
  Future<void>? pendingDelete;
  int deletes = 0;
  String seedSender = 'did:plc:peer';
  @override
  Future<void> deleteMessage(
    ChatConversationId id,
    ChatMessageId message,
  ) async {
    expect(message.value, 'peer-message');
    deletes++;
    await pendingDelete;
  }

  Completer<ChatMessagePage>? pendingHistory;
  Future<void>? pendingLeave;
  String connectionState = 'ready';
  bool muted = false;
  int leaves = 0;
  bool seedMessage = false;
  String seedText = 'private-peer';
  @override
  Future<ChatConversation> setMuted(
    ChatConversationId id, {
    required bool muted,
  }) async {
    this.muted = muted;
    return await conversation(id);
  }

  @override
  Future<void> leave(ChatConversationId id) async {
    leaves++;
    await pendingLeave;
  }

  String? displayName;
  bool accepted = false;
  int acceptCalls = 0;
  String? sentText;
  int sendCalls = 0;
  ChatOperationId? operation;
  final result = Completer<ChatSendResult>();
  @override
  Future<ChatStatus> status() async => ChatStatus.fromMap({
    'generation': 2,
    'enabled': true,
    'state': connectionState,
    'lastSuccessAt': '2026-10-19T12:00:00Z',
    'pushEnabled': true,
    'capabilities': const {
      'canRead': true,
      'canSend': true,
      'canReport': true,
      'canSetPolicy': true,
    },
  });
  @override
  Future<ChatConversationDetails> details(ChatConversationId id) async =>
      ChatConversationDetails(
        conversation: await conversation(id),
        messages: await history(id),
      );

  @override
  Future<ChatConversation> conversation(ChatConversationId id) async =>
      ChatConversation.fromMap({
        'id': id.value,
        'revision': 'rev-1',
        'status': accepted ? 'accepted' : 'request',
        'muted': muted,
        'unreadCount': 1,
        'canSend': accepted,
        'peer': {
          'did': 'did:plc:peer',
          'handle': 'peer.test',
          'displayName': displayName,
        },
      });
  @override
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  }) async =>
      await (pendingHistory?.future ??
          Future.value(
            ChatMessagePage.fromMap({
              'items': [
                if (seedMessage)
                  {
                    'id': 'peer-message',
                    'revision': 'r',
                    'kind': 'text',
                    'text': seedText,
                    'senderDid': seedSender,
                    'sentAt': '2026-10-19T12:00:00Z',
                  },
              ],
            }),
          ));
  @override
  Future<ChatConversation> accept(ChatConversationId id) async {
    acceptCalls++;
    accepted = true;
    return await conversation(id);
  }

  @override
  Future<ChatConversation> updateRead(
    ChatConversationId id,
    ChatMessageId message,
  ) => conversation(id);
  @override
  Future<ChatSendResult> send(
    ChatConversationId id,
    ChatOperationId operation,
    String text,
  ) async {
    sendCalls++;
    this.operation = operation;
    sentText = text;
    return await result.future;
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

class _Events implements BusinessRepository {
  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
