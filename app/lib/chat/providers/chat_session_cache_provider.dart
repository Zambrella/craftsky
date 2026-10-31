import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/providers/chat_providers.dart';
import 'package:craftsky_app/chat/services/chat_session_cache.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Only the account runtime watches this owner. Content providers read it,
/// avoiding a cycle where retained content keeps its own account cache alive.
// The family builder's concrete type is inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final chatSessionCacheProvider = Provider.autoDispose
    .family<ChatSessionCache, AccountKey>((ref, account) {
      chatAccountOperation(ref, account);
      ref.watch(
        chatStatusProvider(account).select((state) => state.value?.generation),
      );
      final cache = ChatSessionCache();
      ref.onDispose(cache.dispose);
      return cache;
    });

void retainChatInbox(Ref ref, AccountKey account) {
  final link = ref.keepAlive();
  final entry = ref
      .read(chatSessionCacheProvider(account))
      .retainList(link.close);
  ref.onDispose(entry.forget);
}

ChatCacheRetention retainChatConversation(Ref ref, AccountKey account) {
  final link = ref.keepAlive();
  final entry = ref
      .read(chatSessionCacheProvider(account))
      .retainConversation(link.close);
  ref.onDispose(entry.forget);
  return entry;
}
