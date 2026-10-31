import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/providers/chat_session_cache_provider.dart';
import 'package:craftsky_app/chat/services/chat_sync_coordinator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Owned by the authenticated shell, independently of messaging pages.
final Provider<ChatSyncCoordinator?> chatSessionRuntimeProvider =
    Provider.autoDispose<ChatSyncCoordinator?>((
      ref,
    ) {
      final lease = ref.watch(
        sessionRegistryProvider.select((value) => value.value?.activeLease),
      );
      if (lease == null) return null;
      final account = lease.session.account;
      ref.watch(accountChatRepositoryProvider(account));
      final status = ref.watch(chatStatusProvider(account)).value;
      if (status == null ||
          !status.enabled ||
          !status.canRead ||
          (status.state != ChatConnectionState.ready &&
              status.state != ChatConnectionState.unavailable)) {
        return null;
      }
      ref
        ..watch(chatUnreadCountsProvider(account))
        ..watch(chatSessionCacheProvider(account))
        ..watch(chatInboxControllerProvider(account, ChatBucket.accepted));
      final sync = ref.watch(chatSyncCoordinatorProvider(account));
      return sync.isLoading || sync.hasError ? null : sync.value;
    });
