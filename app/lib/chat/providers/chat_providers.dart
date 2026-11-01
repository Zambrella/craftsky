import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/data/api_business_repository.dart';
import 'package:craftsky_app/business/data/business_api_client.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/chat/data/api_chat_repository.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_preview.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:craftsky_app/chat/providers/chat_session_cache_provider.dart';
import 'package:craftsky_app/chat/services/chat_conversation_controller.dart';
import 'package:craftsky_app/chat/services/chat_inbox_controller.dart';
import 'package:craftsky_app/chat/services/chat_preview_resolver.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/chat/services/chat_report_controller.dart';
import 'package:craftsky_app/chat/services/chat_session_cache.dart';
import 'package:craftsky_app/chat/services/chat_settings_controller.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/profile/data/api_profile_repository.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/search/data/api_search_repository.dart';
import 'package:craftsky_app/search/data/search_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'chat_providers.g.dart';

ActiveAccountOperation chatAccountOperation(Ref ref, AccountKey account) {
  ref.watch(
    sessionRegistryProvider.select((value) => value.value?.activeLease),
  );
  final operation = ActiveAccountOperation.capture(ref);
  if (operation.ownership == null ||
      operation.ownership!.session.account != account) {
    throw const AccountOperationDiscarded();
  }
  operation.requireCurrent();
  return operation;
}

void refreshChatAfterPush(Ref ref, AccountKey account) {
  ref.invalidate(chatUnreadCountsProvider(account));
  // Force loaded lists and the visible conversation to reconcile even if
  // the heartbeat revision has not advanced. Retain controllers and drafts.
  final coordinator = chatSyncCoordinatorProvider(account);
  if (ref.exists(coordinator)) ref.read(coordinator).value?.retry();
}

@riverpod
Future<ChatRepository> accountChatRepository(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  final dio = await operation.step(
    () => ref.watch(accountDioProvider(account).future),
  );
  operation.requireCurrent();
  final cancel = CancelToken();
  ref.onDispose(cancel.cancel);
  return ApiChatRepository(dio, cancelToken: cancel);
}

@Riverpod(name: 'chatStatusProvider')
class MessagingStatus extends _$MessagingStatus {
  ChatStatus? _lastKnown;
  ActiveAccountLease? _owner;
  void publishConfirmedStatus(ChatStatus status) {
    final operation = chatAccountOperation(ref, account);
    _owner = operation.ownership;
    _lastKnown = status;
    state = AsyncData(status);
  }

  @override
  Future<ChatStatus> build(AccountKey account) async {
    final operation = chatAccountOperation(ref, account);
    if (_owner != operation.ownership) {
      _lastKnown = null;
      _owner = operation.ownership;
    }
    final repository = await operation.step(
      () => ref.watch(accountChatRepositoryProvider(account).future),
    );
    try {
      final status = await operation.step(repository.status);
      _lastKnown = status;
      return status;
    } on AccountOperationDiscarded {
      rethrow;
    } on Object catch (error, stack) {
      operation.requireCurrent();
      final statusCode = error is ApiException
          ? error.details.statusCode
          : null;
      final transient =
          error is ApiNetworkError ||
          (error is ApiServerError &&
              (statusCode == null || statusCode >= 500)) ||
          (error is ApiBadRequest && statusCode == 429);
      final previous = _lastKnown;
      if (transient && previous != null) {
        unawaited(
          GuardedErrorReporter(ref.read(errorReporterProvider))
              .captureException(
                error,
                stackTrace: stack,
                context: ReportContext(
                  feature: 'Chat',
                  operation: 'status',
                  classification: 'chat.failure',
                  outcome: DiagnosticOutcome.retry,
                  workflow: PrivateOperationalFailureContext(
                    accountDid: account.did.value,
                  ),
                ),
              ),
        );
        return previous.degraded();
      }
      _lastKnown = null;
      Error.throwWithStackTrace(error, stack);
    }
  }
}

Future<void> _requireMessaging(
  Ref ref,
  AccountKey account,
  ActiveAccountOperation operation,
) async {
  final status = await operation.step(
    () => ref.watch(
      chatStatusProvider(account).selectAsync(
        (value) => (
          enabled: value.enabled,
          generation: value.generation,
          // Retain already-loaded state while eligibility is transiently
          // unavailable under the same live grant and messaging generation.
          // Every server operation still rechecks current eligibility.
          state: value.state == ChatConnectionState.unavailable && value.canRead
              ? ChatConnectionState.ready
              : value.state,
          canRead: value.canRead,
        ),
      ),
    ),
  );
  if (!status.enabled ||
      status.state != ChatConnectionState.ready ||
      !status.canRead) {
    throw const AccountOperationDiscarded();
  }
}

@riverpod
Future<ChatUnreadCounts> chatUnreadCounts(Ref ref, AccountKey account) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  return await operation.step(repository.unreadCounts);
}

@riverpod
class ChatPendingReads extends _$ChatPendingReads {
  @override
  Map<
    ChatConversationId,
    ({ChatBucket bucket, ChatUnreadCounts counts, Object token})
  >
  build(AccountKey account) {
    chatAccountOperation(ref, account);
    ref.watch(
      chatStatusProvider(account).select((value) => value.value?.generation),
    );
    return const {};
  }

  void Function() begin(ChatConversation conversation) {
    final id = conversation.id;
    final counts = ref.read(chatUnreadCountsProvider(account)).value;
    if (state.containsKey(id) || counts == null) return () {};
    final pending = (
      bucket: conversation.bucket,
      counts: counts,
      token: Object(),
    );
    state = {...state, id: pending};
    return () {
      if (ref.mounted && identical(state[id]?.token, pending.token)) {
        state = {...state}..remove(id);
      }
    };
  }
}

@riverpod
ChatUnreadCounts? displayedChatUnreadCounts(Ref ref, AccountKey account) {
  final counts = ref.watch(chatUnreadCountsProvider(account)).value;
  if (counts == null) return null;
  final pending = ref.watch(chatPendingReadsProvider(account));
  int adjusted(int count, ChatBucket bucket) {
    if (count == 100) return count;
    var result = count;
    for (final read in pending.values.where(
      (value) => value.bucket == bucket,
    )) {
      final baseline = bucket == ChatBucket.accepted
          ? read.counts.accepted
          : read.counts.requests;
      if (baseline == 100) continue;
      final reading = pending.values
          .where(
            (value) =>
                value.bucket == bucket && identical(value.counts, read.counts),
          )
          .length;
      final limit = (baseline - reading).clamp(0, 100);
      if (limit < result) result = limit;
    }
    return result;
  }

  return ChatUnreadCounts.fromMap({
    'accepted': adjusted(counts.accepted, ChatBucket.accepted),
    'requests': adjusted(counts.requests, ChatBucket.request),
  });
}

@riverpod
Future<ChatConversationController> chatConversationController(
  Ref ref,
  AccountKey account,
  ChatConversationId id,
) async {
  ChatCacheRetention? retention;
  var idle = false;
  ref
    ..onCancel(() {
      idle = true;
      retention?.markIdle();
    })
    ..onResume(() {
      idle = false;
      retention?.markActive();
    });
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  final registry = await operation.step(
    () => ref.read(sessionRegistryProvider.future),
  );
  final store = ref.read(chatDraftStoreProvider);
  await operation.step(
    () => store.initialize({
      for (final did in registry.sessions.keys) AccountKey(did.value),
    }),
  );
  final controller = ChatConversationController(
    account: account,
    id: id,
    repository: repository,
    store: store,
    draftLease: store.lease(account),
    operation: operation,
    reporter: ref.read(errorReporterProvider),
    onReadStarted: (conversation) => ref
        .read(chatPendingReadsProvider(account).notifier)
        .begin(conversation),
    // Keep the optimistic read until the refreshed count has settled.
    onReadReconciled: () => operation.step(
      () => ref.read(chatUnreadCountsProvider(account).future),
    ),
    onSharedStateChanged: () async {
      operation.requireCurrent();
      ref.invalidate(chatUnreadCountsProvider(account));
      final updates = <Future<void>>[];
      for (final bucket in ChatBucket.values) {
        final family = chatInboxControllerProvider(account, bucket);
        if (ref.exists(family)) {
          final controller = ref.read(family).value;
          if (controller != null) {
            updates.add(controller.refresh(afterCurrent: true));
          } else {
            // An uncached fetch may predate the action; no loaded list is lost.
            ref.invalidate(family);
          }
        }
      }
      await Future.wait(updates);
    },
  );
  ref.onDispose(controller.dispose);
  await operation.step(controller.load);
  if (!controller.failed && !controller.left) {
    retention = retainChatConversation(ref, account);
    if (idle) retention.markIdle();
  }
  return controller;
}

@riverpod
Future<ChatInboxController> chatInboxController(
  Ref ref,
  AccountKey account,
  ChatBucket bucket,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  final controller = ChatInboxController(
    account: account,
    bucket: bucket,
    repository: repository,
    operation: operation,
    reporter: ref.read(errorReporterProvider),
  );
  ref.onDispose(controller.dispose);
  await operation.step(controller.refresh);
  if (!controller.failed) retainChatInbox(ref, account);
  return controller;
}

@riverpod
Future<ChatSettingsController> chatSettingsController(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  await operation.step(
    () => ref.watch(
      chatStatusProvider(account).selectAsync(
        (value) => (
          generation: value.generation,
          enabled: value.enabled,
          state: value.state,
          canRead: value.canRead,
          canSetPolicy: value.canSetPolicy,
        ),
      ),
    ),
  );
  final status = ref.read(chatStatusProvider(account)).requireValue;
  final repository = await operation.step(
    () => ref.watch(accountChatRepositoryProvider(account).future),
  );
  final controller = ChatSettingsController(
    account: account,
    repository: repository,
    operation: operation,
    status: status,
    onConnectionChanged: (status) => ref
        .read(chatStatusProvider(account).notifier)
        .publishConfirmedStatus(status),
    reporter: ref.read(errorReporterProvider),
    onChanged: () async {
      operation.requireCurrent();
      ref.invalidate(chatStatusProvider(account));
    },
  );
  ref.onDispose(controller.dispose);
  await operation.step(controller.load);
  return controller;
}

@riverpod
Future<ChatRecipientController> chatRecipientController(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  final dio = await operation.step(
    () => ref.watch(messagingChatDioProvider(account).future),
  );
  final controller = ChatRecipientController(
    account: account,
    repository: repository,
    searchRepository: ApiSearchRepository(SearchApiClient(dio)),
    operation: operation,
    reporter: ref.read(errorReporterProvider),
  );
  ref.onDispose(controller.dispose);
  return controller;
}

@riverpod
Future<ChatReportController> chatReportController(
  Ref ref,
  AccountKey account,
  ChatConversationId id,
  ChatMessageId? message,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final canReport = await operation.step(
    () => ref.watch(
      chatStatusProvider(account).selectAsync((value) => value.canReport),
    ),
  );
  if (!canReport) throw const AccountOperationDiscarded();
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  final controller = ChatReportController(
    account: account,
    conversationId: id,
    messageId: message,
    repository: repository,
    operation: operation,
    reporter: ref.read(errorReporterProvider),
  );
  ref.onDispose(controller.dispose);
  return controller;
}

@riverpod
Future<ChatConversationId> chatNotificationOpen(
  Ref ref,
  AccountKey account,
  ChatOpenReference reference,
) async {
  final operation = chatAccountOperation(ref, account);
  try {
    await _requireMessaging(ref, account, operation);
    final repository = await operation.step(
      () => ref.watch(messagingChatRepositoryProvider(account).future),
    );
    return await operation.step(() => repository.notificationOpen(reference));
  } on AccountOperationDiscarded {
    rethrow;
  } on Object catch (error, stack) {
    if (operation.isCurrent) {
      unawaited(
        GuardedErrorReporter(ref.read(errorReporterProvider)).captureException(
          error,
          stackTrace: stack,
          context: ReportContext(
            feature: 'Chat',
            operation: 'notificationOpen',
            classification: 'chat.failure',
            workflow: PrivateOperationalFailureContext(
              accountDid: account.did.value,
            ),
          ),
        ),
      );
    }
    rethrow;
  }
}

@riverpod
Future<ChatSyncCoordinator> chatSyncCoordinator(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  final coordinator = ChatSyncCoordinator(
    account: account,
    repository: repository,
    operation: operation,
    reporter: ref.read(errorReporterProvider),
    onAccessLost: (error, id) {
      operation.requireCurrent();
      final code = error is ApiBadRequest ? error.code : null;
      if (code == 'chat_conversation_unavailable') {
        if (id != null) {
          final family = chatConversationControllerProvider(account, id);
          if (ref.exists(family)) ref.read(family).value?.invalidateAccess();
        }
        return;
      }
      final previous = ref.read(chatStatusProvider(account)).value;
      if (previous != null) {
        final next = switch (code) {
          'chat_disconnected' => ChatConnectionState.disabled,
          'chat_restricted' => ChatConnectionState.restricted,
          'chat_connection_changed' => ChatConnectionState.unavailable,
          _ => ChatConnectionState.authorizationNeeded,
        };
        ref
            .read(chatStatusProvider(account).notifier)
            .publishConfirmedStatus(previous.accessLost(next));
      }
      ref.invalidate(chatStatusProvider(account));
    },
    onRevision: (view, id, snapshot) async {
      operation.requireCurrent();
      ref.invalidate(chatUnreadCountsProvider(account));
      if (snapshot.state == 'authorizationNeeded' ||
          snapshot.state == 'restricted' ||
          ref.read(chatStatusProvider(account)).value?.state ==
              ChatConnectionState.unavailable) {
        ref.invalidate(chatStatusProvider(account));
      }
      final updates = <Future<void>>[];
      if (view == ChatView.conversation && id != null) {
        final family = chatConversationControllerProvider(account, id);
        if (ref.exists(family)) {
          final controller = ref.read(family).value;
          if (controller != null) updates.add(controller.refresh());
        }
      }
      for (final bucket in ChatBucket.values) {
        final family = chatInboxControllerProvider(account, bucket);
        if (ref.exists(family)) {
          final controller = ref.read(family).value;
          if (controller != null) updates.add(controller.refresh());
        }
      }
      await Future.wait(updates);
    },
  );
  ref.onDispose(coordinator.dispose);
  return coordinator;
}

@riverpod
Future<ChatPreviewResolver> accountChatPreviewResolver(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final dio = await operation.step(
    () => ref.watch(messagingChatDioProvider(account).future),
  );
  return ChatPreviewResolver(
    account: account,
    operation: operation,
    posts: ApiPostRepository(PostApiClient(dio)),
    profiles: ApiProfileRepository(ProfileApiClient(dio)),
    events: ApiBusinessRepository(
      BusinessApiClient(dio),
      ref.watch(businessTimeZoneServiceProvider),
    ),
    reporter: ref.read(errorReporterProvider),
  );
}

@riverpod
Future<ChatPreview?> chatPreview(
  Ref ref,
  AccountKey account,
  ChatConversationId id,
  ChatMessageId messageId,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final controller = await operation.step(
    () => ref.watch(chatConversationControllerProvider(account, id).future),
  );
  if (controller.conversation?.bucket != ChatBucket.accepted) return null;
  final message = controller.history.messages
      .where((item) => item.id == messageId)
      .firstOrNull;
  if (message?.kind != ChatMessageKind.text ||
      ChatLinkTarget.firstIn(message!.text!) == null) {
    return null;
  }
  final resolver = await operation.step(
    () => ref.watch(accountChatPreviewResolverProvider(account).future),
  );
  return await operation.step(
    () => resolver.resolveText(message.text!, accepted: true),
  );
}

@riverpod
Future<ChatAvailability> chatRecipientAvailability(
  Ref ref,
  AccountKey account,
  ChatRecipientId recipient,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final repository = await operation.step(
    () => ref.watch(messagingChatRepositoryProvider(account).future),
  );
  return await operation.step(() => repository.availability(recipient.did));
}

@riverpod
Future<Dio> messagingChatDio(Ref ref, AccountKey account) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final shared = await operation.step(
    () => ref.watch(accountDioProvider(account).future),
  );
  final cancel = CancelToken();
  ref.onDispose(cancel.cancel);
  final interceptors = Interceptors()
    ..addAll(shared.interceptors)
    ..add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          options.cancelToken = cancel;
          handler.next(options);
        },
      ),
    );
  // The clone shares the account adapter. Cancel this lifetime's requests;
  // closing it would also close unrelated account operations.
  return shared.clone(interceptors: interceptors);
}

@riverpod
Future<ChatRepository> messagingChatRepository(
  Ref ref,
  AccountKey account,
) async {
  final operation = chatAccountOperation(ref, account);
  await _requireMessaging(ref, account, operation);
  final base = await operation.step(
    () => ref.watch(accountChatRepositoryProvider(account).future),
  );
  if (base is! ApiChatRepository) return base;
  final dio = await operation.step(
    () => ref.watch(messagingChatDioProvider(account).future),
  );
  return ApiChatRepository(dio);
}
