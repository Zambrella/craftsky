import 'package:craftsky_app/bootstrap.dart';
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
        (server) => server.reply(200, {'items': [], 'totalCount': 0}),
        queryParameters: {'limit': 50},
      )
      ..onGet(
        '/v1/profiles/me/following',
        (server) => server.reply(200, {'items': [], 'totalCount': 0}),
        queryParameters: {'limit': 25, 'cursor': 'next'},
      );

    final api = ProfileApiClient(dio);
    final followers = await api.listFollowersMe(limit: 50);
    final following = await api.listFollowingMe(limit: 25, cursor: 'next');

    expect(followers.totalCount, 0);
    expect(following.totalCount, 0);
    expect(adapter, isNotNull);
  });
}
