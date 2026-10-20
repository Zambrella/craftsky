import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/chat/services/chat_preview_resolver.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

import '../feed/fakes/fake_post_repository.dart';
import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/post_fixtures.dart';

void main() {
  test(
    'IT-016 resolves only accepted first target and canonical focused reply',
    () async {
      var current = true;
      var deny = false;
      var posts = 0;
      var profiles = 0;
      final reply =
          postFixture(
            did: 'did:plc:reply',
            rkey: 'reply',
            text: 'Reply excerpt',
          ).copyWith(
            reply: PostReply(
              root: PostRef(
                uri: 'at://did:plc:root/social.craftsky.feed.post/root',
                cid: 'bafyroot',
              ),
              parent: PostRef(
                uri: 'at://did:plc:root/social.craftsky.feed.post/root',
                cid: 'bafyroot',
              ),
            ),
            images: [
              PostImage(
                cid: 'bafyimage',
                mime: 'image/jpeg',
                size: 1,
                alt: 'Permitted image',
                thumb: 'https://cdn.test/image',
              ),
            ],
          );
      final resolver = ChatPreviewResolver(
        account: AccountKey('did:plc:viewer'),
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
        posts: FakePostRepository(
          onFetch: (did, key) async {
            posts++;
            expect(did.value, 'did:plc:reply');
            expect(key.value, 'reply');
            if (deny) {
              throw const ApiBadRequest(
                'not_found',
                details: ApiFailureDetails(statusCode: 404),
              );
            }
            return reply;
          },
        ),
        profiles: FakeProfileRepository(
          onFetch: (_) async {
            profiles++;
            return Profile(
              did: 'did:plc:later',
              handle: 'later.test',
              crafts: const [],
              description: 'Profile excerpt',
            );
          },
        ),
        events: _Events(),
      );
      const text =
          'https://app.craftsky.social/posts/did:plc:root/root?focus=at%3A%2F%2Fdid%3Aplc%3Areply%2Fsocial.craftsky.feed.post%2Freply '
          'https://app.craftsky.social/profiles/did:plc:later';
      expect(await resolver.resolveText(text, accepted: false), isNull);
      expect(posts, 0);
      final preview = await resolver.resolveText(text, accepted: true);
      expect(posts, 1);
      expect(profiles, 0);
      expect(preview?.excerpt, 'Reply excerpt');
      expect(preview?.target.did.value, 'did:plc:root');
      expect(preview?.target.rkey?.value, 'root');
      expect(preview?.target.focus, reply.uri);
      expect(preview?.thumbnail?.toString(), 'https://cdn.test/image');
      expect(preview.toString(), isNot(contains('Reply excerpt')));
      deny = true;
      expect(await resolver.resolveText(text, accepted: true), isNull);
      expect(posts, 2);
      expect(profiles, 0);
      expect(
        (await resolver.resolveText(
          'https://app.craftsky.social/profiles/did:plc:later',
          accepted: true,
        ))?.excerpt,
        'Profile excerpt',
      );
      expect(profiles, 1);
      expect(
        (await resolver.resolveText(
          'https://app.craftsky.social/events/did:plc:host/event',
          accepted: true,
        ))?.title,
        'Workshop',
      );
      current = false;
      expect(await resolver.resolveText(text, accepted: true), isNull);
      expect(posts, 2);
    },
  );
}

class _Events implements BusinessRepository {
  @override
  Future<BusinessEvent> getEvent(Did owner, RecordKey rkey) async =>
      BusinessEvent(
        did: owner.value,
        rkey: rkey.value,
        uri: 'at://did:plc:host/social.craftsky.business.event/event',
        cid: 'bafyevent',
        name: 'Workshop',
        startsAt: DateTime.utc(2026, 11),
        endsAt: DateTime.utc(2026, 11, 1, 2),
        roles: const [],
        status: const BusinessOpenValue(value: 'scheduled', known: true),
        isAllDay: false,
        createdAt: DateTime.utc(2026),
        past: false,
        publicSuppressionReasons: const [],
        upcomingExclusionReasons: const [],
      );

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}
