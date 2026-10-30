import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/pages/new_message_page.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/recording_messenger.dart';

void main() {
  testWidgets('UT-010 unavailable recipient actions fit '
      'large text and keyboard', (tester) async {
    await tester.binding.setSurfaceSize(const Size(320, 568));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final chat = _Chat()..available = false;
    final controller = ChatRecipientController(
      account: AccountKey('did:plc:owner'),
      repository: chat,
      searchRepository: _Search(),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      MessengerScope(
        messenger: RecordingMessenger(),
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: const TextScaler.linear(2),
              viewInsets: const EdgeInsets.only(bottom: 250),
            ),
            child: child!,
          ),
          home: NewMessageView(
            controller: controller,
            initialRecipient: Did.parse('did:plc:abcdefghijklmnopqrstuvwxyz'),
            onOpen: (_) {},
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.text('Start conversation'));
    await tester.pumpAndSettle();
    expect(find.text('Start conversation').hitTestable(), findsOneWidget);
    expect(chat.starts, 0);
  });

  testWidgets(
    'IT-003 cold profile recipient and changed query stay read-only until '
    'Start',
    (tester) async {
      final chat = _Chat();
      final controller = ChatRecipientController(
        account: AccountKey('did:plc:owner'),
        repository: chat,
        searchRepository: _Search(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      Widget view(String did) => MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: NewMessageView(
          controller: controller,
          initialRecipient: Did.parse(did),
          onOpen: (_) {},
        ),
      );
      await tester.pumpWidget(view('did:plc:external'));
      await tester.pumpAndSettle();
      expect(controller.recipient, Did.parse('did:plc:external'));
      expect(chat.starts, 0);
      expect(controller.canStart, true);
      await tester.pumpWidget(view('did:plc:other'));
      await tester.pumpAndSettle();
      expect(controller.recipient, Did.parse('did:plc:other'));
      expect(chat.starts, 0);
    },
  );

  testWidgets(
    'IT-003 tapping a CraftSky search result opens its conversation',
    (tester) async {
      final chat = _Chat();
      final controller = ChatRecipientController(
        account: AccountKey('did:plc:owner'),
        repository: chat,
        searchRepository: _Search(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
      );
      addTearDown(controller.dispose);
      String? opened;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: NewMessageView(
            controller: controller,
            onOpen: (id) => opened = id.value,
          ),
        ),
      );
      await tester.enterText(find.byType(TextField), 'craft');
      await tester.pump();
      expect(find.byType(CraftskySkeleton), findsOneWidget);
      expect(find.byType(AccountRowSkeleton), findsWidgets);
      expect(find.byType(StitchProgressIndicator), findsNothing);
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      await tester.tap(find.text('peer.test'));
      await tester.pumpAndSettle();
      expect(find.text('external.test'), findsNothing);
      expect(find.text('Start conversation'), findsNothing);
      expect(opened, 'canonical');
      expect(chat.starts, 1);
      expect(tester.takeException(), isNull);
    },
  );
  testWidgets('IT-003 denied search result uses CraftSky snackbar feedback', (
    tester,
  ) async {
    final chat = _Chat()..available = false;
    final messenger = RecordingMessenger();
    final controller = ChatRecipientController(
      account: AccountKey('did:plc:owner'),
      repository: chat,
      searchRepository: _Search(),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
    );
    addTearDown(controller.dispose);
    await tester.pumpWidget(
      MessengerScope(
        messenger: messenger,
        child: MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: appLocalizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: NewMessageView(
            controller: controller,
            onOpen: (_) => fail('Denied recipient opened'),
          ),
        ),
      ),
    );
    await tester.enterText(find.byType(TextField), 'craft');
    await tester.pump(const Duration(milliseconds: 350));
    await tester.pumpAndSettle();
    await tester.tap(find.text('peer.test'));
    await tester.pumpAndSettle();
    const denied =
        'This person cannot receive a new message from you right now.';
    expect(messenger.calls.single.$1, 'info');
    expect(messenger.calls.single.$2, denied);
    expect(find.text(denied), findsNothing);
    expect(find.text('Start conversation'), findsNothing);
    expect(chat.starts, 0);
    await tester.tap(find.text('peer.test'));
    await tester.pumpAndSettle();
    expect(messenger.calls, hasLength(2));
    expect(messenger.calls.last.$2, denied);
    expect(tester.takeException(), isNull);
  });
  test(
    'IT-003 selection checks availability but only explicit start creates',
    () async {
      final chat = _Chat();
      final search = _Search();
      var current = true;
      final controller = ChatRecipientController(
        account: AccountKey('did:plc:owner'),
        repository: chat,
        searchRepository: search,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
      );
      addTearDown(controller.dispose);
      await controller.search('craft');
      expect(search.query, 'craft');
      expect(controller.items.map((item) => item.did.value), ['did:plc:peer']);
      await controller.select(controller.items.single);
      expect(chat.starts, 0);
      expect(controller.canStart, true);
      expect((await controller.start())?.value, 'canonical');
      expect(chat.starts, 1);
      chat.available = false;
      await controller.select(controller.items.single);
      expect(controller.canStart, false);
      expect(await controller.start(), isNull);
      expect(chat.starts, 1);
      chat.available = true;
      await controller.select(controller.items.single);
      current = false;
      expect(await controller.start(), isNull);
      expect(chat.starts, 1);
    },
  );

  for (final outcome in [
    'success',
    'owner changed',
    'availability failed',
    'start failed',
  ]) {
    testWidgets('IT-003 tap-to-open guards pending requests: $outcome', (
      tester,
    ) async {
      final availability = Completer<ChatAvailability>();
      final start = Completer<void>();
      final chat = _Chat()
        ..heldAvailability = availability.future
        ..heldStart = start.future;
      var current = true;
      final controller = ChatRecipientController(
        account: AccountKey('did:plc:owner'),
        repository: chat,
        searchRepository: _Search(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
      );
      addTearDown(controller.dispose);
      await controller.search('craft');
      final messenger = RecordingMessenger();
      var opens = 0;
      await tester.pumpWidget(
        MessengerScope(
          messenger: messenger,
          child: MaterialApp(
            theme: AppTheme.lightThemeData,
            localizationsDelegates: appLocalizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: NewMessageView(
              controller: controller,
              onOpen: (_) => opens++,
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('peer.test'));
      await tester.tap(find.text('peer.test'));
      await tester.pump();
      expect(chat.checks, 1);
      expect(chat.starts, 0);
      if (outcome == 'owner changed') current = false;
      if (outcome == 'availability failed') {
        availability.completeError(const ApiNetworkError('offline'));
      } else {
        availability.complete(
          ChatAvailability.fromMap(const {'canChat': true}),
        );
      }
      await tester.pump();
      final starts = outcome == 'success' || outcome == 'start failed';
      expect(chat.starts, starts ? 1 : 0);
      if (starts) {
        await tester.tap(find.text('peer.test'));
        await tester.pump();
        expect(chat.starts, 1);
        if (outcome == 'start failed') {
          start.completeError(const ApiNetworkError('offline'));
        } else {
          start.complete();
        }
      }
      // The owner change disposes this private view in the real route gate.
      if (outcome == 'owner changed') {
        await tester.pumpWidget(const SizedBox.shrink());
      }
      await tester.pumpAndSettle();
      expect(opens, outcome == 'success' ? 1 : 0);
      if (outcome.endsWith('failed')) {
        expect(messenger.calls.single.$1, 'error');
        expect(
          messenger.calls.single.$2,
          'Messaging is temporarily unavailable. Try again.',
        );
      } else {
        expect(messenger.calls, isEmpty);
      }
      expect(tester.takeException(), isNull);
    });
  }
}

class _Search implements SearchRepository {
  String? query;
  @override
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) async {
    query = q;
    return ProfileSearchPage(
      items: [
        ProfileSearchResult(
          did: 'did:plc:peer',
          handle: 'peer.test',
          isCraftskyProfile: true,
          viewerIsFollowing: false,
        ),
        ProfileSearchResult(
          did: 'did:plc:external',
          handle: 'external.test',
          isCraftskyProfile: false,
          viewerIsFollowing: false,
        ),
        ProfileSearchResult(
          did: 'did:plc:owner',
          handle: 'owner.test',
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
  int starts = 0;
  int checks = 0;
  bool available = true;
  Future<ChatAvailability>? heldAvailability;
  Future<void>? heldStart;
  @override
  Future<ChatAvailability> availability(Did recipient) async {
    checks++;
    return await (heldAvailability ??
        Future.value(ChatAvailability.fromMap({'canChat': available})));
  }

  @override
  Future<ChatConversation> start(Did recipient) async {
    starts++;
    await heldStart;
    return ChatConversation.fromMap({
      'id': 'canonical',
      'revision': 'r',
      'status': 'accepted',
      'muted': false,
      'unreadCount': 0,
      'canSend': true,
      'peer': {'did': recipient.value, 'handle': 'peer.test'},
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
