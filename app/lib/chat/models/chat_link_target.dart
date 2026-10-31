import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum ChatLinkKind { post, profile, event }

/// A display-only candidate. Never authorizes a content read or message write.
final class ChatLinkTarget {
  const new _({required this.kind, required this.did, this.rkey, this.focus});
  const new post({required this.did, required this.rkey, this.focus})
    : kind = ChatLinkKind.post;
  const new profile({required this.did})
    : kind = ChatLinkKind.profile,
      rkey = null,
      focus = null;
  const new event({required this.did, required this.rkey})
    : kind = ChatLinkKind.event,
      focus = null;
  final ChatLinkKind kind;
  final Did did;
  final RecordKey? rkey;
  final AtUri? focus;

  static final _httpLinks = RegExp(
    r'''https?://[^\s<>"\x00-\x20]+''',
    caseSensitive: false,
  );
  static Iterable<ChatTextLink> textLinksIn(String text) sync* {
    for (final match in _httpLinks.allMatches(text)) {
      final raw = match.group(0)!.replaceFirst(RegExp(r'[.,;!\)\]\}]+$'), '');
      final uri = Uri.tryParse(raw);
      if (uri != null && uri.host.isNotEmpty && uri.userInfo.isEmpty) {
        yield ChatTextLink(
          start: match.start,
          end: match.start + raw.length,
          uri: uri,
        );
      }
    }
  }

  static Iterable<Uri> linksIn(String text) =>
      textLinksIn(text).map((link) => link.uri);
  static ChatLinkTarget? firstIn(String text) {
    for (final uri in linksIn(text)) {
      final target = fromUri(uri);
      if (target != null) return target;
    }
    return null;
  }

  static ChatLinkTarget? fromUri(Uri uri) {
    if (uri.scheme != 'https' ||
        uri.host != 'app.craftsky.social' ||
        uri.port != 443 ||
        uri.userInfo.isNotEmpty ||
        uri.hasFragment) {
      return null;
    }
    try {
      final parts = uri.pathSegments;
      if (parts.any((part) => part == '.' || part == '..')) return null;
      if (parts.length == 2 && parts[0] == 'profiles' && !uri.hasQuery) {
        return ChatLinkTarget._(
          kind: ChatLinkKind.profile,
          did: Did.parse(parts[1]),
        );
      }
      if (parts.length == 3 && (parts[0] == 'posts' || parts[0] == 'events')) {
        final kind = parts[0] == 'posts'
            ? ChatLinkKind.post
            : ChatLinkKind.event;
        AtUri? focus;
        if (uri.hasQuery) {
          if (kind != ChatLinkKind.post ||
              uri.queryParametersAll.length != 1 ||
              uri.queryParametersAll['focus']?.length != 1) {
            return null;
          }
          focus = AtUri.parse(uri.queryParameters['focus']!);
          if (parseCraftskyPostUri(focus) == null) return null;
        }
        return ChatLinkTarget._(
          kind: kind,
          did: Did.parse(parts[1]),
          rkey: RecordKey.parse(parts[2]),
          focus: focus,
        );
      }
    } on Object {
      return null;
    }
    return null;
  }

  @override
  String toString() => 'ChatLinkTarget(<redacted>)';
}

final class ChatTextLink {
  const new({required this.start, required this.end, required this.uri});
  final int start;
  final int end;
  final Uri uri;
  @override
  String toString() => 'ChatTextLink(<redacted>)';
}
