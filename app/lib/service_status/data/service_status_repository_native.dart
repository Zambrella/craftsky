import 'dart:io';

import 'package:craftsky_app/service_status/data/service_status_body.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:dio/dio.dart';

ServiceStatusRepository createServiceStatusRepository(Uri uri) =>
    NativeServiceStatusRepository(uri);
bool isExpectedStatusPlatformFailure(Object error) => error is IOException;

final class NativeServiceStatusRepository extends ServiceStatusRepository {
  NativeServiceStatusRepository(this.uri);
  final Uri uri;
  final _client = Dio(
    BaseOptions(
      followRedirects: false,
      validateStatus: (_) => true,
      responseType: ResponseType.stream,
      headers: {'Accept': 'application/json'},
    ),
  );

  @override
  StatusRequest start() {
    final token = CancelToken();
    Future<ServiceStatusDocument> read() async {
      try {
        final response = await _client.getUri<ResponseBody>(
          uri,
          cancelToken: token,
        );
        if (response.statusCode != 200 ||
            response.headers
                    .value('content-type')
                    ?.split(';')
                    .first
                    .trim()
                    .toLowerCase() !=
                'application/json') {
          throw const FormatException('Status response');
        }
        // Dio already cancels the response stream with the token. Only the
        // small body-size bound is ours; the controller owns the deadline.
        return ServiceStatusDocument.decode(
          await readStatusBody(response.data!.stream),
        );
      } finally {
        token.cancel();
      }
    }

    return StatusRequest(read(), token.cancel);
  }

  @override
  void close() => _client.close(force: true);
}
