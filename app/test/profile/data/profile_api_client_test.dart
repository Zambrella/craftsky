import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  setUpAll(initializeMappers);

  Dio buildDio() {
    return Dio(BaseOptions(baseUrl: 'https://appview.example.com'))
      ..interceptors.add(const ErrorMappingInterceptor());
  }

  Map<String, dynamic> sampleProfile() => {
    'did': 'did:plc:alice',
    'handle': 'alice.craftsky.social',
    'displayName': 'Alice',
    'description': 'textile person',
    'crafts': ['sewing'],
  };

  test(
    'serializes changed avatar and banner blobs in profile updates',
    () async {
      final dio = buildDio();
      const avatar = UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: 'bafavatar'),
        mimeType: 'image/jpeg',
        size: 10,
      );
      const banner = UploadedBlob(
        type: 'blob',
        ref: UploadedBlobRef(link: 'bafbanner'),
        mimeType: 'image/png',
        size: 20,
      );

      DioAdapter(dio: dio).onPut(
        '/v1/profiles/me',
        (server) => server.reply(200, sampleProfile()),
        data: {
          'displayName': 'Alice',
          'crafts': ['sewing'],
          'avatar': {
            r'$type': 'blob',
            'ref': {r'$link': 'bafavatar'},
            'mimeType': 'image/jpeg',
            'size': 10,
          },
          'banner': {
            r'$type': 'blob',
            'ref': {r'$link': 'bafbanner'},
            'mimeType': 'image/png',
            'size': 20,
          },
        },
      );

      final profile = await ProfileApiClient(dio).updateMyProfile(
        displayName: 'Alice',
        crafts: ['sewing'],
        avatar: avatar,
        banner: banner,
      );

      expect(profile.handle.toString(), 'alice.craftsky.social');
    },
  );

  test('serializes explicit null when clearing profile images', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me',
      (server) => server.reply(200, sampleProfile()),
      data: {'avatar': null, 'banner': null},
    );

    await ProfileApiClient(
      dio,
    ).updateMyProfile(clearAvatar: true, clearBanner: true);
  });

  test('REG-001 omits descriptionFacets from profile update body', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me',
      (server) => server.reply(200, sampleProfile()),
      data: {
        'displayName': 'Alice',
        'description': 'textile person #Mending',
        'crafts': ['sewing'],
      },
    );

    final profile = await ProfileApiClient(dio).updateMyProfile(
      displayName: 'Alice',
      description: 'textile person #Mending',
      crafts: ['sewing'],
    );

    expect(profile.handle.toString(), 'alice.craftsky.social');
  });

  test('POST follow uses CraftSky endpoint and no token fields', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/profiles/@bob.craftsky.social/follows',
      (server) => server.reply(200, sampleProfile()),
    );

    final profile = await ProfileApiClient(
      dio,
    ).followProfile('bob.craftsky.social');

    expect(profile.did.toString(), 'did:plc:alice');
  });

  test('DELETE unfollow uses CraftSky endpoint and no token fields', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onDelete(
      '/v1/profiles/@bob.craftsky.social/follows',
      (server) => server.reply(200, sampleProfile()),
    );

    final profile = await ProfileApiClient(
      dio,
    ).unfollowProfile('bob.craftsky.social');

    expect(profile.did.toString(), 'did:plc:alice');
  });

  test('relationship mutations use the six profile endpoints', () async {
    final dio = buildDio();
    final adapter = DioAdapter(dio: dio);
    final response = {
      'muted': false,
      'blocking': false,
      'blockedBy': false,
    };
    adapter
      ..onPost(
        '/v1/profiles/@bob.craftsky.social/mutes',
        (server) => server.reply(200, {...response, 'muted': true}),
      )
      ..onDelete(
        '/v1/profiles/@bob.craftsky.social/mutes',
        (server) => server.reply(200, response),
      )
      ..onPost(
        '/v1/profiles/@bob.craftsky.social/blocks',
        (server) => server.reply(200, {
          ...response,
          'blocking': true,
          'uri': 'at://did:plc:alice/app.bsky.graph.block/3abc',
          'cid': 'bafyblock',
          'rkey': '3abc',
        }),
      )
      ..onDelete(
        '/v1/profiles/@bob.craftsky.social/blocks',
        (server) => server.reply(200, response),
      )
      ..onGet(
        '/v1/profiles/me/mutes',
        (server) => server.reply(200, {
          'items': [
            {
              'did': 'did:plc:bob',
              'handle': 'bob.craftsky.social',
              'isCraftskyProfile': true,
              'muted': true,
              'blocking': false,
              'blockedBy': false,
            },
          ],
          'cursor': 'next-mute',
        }),
        queryParameters: {'limit': 20},
      )
      ..onGet(
        '/v1/profiles/me/blocks',
        (server) => server.reply(200, {
          'items': [
            {
              'did': 'did:plc:bob',
              'handle': 'bob.craftsky.social',
              'isCraftskyProfile': true,
              'muted': false,
              'blocking': true,
              'blockedBy': false,
            },
          ],
        }),
        queryParameters: {'cursor': 'opaque'},
      );

    final api = ProfileApiClient(dio);
    expect((await api.muteProfile('bob.craftsky.social')).muted, isTrue);
    expect((await api.unmuteProfile('bob.craftsky.social')).muted, isFalse);
    final block = await api.blockProfile('bob.craftsky.social');
    expect(block.blocking, isTrue);
    expect(block.rkey, '3abc');
    expect(block.initialized, isTrue);
    expect((await api.unblockProfile('bob.craftsky.social')).blocking, isFalse);
    expect((await api.listMutedProfiles(limit: 20)).items.single.muted, isTrue);
    expect(
      (await api.listBlockedProfiles(cursor: 'opaque')).items.single.blocking,
      isTrue,
    );
  });

  test('GET mutual followers sends pagination and decodes page', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/profiles/@bob.craftsky.social/mutual-followers',
      (server) => server.reply(200, {
        'items': [
          {
            'did': 'did:plc:carol',
            'handle': 'carol.craftsky.social',
            'displayName': 'Carol',
            'isCraftskyProfile': true,
          },
        ],
        'cursor': 'next',
        'totalCount': 12,
      }),
      queryParameters: {'limit': 2, 'cursor': 'opaque'},
    );

    final page = await ProfileApiClient(dio).listMutualFollowers(
      'bob.craftsky.social',
      limit: 2,
      cursor: 'opaque',
    );

    expect(page.totalCount, 12);
    expect(page.cursor, 'next');
    expect(page.items.single.handle.toString(), 'carol.craftsky.social');
  });

  test('GET self followers and following use me endpoints', () async {
    final dio = buildDio();
    final adapter = DioAdapter(dio: dio)
      ..onGet(
        '/v1/profiles/me/followers',
        (server) => server.reply(200, {
          'items': <Map<String, dynamic>>[],
          'totalCount': 0,
        }),
        queryParameters: {'limit': 50},
      )
      ..onGet(
        '/v1/profiles/me/following',
        (server) => server.reply(200, {
          'items': <Map<String, dynamic>>[],
          'totalCount': 0,
        }),
        queryParameters: {'limit': 25, 'cursor': 'next'},
      );

    final api = ProfileApiClient(dio);
    final followers = await api.listFollowersMe(limit: 50);
    final following = await api.listFollowingMe(limit: 25, cursor: 'next');

    expect(followers.totalCount, 0);
    expect(following.totalCount, 0);
    expect(adapter, isNotNull);
  });

  test('POST profile report body and parses accepted response', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/profiles/@bob.craftsky.social/reports',
      (server) => server.reply(201, {
        'reportId': 'report-profile-1',
        'status': 'accepted',
      }),
      data: {'reasonType': 'impersonation', 'details': 'private details'},
    );

    final result = await ProfileApiClient(dio).reportProfile(
      'bob.craftsky.social',
      const ReportSubmission(
        reasonType: 'impersonation',
        details: 'private details',
      ),
    );

    expect(result.reportId, 'report-profile-1');
    expect(result.status, 'accepted');
  });
}
