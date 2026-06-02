import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
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

  test('IT-004 includes descriptionFacets in profile update body', () async {
    final dio = buildDio();
    final descriptionFacets = [
      {
        'index': {'byteStart': 14, 'byteEnd': 22},
        'features': [
          {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'Mending'},
        ],
      },
    ];
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me',
      (server) => server.reply(200, sampleProfile()),
      data: {
        'displayName': 'Alice',
        'description': 'textile person #Mending',
        'crafts': ['sewing'],
        'descriptionFacets': descriptionFacets,
      },
    );

    final profile = await ProfileApiClient(dio).updateMyProfile(
      displayName: 'Alice',
      description: 'textile person #Mending',
      crafts: ['sewing'],
      descriptionFacets: descriptionFacets,
    );

    expect(profile.handle.toString(), 'alice.craftsky.social');
  });

  test(
    'IT-006 maps current AppView descriptionFacets rejection as expected gap',
    () async {
      final dio = buildDio();
      final descriptionFacets = [
        {
          'index': {'byteStart': 15, 'byteEnd': 20},
          'features': [
            {r'$type': 'app.bsky.richtext.facet#tag', 'tag': 'Lace'},
          ],
        },
      ];
      DioAdapter(dio: dio).onPut(
        '/v1/profiles/me',
        (server) => server.reply(400, {
          'error': 'unexpected_field',
          'message': 'unexpected field descriptionFacets',
          'requestId': 'req-description-facets-gap',
        }),
        data: {
          'description': 'Knitting with #Lace',
          'descriptionFacets': descriptionFacets,
        },
      );

      await expectLater(
        ProfileApiClient(dio).updateMyProfile(
          description: 'Knitting with #Lace',
          descriptionFacets: descriptionFacets,
        ),
        throwsA(
          isA<ApiBadRequest>().having(
            (error) => error.code,
            'known current backend gap code',
            'unexpected_field',
          ),
        ),
      );
    },
  );

  test('POST follow uses Craftsky endpoint and no token fields', () async {
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

  test('DELETE unfollow uses Craftsky endpoint and no token fields', () async {
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
