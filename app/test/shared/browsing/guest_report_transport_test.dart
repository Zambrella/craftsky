import 'dart:convert';
import 'dart:typed_data';

import 'package:craftsky_app/feed/data/post_api_client.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/profile/data/profile_api_client.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/browsing/public_read_transport.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

class _ReportsAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return ResponseBody.fromString(
      jsonEncode({'reportId': 'guest-report', 'status': 'accepted'}),
      201,
      headers: {
        'content-type': ['application/json'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  test(
    'guest reports omit credentials and other writes stay blocked',
    () async {
      final adapter = _ReportsAdapter();
      final dio =
          Dio(
              BaseOptions(
                baseUrl: 'https://appview.example.test',
                headers: {
                  'Authorization': 'Bearer PRIVATE',
                  'Cookie': 'PRIVATE',
                },
              ),
            )
            ..httpClientAdapter = adapter
            ..interceptors.add(PublicReadInterceptor());
      const submission = ReportSubmission(
        reasonType: 'privacy',
        details: 'Private details',
        contactEmail: 'visitor@example.test',
      );
      final postResult = await PostApiClient(dio).reportPost(
        Did.parse('did:plc:alice'),
        RecordKey.parse('first'),
        submission,
      );
      final profileResult = await ProfileApiClient(dio)
          .reportProfile('did:plc:alice', submission);
      expect(postResult.status, 'accepted');
      expect(profileResult.status, 'accepted');
      expect(adapter.requests.map((r) => r.path), [
        '/v1/public/posts/did:plc:alice/first/reports',
        '/v1/public/profiles/@did:plc:alice/reports',
      ]);
      for (final request in adapter.requests) {
        expect(request.method, 'POST');
        expect(
          request.headers.keys.map((key) => key.toLowerCase()),
          isNot(contains('authorization')),
        );
        expect(
          request.headers.keys.map((key) => key.toLowerCase()),
          isNot(contains('cookie')),
        );
        expect(request.data, submission.toMap());
      }
      for (final path in [
        '/v1/posts/did:plc:alice/first/likes',
        '/v1/profiles/me/reports',
        '/v1/profiles/did:plc:alice/posts/reports',
      ]) {
        await expectLater(
          dio.post<Object?>(path, data: submission.toMap()),
          throwsA(isA<DioException>()),
        );
      }
      expect(adapter.requests, hasLength(2));
      dio.close();
    },
  );
}
