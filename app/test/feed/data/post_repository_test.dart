import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);
  const langs = ['en'];

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
      'text': text,
      'langs': langs,
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
      'viewerHasSaved': false,
      'sponsored': false,
      'createdAt': '2026-05-04T18:23:45.000Z',
      'indexedAt': '2026-05-04T18:23:47.000Z',
      'author': {'did': 'did:plc:alice', 'handle': 'alice.craftsky.social'},
    };
  }

  group('ApiPostRepository.create', () {
    test('IT-002 forwards facets to the API client', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      final facets = [
        {
          'index': {'byteStart': 0, 'byteEnd': 6},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'Mending'},
          ],
        },
      ];
      DioAdapter(dio: dio).onPost(
        '/v1/posts',
        (server) => server.reply(201, samplePost(text: '#Mending')),
        data: {
          'text': '#Mending',
          'langs': langs,
          'sponsored': true,
          'facets': facets,
        },
      );

      final post =
          await ApiPostRepository(
            PostApiClient(dio),
          ).create(
            text: '#Mending',
            langs: langs,
            sponsored: true,
            facets: facets,
          );

      expect(post.text, '#Mending');
    });

    test('IT-003 forwards project through repository interface', () async {
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#embroidery',
        ),
      );
      Project? capturedProject;
      final repo = FakePostRepository(
        onCreateWithFacets:
            ({required text, reply, project, images, facets}) async {
              capturedProject = project;
              return PostMapper.fromMap(
                samplePost(text: text)..['project'] = project?.toMap(),
              );
            },
      );

      final asInterface = repo as PostRepository;
      final post = await asInterface.create(
        text: 'project',
        langs: langs,
        sponsored: false,
        project: project,
      );

      expect(capturedProject, project);
      expect(repo.lastCreateLangs, langs);
      expect(post.project, project);
    });

    test('IT-002 repository rejects project-plus-reply', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#embroidery',
        ),
      );
      final reply = PostReply(
        root: PostRef(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/root',
          cid: 'bafy_root',
        ),
        parent: PostRef(
          uri: 'at://did:plc:alice/social.craftsky.feed.post/parent',
          cid: 'bafy_parent',
        ),
      );

      await expectLater(
        () =>
            ApiPostRepository(
              PostApiClient(dio),
            ).create(
              text: 'invalid',
              langs: langs,
              sponsored: false,
              project: project,
              reply: reply,
            ),
        throwsA(isA<AssertionError>()),
      );
    });

    test('REG-003 repository rejects project-plus-quote', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      const project = Project(
        common: ProjectCommon(
          craftType: 'social.craftsky.feed.defs#embroidery',
        ),
      );
      final quote = PostRef(
        uri: 'at://did:plc:alice/social.craftsky.feed.post/target',
        cid: 'bafy_target',
      );

      await expectLater(
        () =>
            ApiPostRepository(
              PostApiClient(dio),
            ).create(
              text: 'invalid',
              langs: langs,
              sponsored: false,
              project: project,
              quote: quote,
            ),
        throwsA(isA<AssertionError>()),
      );
    });
  });

  group('PostRepository.listProjectsByAuthor', () {
    test('IT-006 fake exposes projects method with cursor and limit', () async {
      String? seenHandle;
      String? seenCursor;
      int? seenLimit;
      final repo = FakePostRepository(
        onListProjectsByAuthor: (handleOrDid, {cursor, limit}) async {
          seenHandle = handleOrDid;
          seenCursor = cursor;
          seenLimit = limit;
          return const PostPage(items: [], cursor: 'next-projects');
        },
      );

      final asInterface = repo as PostRepository;
      final page = await asInterface.listProjectsByAuthor(
        'alice.craftsky.social',
        cursor: 'c1',
        limit: 10,
      );

      expect(seenHandle, 'alice.craftsky.social');
      expect(seenCursor, 'c1');
      expect(seenLimit, 10);
      expect(page.cursor, 'next-projects');
    });
  });

  group('PostRepository.listTimeline', () {
    test('fake exposes timeline method without handle or DID input', () async {
      String? seenCursor;
      int? seenLimit;
      final repo = FakePostRepository(
        onListTimeline: ({cursor, limit}) async {
          seenCursor = cursor;
          seenLimit = limit;
          return const TimelinePage(items: [], cursor: 'next');
        },
      );

      final asInterface = repo as PostRepository;
      final page = await asInterface.listTimeline(cursor: 'c1', limit: 20);

      expect(seenCursor, 'c1');
      expect(seenLimit, 20);
      expect(page.cursor, 'next');
    });
  });

  group('PostRepository interaction lists', () {
    final did = Did.parse('did:plc:bob');
    final rkey = RecordKey.parse('response:1');

    test('IT-009 ApiPostRepository delegates all list arguments', () async {
      final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
      final query = {'cursor': 'opaque:next', 'limit': '12'};
      DioAdapter(dio: dio)
        ..onGet(
          '/v1/posts/did%3Aplc%3Abob/response%3A1/likes',
          (server) => server.reply(200, {
            'items': <Map<String, dynamic>>[],
            'totalCount': 3,
          }),
          queryParameters: query,
        )
        ..onGet(
          '/v1/posts/did%3Aplc%3Abob/response%3A1/reposts',
          (server) => server.reply(200, {
            'items': <Map<String, dynamic>>[],
            'totalCount': 2,
          }),
          queryParameters: query,
        )
        ..onGet(
          '/v1/posts/did%3Aplc%3Abob/response%3A1/quotes',
          (server) => server.reply(200, {
            'items': <Map<String, dynamic>>[],
            'cursor': 'quotes-next',
          }),
          queryParameters: query,
        );
      final repository = ApiPostRepository(PostApiClient(dio));

      final likes = await repository.listLikes(
        did,
        rkey,
        cursor: 'opaque:next',
        limit: 12,
      );
      final reposts = await repository.listReposts(
        did,
        rkey,
        cursor: 'opaque:next',
        limit: 12,
      );
      final quotes = await repository.listQuotes(
        did,
        rkey,
        cursor: 'opaque:next',
        limit: 12,
      );

      expect(likes.totalCount, 3);
      expect(reposts.totalCount, 2);
      expect(quotes.cursor, 'quotes-next');
    });

    test(
      'IT-009 fake exposes programmable callbacks for upcoming tests',
      () async {
        final calls = <String>[];
        final repository = FakePostRepository(
          onListLikes: (seenDid, seenRkey, {cursor, limit}) async {
            calls.add('likes:$seenDid:$seenRkey:$cursor:$limit');
            return const ProfileAccountPage(items: [], totalCount: 1);
          },
          onListReposts: (seenDid, seenRkey, {cursor, limit}) async {
            calls.add('reposts:$seenDid:$seenRkey:$cursor:$limit');
            return const ProfileAccountPage(items: [], totalCount: 2);
          },
          onListQuotes: (seenDid, seenRkey, {cursor, limit}) async {
            calls.add('quotes:$seenDid:$seenRkey:$cursor:$limit');
            return const PostPage(items: [], cursor: 'next');
          },
        );
        final asInterface = repository as PostRepository;

        await asInterface.listLikes(did, rkey, cursor: 'c1', limit: 7);
        await asInterface.listReposts(did, rkey, cursor: 'c2', limit: 8);
        await asInterface.listQuotes(did, rkey, cursor: 'c3', limit: 9);

        expect(calls, [
          'likes:did:plc:bob:response:1:c1:7',
          'reposts:did:plc:bob:response:1:c2:8',
          'quotes:did:plc:bob:response:1:c3:9',
        ]);
      },
    );
  });
}
