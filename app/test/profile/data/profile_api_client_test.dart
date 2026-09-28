import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
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
    'pronouns': 'she/her',
    'description': 'textile person',
    'crafts': ['sewing'],
  };

  test('decodes and serializes free-form pronouns', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me',
      (server) => server.reply(200, sampleProfile()),
      data: {
        'displayName': 'Alice',
        'pronouns': 'she/they',
        'description': 'textile person',
        'crafts': ['sewing'],
      },
    );

    final profile = await ProfileApiClient(dio).updateMyProfile(
      operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c951',
      displayName: 'Alice',
      pronouns: 'she/they',
      description: 'textile person',
      crafts: ['sewing'],
    );

    expect(profile.pronouns, 'she/her');
  });

  test(
    'profile update forwards its key and parses ambiguous responses',
    () async {
      final dio = buildDio();
      final requests = <RequestOptions>[];
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onPut(
        '/v1/profiles/me',
        (server) => server.reply(
          202,
          {'status': 'ambiguous'},
          headers: {
            Headers.contentTypeHeader: [Headers.jsonContentType],
            'Retry-After': ['2'],
          },
        ),
        data: {'displayName': 'Alice'},
      );

      const operationKey = '018f47a5-1837-7ad1-8f6d-8e8d2a89c957';
      await expectLater(
        ProfileApiClient(dio).updateMyProfile(
          operationKey: operationKey,
          displayName: 'Alice',
        ),
        throwsA(
          isA<PdsMutationAmbiguousException>().having(
            (error) => error.retryAfterSeconds,
            'retryAfterSeconds',
            2,
          ),
        ),
      );
      expect(requests.single.headers['Idempotency-Key'], operationKey);
    },
  );

  test('decodes upcoming event availability from a profile response', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/profiles/@maker.test',
      (server) => server.reply(200, {
        ...sampleProfile(),
        'hasUpcomingEvents': true,
      }),
    );

    final profile = await ProfileApiClient(dio).getProfile('maker.test');

    expect(profile.hasUpcomingEvents, isTrue);
  });

  Map<String, dynamic> sampleFollowerGrowth({
    required bool populated,
    String period = '30d',
  }) => {
    'period': period,
    'rangeStart': period == '7d' ? '2026-08-19' : '2026-07-27',
    'rangeEnd': '2026-08-25',
    'availableFrom': populated ? '2026-07-01' : null,
    'latestSnapshotDate': populated ? '2026-08-25' : null,
    'latestCapturedAt': populated ? '2026-08-25T00:00:02Z' : null,
    'latestFollowerCount': populated ? 42 : null,
    'netChange': populated ? 5 : null,
    'points': List.generate(period == '7d' ? 7 : 30, (index) {
      final start = period == '7d'
          ? DateTime.utc(2026, 8, 19)
          : DateTime.utc(2026, 7, 27);
      final date = start.add(Duration(days: index));
      return {
        'date': date.toIso8601String().substring(0, 10),
        'count': populated
            ? switch (index) {
                0 => 37,
                6 when period == '7d' => 42,
                29 => 42,
                _ => null,
              }
            : null,
      };
    }),
  };

  test('GET follower growth sends period and decodes history states', () async {
    final populatedDio = buildDio();
    DioAdapter(dio: populatedDio).onGet(
      '/v1/profiles/me/follower-growth',
      (server) => server.reply(200, sampleFollowerGrowth(populated: true)),
      queryParameters: {'period': '30d'},
    );

    final populated = await ProfileApiClient(
      populatedDio,
    ).getFollowerGrowth(FollowerGrowthPeriod.thirtyDays);

    expect(populated.latestFollowerCount, 42);
    expect(populated.points[1].count, isNull);

    final emptyDio = buildDio();
    DioAdapter(dio: emptyDio).onGet(
      '/v1/profiles/me/follower-growth',
      (server) => server.reply(200, sampleFollowerGrowth(populated: false)),
      queryParameters: {'period': '30d'},
    );

    final empty = await ProfileApiClient(
      emptyDio,
    ).getFollowerGrowth(FollowerGrowthPeriod.thirtyDays);

    expect(empty.availableFrom, isNull);
    expect(empty.latestFollowerCount, isNull);
  });

  test('rejects a response for a different requested period', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onGet(
      '/v1/profiles/me/follower-growth',
      (server) => server.reply(
        200,
        sampleFollowerGrowth(populated: true, period: '7d'),
      ),
      queryParameters: {'period': '30d'},
    );

    await expectLater(
      ProfileApiClient(dio).getFollowerGrowth(FollowerGrowthPeriod.thirtyDays),
      throwsFormatException,
    );
  });

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
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c952',
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
    ).updateMyProfile(
      operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c953',
      clearAvatar: true,
      clearBanner: true,
    );
  });

  test('replaces customisation with exactly the two wire fields', () async {
    final dio = buildDio();
    DioAdapter(dio: dio).onPut(
      '/v1/profiles/me/customisation',
      (server) => server.reply(200, {
        'colour': 'teal',
        'profileBackground': 'x2',
      }),
      data: {
        'colour': 'teal',
        'profileBackground': 'x2',
      },
    );

    final saved = await ProfileApiClient(dio).updateMyCustomisation(
      const ProfileCustomisation(
        colour: 'teal',
        background: 'x2',
      ),
    );

    expect(
      saved,
      const ProfileCustomisation(
        colour: 'teal',
        background: 'x2',
      ),
    );
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
      operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c954',
      displayName: 'Alice',
      description: 'textile person #Mending',
      crafts: ['sewing'],
    );

    expect(profile.handle.toString(), 'alice.craftsky.social');
  });

  test(
    'POST follow forwards its canonical key and preserves Profile',
    () async {
      final dio = buildDio();
      final requests = <RequestOptions>[];
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onPost(
        '/v1/profiles/@bob.craftsky.social/follows',
        (server) => server.reply(200, sampleProfile()),
      );

      const operationKey = '018f47a5-1837-7ad1-8f6d-8e8d2a89c950';
      final profile = await ProfileApiClient(
        dio,
      ).followProfile('bob.craftsky.social', operationKey: operationKey);

      expect(profile.did.toString(), 'did:plc:alice');
      expect(requests.single.headers['Idempotency-Key'], operationKey);
    },
  );

  test(
    'DELETE unfollow forwards its canonical key and preserves Profile',
    () async {
      final dio = buildDio();
      final requests = <RequestOptions>[];
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onDelete(
        '/v1/profiles/@bob.craftsky.social/follows',
        (server) => server.reply(200, sampleProfile()),
      );

      const operationKey = '018f47a5-1837-7ad1-8f6d-8e8d2a89c950';
      final profile = await ProfileApiClient(
        dio,
      ).unfollowProfile('bob.craftsky.social', operationKey: operationKey);

      expect(profile.did.toString(), 'did:plc:alice');
      expect(requests.single.headers['Idempotency-Key'], operationKey);
    },
  );

  test('follow surfaces the shared ambiguous response contract', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/profiles/@bob.craftsky.social/follows',
      (server) => server.reply(
        202,
        {'status': 'ambiguous'},
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
          'Retry-After': ['3'],
        },
      ),
    );

    expect(
      ProfileApiClient(dio).followProfile(
        'bob.craftsky.social',
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c950',
      ),
      throwsA(
        isA<PdsMutationAmbiguousException>().having(
          (error) => error.retryAfterSeconds,
          'retryAfterSeconds',
          3,
        ),
      ),
    );
  });

  test('unfollow surfaces the shared ambiguous response contract', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onDelete(
      '/v1/profiles/@bob.craftsky.social/follows',
      (server) => server.reply(
        202,
        {'status': 'ambiguous'},
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
          'Retry-After': ['2'],
        },
      ),
    );

    expect(
      ProfileApiClient(dio).unfollowProfile(
        'bob.craftsky.social',
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c950',
      ),
      throwsA(
        isA<PdsMutationAmbiguousException>().having(
          (error) => error.retryAfterSeconds,
          'retryAfterSeconds',
          2,
        ),
      ),
    );
  });

  test('follow rejects a non-canonical operation key before sending', () {
    final dio = buildDio();

    expect(
      ProfileApiClient(
        dio,
      ).followProfile('bob.craftsky.social', operationKey: 'not-a-uuid'),
      throwsArgumentError,
    );
  });

  test('relationship mutations use the mute and list endpoints', () async {
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
    expect((await api.listMutedProfiles(limit: 20)).items.single.muted, isTrue);
    expect(
      (await api.listBlockedProfiles(cursor: 'opaque')).items.single.blocking,
      isTrue,
    );
  });

  test(
    'block forwards its canonical key and decodes accepted relationship',
    () async {
      final dio = buildDio();
      final requests = <RequestOptions>[];
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onPost(
        '/v1/profiles/@bob.craftsky.social/blocks',
        (server) => server.reply(200, {
          'muted': false,
          'blocking': true,
          'blockedBy': false,
          'uri': 'at://did:plc:alice/app.bsky.graph.block/3abc',
          'cid': 'bafyblock',
          'rkey': '3abc',
        }),
      );

      const operationKey = '018f47a5-1837-7ad1-8f6d-8e8d2a89c950';
      final relationship = await ProfileApiClient(
        dio,
      ).blockProfile('bob.craftsky.social', operationKey: operationKey);

      expect(relationship.blocking, isTrue);
      expect(relationship.rkey, '3abc');
      expect(requests.single.headers['Idempotency-Key'], operationKey);
    },
  );

  test(
    'unblock forwards its canonical key and accepts only empty 204',
    () async {
      final dio = buildDio();
      final requests = <RequestOptions>[];
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            requests.add(options);
            handler.next(options);
          },
        ),
      );
      DioAdapter(dio: dio).onDelete(
        '/v1/profiles/@bob.craftsky.social/blocks',
        (server) => server.reply(204, null),
      );

      const operationKey = '018f47a5-1837-7ad1-8f6d-8e8d2a89c950';
      await ProfileApiClient(
        dio,
      ).unblockProfile('bob.craftsky.social', operationKey: operationKey);

      expect(requests.single.headers['Idempotency-Key'], operationKey);
    },
  );

  test('block surfaces only the exact ambiguous response contract', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onPost(
      '/v1/profiles/@bob.craftsky.social/blocks',
      (server) => server.reply(
        202,
        {'status': 'ambiguous'},
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
          'Retry-After': ['3'],
        },
      ),
    );

    expect(
      ProfileApiClient(dio).blockProfile(
        'bob.craftsky.social',
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c950',
      ),
      throwsA(isA<PdsMutationAmbiguousException>()),
    );
  });

  test('unblock surfaces only the exact ambiguous response contract', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onDelete(
      '/v1/profiles/@bob.craftsky.social/blocks',
      (server) => server.reply(
        202,
        {'status': 'ambiguous'},
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
          'Retry-After': ['2'],
        },
      ),
    );

    expect(
      ProfileApiClient(dio).unblockProfile(
        'bob.craftsky.social',
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c950',
      ),
      throwsA(isA<PdsMutationAmbiguousException>()),
    );
  });

  test('unblock retains its key after a malformed success response', () {
    final dio = buildDio();
    DioAdapter(dio: dio).onDelete(
      '/v1/profiles/@bob.craftsky.social/blocks',
      (server) => server.reply(200, {'blocking': false}),
    );

    expect(
      ProfileApiClient(dio).unblockProfile(
        'bob.craftsky.social',
        operationKey: '018f47a5-1837-7ad1-8f6d-8e8d2a89c950',
      ),
      throwsA(isA<PdsMutationAmbiguousException>()),
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
