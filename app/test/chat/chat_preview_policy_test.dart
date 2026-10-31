import 'package:craftsky_app/chat/models/chat_link_target.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-008 chooses the first supported canonical link, preserving reply focus',
    () {
      final target = ChatLinkTarget.firstIn(
        'https://example.test/no-card '
        'https://app.craftsky.social/posts/did:plc:root/root?focus=at%3A%2F%2Fdid%3Aplc%3Areply%2Fsocial.craftsky.feed.post%2Freply '
        'https://app.craftsky.social/profiles/did:plc:later',
      );
      expect(target?.kind, ChatLinkKind.post);
      expect(target?.did.value, 'did:plc:root');
      expect(
        target?.focus?.value,
        'at://did:plc:reply/social.craftsky.feed.post/reply',
      );
      expect(target.toString(), isNot(contains('did:plc:root')));
      for (final text in [
        'https://app.craftsky.social.evil.test/profiles/did:plc:peer',
        'https://app.craftsky.social@evil.test/profiles/did:plc:peer',
        'https://user@app.craftsky.social/profiles/did:plc:peer',
        'http://app.craftsky.social/profiles/did:plc:peer',
        'https://app.craftsky.social:444/profiles/did:plc:peer',
        'https://app.craftsky.social/profiles/%252Fprivate',
      ]) {
        expect(ChatLinkTarget.firstIn(text), isNull, reason: text);
      }
      expect(
        ChatLinkTarget.firstIn(
          'https://app.craftsky.social/events/did:plc:peer/event.',
        )?.kind,
        ChatLinkKind.event,
      );
      expect(
        ChatLinkTarget.firstIn(
          'https://app.craftsky.social/profiles/did:plc:peer',
        )?.kind,
        ChatLinkKind.profile,
      );
    },
  );
}
