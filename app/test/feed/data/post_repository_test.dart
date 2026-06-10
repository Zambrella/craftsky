import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/data/api_post_repository.dart';
import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/feed/data/post_repository.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

import '../fakes/fake_post_repository.dart';

void main() {
  setUpAll(initializeMappers);

  Map<String, dynamic> samplePost({String text = 'hello'}) {
    return {
      'uri': 'at://did:plc:alice/social.craftsky.feed.post/3lf2abc',
      'cid': 'bafy123',
      'rkey': '3lf2abc',
      'text': text,
      'tags': <String>[],
      'likeCount': 0,
      'repostCount': 0,
      'replyCount': 0,
      'viewerHasLiked': false,
      'viewerHasReposted': false,
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
        data: {'text': '#Mending', 'facets': facets},
      );

      final post = await ApiPostRepository(
        PostApiClient(dio),
      ).create(text: '#Mending', facets: facets);

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
      final post = await asInterface.create(text: 'project', project: project);

      expect(capturedProject, project);
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
        () => ApiPostRepository(
          PostApiClient(dio),
        ).create(text: 'invalid', project: project, reply: reply),
        throwsA(isA<ArgumentError>()),
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
          return const PostPage(items: [], cursor: 'next');
        },
      );

      final asInterface = repo as PostRepository;
      final page = await asInterface.listTimeline(cursor: 'c1', limit: 20);

      expect(seenCursor, 'c1');
      expect(seenLimit, 20);
      expect(page.cursor, 'next');
    });
  });
}
