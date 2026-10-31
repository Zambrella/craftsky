import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';

/// One private, transient initial conversation response.
final class ChatConversationDetails {
  const new({required this.conversation, required this.messages});

  factory fromMap(Map<String, dynamic> map) {
    final conversation = map['conversation'];
    final messages = map['messages'];
    if (conversation is! Map<String, dynamic> ||
        messages is! Map<String, dynamic>) {
      throw const FormatException('Invalid conversation details');
    }
    return ChatConversationDetails(
      conversation: ChatConversation.fromMap(conversation),
      messages: ChatMessagePage.fromMap(messages),
    );
  }

  final ChatConversation conversation;
  final ChatMessagePage messages;

  @override
  String toString() => 'ChatConversationDetails(<redacted>)';
}
