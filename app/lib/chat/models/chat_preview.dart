import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:flutter/foundation.dart';

/// Compact current-viewer content; memory-only, without nested social actions.
@immutable
final class ChatPreview {
  const new({
    required this.target,
    required this.title,
    required this.excerpt,
    this.thumbnail,
    this.startsAt,
  });
  final ChatLinkTarget target;
  final String title;
  final String excerpt;
  final Uri? thumbnail;
  final DateTime? startsAt;
  @override
  String toString() => 'ChatPreview(<redacted>)';
}
