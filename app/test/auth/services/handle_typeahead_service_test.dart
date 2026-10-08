import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  late Dio dio;
  late DioAdapter adapter;
  late HandleTypeaheadService service;

  setUp(() {
    dio = HandleTypeaheadService.createDio();
    adapter = DioAdapter(dio: dio);
    service = HandleTypeaheadService(dio: dio);
  });
  tearDown(() => service.close());

  void reply(Object? body, {int status = 200}) {
    adapter.onGet(
      HandleTypeaheadService.endpoint,
      (server) => server.reply(status, body),
      queryParameters: {'q': 'alice', 'limit': 5},
    );
  }

  test(
    'uses bounded credential-free transport and exact public request',
    () async {
      expect(dio.interceptors.whereType<InterceptorsWrapper>(), isEmpty);
      expect(dio.options.followRedirects, isFalse);
      expect(dio.options.connectTimeout, const Duration(seconds: 5));
      expect(dio.options.sendTimeout, const Duration(seconds: 5));
      expect(dio.options.receiveTimeout, const Duration(seconds: 5));
      expect(dio.options.headers, {'X-Client': 'craftsky.social'});
      late RequestOptions request;
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            request = options;
            handler.next(options);
          },
        ),
      );
      reply({
        'actors': [
          {'handle': 'alice.example', 'displayName': ' Alice '},
        ],
      });
      final result = await service.search(' alice ');
      expect(request.uri.origin, 'https://typeahead.waow.tech');
      expect(request.method, 'GET');
      expect(result.single.handle, 'alice.example');
      expect(result.single.displayName, 'Alice');
    },
  );

  test('filters invalid, unavailable and duplicate handles', () async {
    reply({
      'actors': [
        null,
        4,
        <String, Object?>{},
        {'handle': 5},
        {'handle': 'bad handle'},
        {'handle': 'handle.invalid'},
        {'handle': ''},
        {'handle': 'Alice.Example'},
        {'handle': 'alice.example', 'displayName': 'duplicate'},
        {'handle': 'bob.example', 'displayName': 4},
        {'handle': 'carol.example', 'displayName': ' '},
      ],
    });
    final result = await service.search('alice');
    expect(result.map((s) => s.handle), [
      'alice.example',
      'bob.example',
      'carol.example',
    ]);
    expect(result.map((s) => s.displayName), [null, null, null]);
  });

  test('preserves public HTTPS avatar URLs', () async {
    reply({
      'actors': [
        {
          'handle': 'alice.example',
          'avatar': ' https://cdn.bsky.app/img/avatar/plain/alice.jpg ',
        },
      ],
    });
    final result = await service.search('alice');
    expect(
      result.single.avatarUrl,
      'https://cdn.bsky.app/img/avatar/plain/alice.jpg',
    );
  });

  for (final avatar in <Object?>[
    null,
    '',
    ' ',
    42,
    '/avatar.jpg',
    'http://example.com/avatar.jpg',
    'data:image/png;base64,abc',
    'https://',
    'https://user:password@example.com/avatar.jpg',
  ]) {
    test(
      'ignores unusable avatar $avatar without dropping the handle',
      () async {
        reply({
          'actors': [
            {'handle': 'alice.example', 'avatar': avatar},
          ],
        });
        final result = await service.search('alice');
        expect(result.single.handle, 'alice.example');
        expect(result.single.avatarUrl, isNull);
      },
    );
  }

  test('caps valid suggestions at five', () async {
    reply({
      'actors': List.generate(10, (i) => {'handle': 'actor$i.example'}),
    });
    expect(await service.search('alice'), hasLength(5));
  });

  test('blank query avoids the network', () async {
    expect(await service.search('  '), isEmpty);
  });

  for (final body in <Object?>[
    null,
    [],
    {},
    {'actors': 'bad'},
  ]) {
    test('unusable response $body returns no suggestions', () async {
      reply(body);
      expect(await service.search('alice'), isEmpty);
    });
  }

  test('HTTP failure propagates', () async {
    reply({}, status: 503);
    await expectLater(service.search('alice'), throwsA(isA<DioException>()));
  });

  test('redirect is not accepted as a successful result', () async {
    reply({}, status: 302);
    await expectLater(service.search('alice'), throwsA(isA<DioException>()));
  });

  test('forwards widget cancellation', () async {
    final token = CancelToken()..cancel('superseded');
    await expectLater(
      service.search('alice', cancelToken: token),
      throwsA(
        isA<DioException>().having(
          (error) => error.type,
          'type',
          DioExceptionType.cancel,
        ),
      ),
    );
  });
}
