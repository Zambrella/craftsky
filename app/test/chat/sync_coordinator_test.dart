import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/api_chat_repository.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  for (final transition in [
    'background and resume',
    'view switch',
    'backgrounded response',
  ]) {
    testWidgets(
      'UT-006 in-flight rate limit survives $transition',
      (tester) async {
        final remote = _Repository();
        final refreshedViews = <ChatView>[];
        final coordinator = ChatSyncCoordinator(
          account: AccountKey('did:plc:owner'),
          repository: remote,
          now: () => tester.binding.clock.now(),
          operation: ActiveAccountOperation.withGuard(
            ownership: null,
            isCurrent: () => true,
          ),
          onRevision: (view, id, snapshot) async {
            refreshedViews.add(view);
          },
        );
        addTearDown(coordinator.dispose);
        coordinator
          ..setView(
            ChatView.conversation,
            conversationId: ChatConversationId.parse('conversation'),
          )
          ..setForeground(active: true);
        await tester.pump();
        expect(remote.calls, 1);
        final held = Completer<ChatSyncSnapshot>();
        remote.held = held;
        await tester.pump(const Duration(seconds: 4));
        expect(remote.calls, 2);
        switch (transition) {
          case 'background and resume':
            coordinator
              ..setForeground(active: false)
              ..setForeground(active: true);
          case 'view switch':
            coordinator.setView(ChatView.inbox);
          case 'backgrounded response':
            coordinator.setForeground(active: false);
        }
        remote
          ..held = null
          ..revision = 2;
        held.completeError(
          const ApiBadRequest(
            'rate_limited',
            details: ApiFailureDetails(statusCode: 429, retryAfterSeconds: 120),
          ),
        );
        await tester.pump();
        final backgrounded = transition == 'backgrounded response';
        if (backgrounded) {
          await tester.pump(const Duration(seconds: 20));
          expect(remote.calls, 2);
          coordinator.setForeground(active: true);
        }
        coordinator.retry();
        await tester.pump(Duration(seconds: backgrounded ? 99 : 119));
        expect(remote.calls, 2);
        expect(refreshedViews, [ChatView.conversation]);
        expect(coordinator.snapshot?.revision, 1);
        await tester.pump(const Duration(seconds: 1));
        expect(remote.calls, 3);
        expect(refreshedViews, [
          ChatView.conversation,
          if (transition == 'view switch')
            ChatView.inbox
          else
            ChatView.conversation,
        ]);
        expect(coordinator.snapshot?.revision, 2);
        coordinator.dispose();
      },
    );
  }
  for (final boundary in ['dispose', 'owner activation change']) {
    testWidgets('UT-006 late rate limit is discarded after $boundary', (
      tester,
    ) async {
      final remote = _Repository();
      var current = true;
      var refreshes = 0;
      final coordinator = ChatSyncCoordinator(
        account: AccountKey('did:plc:owner'),
        repository: remote,
        now: () => tester.binding.clock.now(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
        onRevision: (view, id, snapshot) async {
          refreshes++;
        },
      );
      addTearDown(coordinator.dispose);
      coordinator
        ..setView(ChatView.inbox)
        ..setForeground(active: true);
      await tester.pump();
      final held = Completer<ChatSyncSnapshot>();
      remote.held = held;
      await tester.pump(const Duration(seconds: 30));
      expect(remote.calls, 2);
      if (boundary == 'dispose') {
        coordinator.dispose();
      } else {
        current = false;
      }
      remote.held = null;
      held.completeError(
        const ApiBadRequest(
          'rate_limited',
          details: ApiFailureDetails(statusCode: 429, retryAfterSeconds: 120),
        ),
      );
      await tester.pump();
      coordinator
        ..retry()
        ..setForeground(active: false)
        ..setForeground(active: true);
      final nextRemote = _Repository();
      final next = ChatSyncCoordinator(
        account: AccountKey('did:plc:next-owner'),
        repository: nextRemote,
        now: () => tester.binding.clock.now(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onRevision: (view, id, snapshot) async {},
      );
      addTearDown(next.dispose);
      next
        ..setView(ChatView.inbox)
        ..setForeground(active: true);
      await tester.pump();
      expect(nextRemote.calls, 1);
      expect(next.snapshot?.revision, 1);
      next.dispose();
      await tester.pump(const Duration(seconds: 121));
      expect(remote.calls, 2);
      expect(refreshes, 1);
      coordinator.dispose();
    });
  }
  for (final pendingResponse in [false, true]) {
    testWidgets('UT-006 consumer HTTP Retry-After survives repository mapping '
        'pending=$pendingResponse', (
      tester,
    ) async {
      Future<void> settleHttp() async {
        for (var i = 0; i < 12; i++) {
          await tester.pump(const Duration(milliseconds: 1));
        }
      }

      var calls = 0;
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.test'))
        ..interceptors.addAll([
          InterceptorsWrapper(
            onRequest: (options, handler) {
              calls++;
              handler.next(options);
            },
          ),
          const ErrorMappingInterceptor(),
        ]);
      final adapter = DioAdapter(dio: dio);
      void reply({required bool limited, Duration? delay}) {
        adapter
          ..reset()
          ..onPost(
            '/v1/chat/sync',
            (server) => limited
                ? server.reply(
                    429,
                    {
                      'error': 'rate_limited',
                      'message': 'messaging is temporarily rate limited',
                      'requestId': 'synthetic',
                    },
                    headers: {
                      'content-type': ['application/json'],
                      'retry-after': ['120'],
                    },
                    delay: delay,
                  )
                : server.reply(200, {'state': 'ready', 'revision': 1}),
            data: Matchers.any,
          );
      }

      reply(limited: false);
      final coordinator = ChatSyncCoordinator(
        account: AccountKey('did:plc:owner'),
        repository: ApiChatRepository(dio),
        now: () => tester.binding.clock.now(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onRevision: (view, id, snapshot) async {},
      );
      addTearDown(coordinator.dispose);
      coordinator
        ..setView(
          ChatView.conversation,
          conversationId: ChatConversationId.parse('conversation'),
        )
        ..setForeground(active: true);
      await settleHttp();
      expect(calls, 1);
      expect(coordinator.snapshot?.revision, 1);
      reply(
        limited: true,
        delay: pendingResponse ? const Duration(seconds: 5) : null,
      );
      await tester.pump(const Duration(seconds: 4));
      await settleHttp();
      if (pendingResponse) {
        expect(coordinator.busy, isTrue);
        coordinator
          ..setForeground(active: false)
          ..setForeground(active: true);
        await tester.pump(const Duration(seconds: 5));
        await settleHttp();
      }
      expect(coordinator.failed, !pendingResponse);
      expect(calls, 2);
      coordinator
        ..retry()
        ..setForeground(active: false)
        ..setForeground(active: true);
      await tester.pump(const Duration(seconds: 119));
      expect(calls, 2);
      expect(coordinator.snapshot?.revision, 1);
      reply(limited: false);
      await tester.pump(const Duration(seconds: 1));
      await settleHttp();
      expect(calls, 3);
      expect(coordinator.failed, false);
      coordinator.dispose();
    });
  }
  testWidgets('UT-006 recovery refreshes unchanged revisions after failure', (
    tester,
  ) async {
    final remote = _Repository();
    var refreshes = 0;
    final coordinator = ChatSyncCoordinator(
      account: AccountKey('did:plc:owner'),
      repository: remote,
      now: () => tester.binding.clock.now(),
      operation: ActiveAccountOperation.withGuard(
        ownership: null,
        isCurrent: () => true,
      ),
      onRevision: (view, id, snapshot) async {
        refreshes++;
      },
    );
    addTearDown(coordinator.dispose);
    coordinator
      ..setView(
        ChatView.conversation,
        conversationId: ChatConversationId.parse('conversation'),
      )
      ..setForeground(active: true);
    await tester.pump();
    remote.failure = const ApiNetworkError('offline');
    await tester.pump(const Duration(seconds: 4));
    expect(coordinator.failed, true);
    remote.failure = null;
    await tester.pump(const Duration(seconds: 4));
    expect(coordinator.failed, false);
    expect(refreshes, 2);
    expect(coordinator.snapshot?.revision, 1);
    coordinator.dispose();
  });

  testWidgets(
    'UT-006 degraded sync retains state and honors Retry-After, including '
    'manual retry',
    (tester) async {
      final remote = _Repository();
      var refreshes = 0;
      final coordinator = ChatSyncCoordinator(
        account: AccountKey('did:plc:owner'),
        repository: remote,
        now: () => tester.binding.clock.now(),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => true,
        ),
        onRevision: (view, id, snapshot) async {
          refreshes++;
        },
      );
      addTearDown(coordinator.dispose);
      coordinator
        ..setView(
          ChatView.conversation,
          conversationId: ChatConversationId.parse('conversation'),
        )
        ..setForeground(active: true);
      await tester.pump();
      remote.failure = const ApiServerError(
        'rate_limited',
        details: ApiFailureDetails(statusCode: 429, retryAfterSeconds: 120),
      );
      await tester.pump(const Duration(seconds: 4));
      expect(coordinator.failed, true);
      expect(coordinator.snapshot?.revision, 1);
      expect(refreshes, 1);
      coordinator
        ..retry()
        ..setForeground(active: false)
        ..setForeground(active: true);
      await tester.pump(const Duration(seconds: 119));
      expect(remote.calls, 2);
      remote
        ..failure = null
        ..revision = 2;
      await tester.pump(const Duration(seconds: 1));
      expect(remote.calls, 3);
      expect(coordinator.failed, false);
      expect(refreshes, 2);
      coordinator.dispose();
    },
  );

  testWidgets(
    'UT-006 one foreground timer coalesces and stops at lifecycle boundaries',
    (tester) async {
      final remote = _Repository();
      var current = true;
      var refreshes = 0;
      final coordinator = ChatSyncCoordinator(
        account: AccountKey('did:plc:owner'),
        repository: remote,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
        onRevision: (view, id, snapshot) async {
          refreshes++;
        },
      );
      addTearDown(coordinator.dispose);
      coordinator
        ..setView(
          ChatView.conversation,
          conversationId: ChatConversationId.parse('conversation'),
        )
        ..setForeground(active: true);
      await tester.pump();
      expect(remote.calls, 1);
      expect(refreshes, 1);
      await tester.pump(const Duration(seconds: 3));
      expect(remote.calls, 1);
      remote.held = Completer<ChatSyncSnapshot>();
      await tester.pump(const Duration(seconds: 1));
      expect(remote.calls, 2);
      coordinator
        ..retry()
        ..retry();
      await tester.pump(const Duration(seconds: 20));
      expect(remote.calls, 2);
      coordinator.setForeground(active: false);
      remote.held!.complete(_snapshot(2));
      await tester.pump();
      expect(refreshes, 1);
      await tester.pump(const Duration(seconds: 40));
      expect(remote.calls, 2);
      remote
        ..held = null
        ..revision = 2;
      coordinator
        ..setView(ChatView.inbox)
        ..setForeground(active: true);
      await tester.pump();
      expect(remote.calls, 3);
      expect(refreshes, 2);
      await tester.pump(const Duration(seconds: 29));
      expect(remote.calls, 3);
      await tester.pump(const Duration(seconds: 1));
      expect(remote.calls, 4);
      expect(refreshes, 2);
      current = false;
      await tester.pump(const Duration(seconds: 60));
      expect(remote.calls, 4);
    },
  );
}

ChatSyncSnapshot _snapshot(int revision) => ChatSyncSnapshot.fromMap({
  'revision': revision,
  'state': 'ready',
  'lastSuccessAt': '2026-10-19T12:00:00Z',
});

class _Repository implements ChatRepository {
  Object? failure;
  int calls = 0;
  int revision = 1;
  Completer<ChatSyncSnapshot>? held;
  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    calls++;
    if (failure case final error?) {
      Error.throwWithStackTrace(error, StackTrace.current);
    }
    return held == null ? _snapshot(revision) : await held!.future;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
