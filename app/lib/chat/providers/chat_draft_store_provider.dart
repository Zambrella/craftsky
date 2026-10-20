import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Install-wide serializer survives account provider invalidation.
final chatDraftStoreProvider = Provider<ChatDraftStore>(
  (ref) => ChatDraftStore(ChatDraftStore.localStorage),
);
