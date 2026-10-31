import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

abstract interface class ChatRepository {
  Future<ChatStatus> status();
  Future<ChatStatus> setConnection({required bool enabled});
  Future<ChatUnreadCounts> unreadCounts();
  Future<bool> pushPreference();
  Future<bool> setPushPreference({required bool enabled});
  Future<ChatPolicySetting> incomingPolicy();
  Future<ChatPolicySetting> setIncomingPolicy(ChatIncomingPolicy value);
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  });
  Future<ChatConversationId> notificationOpen(ChatOpenReference reference);
  Future<ChatReceipt> report(
    ChatOperationId operation,
    ChatConversationId id,
    ChatMessageId message,
    String reasonType,
    String? reason,
  );
  Future<ChatReceipt> reportConversation(
    ChatOperationId operation,
    ChatConversationId id,
    String reasonType,
    String? reason,
  );
  Future<Uri> authorize({
    required String handoffMode,
    required bool pushEnabled,
  });
  Future<ChatConversation> conversation(ChatConversationId id);
  Future<ChatConversationDetails> details(ChatConversationId id);
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  });
  Future<ChatSendResult> send(
    ChatConversationId id,
    ChatOperationId operation,
    String text,
  );
  Future<ChatReceipt> receipt(ChatOperationId operation);
  Future<ChatConversation> updateRead(
    ChatConversationId id,
    ChatMessageId message,
  );
  Future<ChatConversation> accept(ChatConversationId id);
  Future<ChatConversation> setMuted(
    ChatConversationId id, {
    required bool muted,
  });
  Future<void> leave(ChatConversationId id);
  Future<ChatAvailability> availability(Did recipient);
  Future<ChatConversation> start(Did recipient);
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  });
}
