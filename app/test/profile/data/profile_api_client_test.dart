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
}
