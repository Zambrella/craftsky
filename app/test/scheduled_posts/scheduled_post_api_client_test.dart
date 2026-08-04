import 'package:craftsky_app/scheduled_posts/data/scheduled_post_api_client.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  test('IT-017 preserves the existing scheduled-post wire contract', () async {
    final dio = Dio(
      BaseOptions(
        baseUrl: 'https://appview.example.com',
        receiveTimeout: const Duration(seconds: 15),
      ),
    );
    final requests = <RequestOptions>[];
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          handler.next(options);
        },
      ),
    );
    final detail = <String, dynamic>{
      'id': 'schedule-1',
      'operationId': 'operation-1',
      'status': 'scheduled',
      'scheduledAt': '2026-08-05T09:30:00.000Z',
      'payload': <String, dynamic>{
        'text': 'A future post',
        'media': <Map<String, dynamic>>[
          {'id': 'media-1', 'mimeType': 'image/jpeg'},
        ],
      },
    };
    DioAdapter(dio: dio)
      ..onPost(
        '/v1/scheduled-posts',
        (server) => server.reply(201, detail),
        data: {
          'operationId': 'operation-1',
          'scheduledAt': '2026-08-05T09:30:00.000Z',
          'payload': detail['payload'],
        },
      )
      ..onPut(
        '/v1/scheduled-posts/schedule-1',
        (server) => server.reply(200, detail),
        data: {
          'scheduledAt': '2026-08-05T09:30:00.000Z',
          'payload': detail['payload'],
        },
      )
      ..onPost(
        '/v1/scheduled-posts/schedule-1/publication',
        (server) => server.reply(202, <String, dynamic>{}),
        data: {'payload': detail['payload']},
      )
      ..onPut(
        '/v1/scheduled-post-media/media-1',
        (server) => server.reply(201, <String, dynamic>{}),
        data: [1, 2, 3, 4],
      );
    final client = ScheduledPostApiClient(dio);
    final at = DateTime.parse('2026-08-05T10:30:00+01:00');

    await client.create(
      operationId: 'operation-1',
      scheduledAt: at,
      payload: detail['payload']! as Map<String, dynamic>,
    );
    await client.update(
      id: 'schedule-1',
      scheduledAt: at,
      payload: detail['payload']! as Map<String, dynamic>,
    );
    await client.publishNow(
      id: 'schedule-1',
      payload: detail['payload']! as Map<String, dynamic>,
    );
    await client.stageMedia(
      id: 'media-1',
      bytes: const [1, 2, 3, 4],
      mimeType: 'image/jpeg',
    );

    expect(
      requests.map((request) => '${request.method} ${request.path}'),
      [
        'POST /v1/scheduled-posts',
        'PUT /v1/scheduled-posts/schedule-1',
        'POST /v1/scheduled-posts/schedule-1/publication',
        'PUT /v1/scheduled-post-media/media-1',
      ],
    );
    expect(requests.last.contentType, 'image/jpeg');
    expect(requests.last.receiveTimeout, Duration.zero);
    expect(requests.last.data, [1, 2, 3, 4]);
  });
}
