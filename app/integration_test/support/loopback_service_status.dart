import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/service_status/data/service_status_repository.dart';

/// Independent anonymous origin; never shares AppView clients or headers.
final class LoopbackServiceStatus {
  LoopbackServiceStatus._(this.server, this.repository);
  final HttpServer server;
  final ServiceStatusRepository repository;
  final requests = <Map<String, List<String>>>[];
  Map<String, Object> document = {
    'schemaVersion': 1,
    'mode': 'normal',
    'revision': 'initial',
  };
  static Future<LoopbackServiceStatus> start() async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final status = LoopbackServiceStatus._(
      server,
      createServiceStatusRepository(
        Uri.parse('http://127.0.0.1:${server.port}/app.json'),
      ),
    );
    server.listen((request) {
      final headers = <String, List<String>>{};
      request.headers.forEach((name, values) => headers[name] = values);
      status.requests.add(headers);
      request.response.headers.contentType = ContentType.json;
      request.response.write(jsonEncode(status.document));
      unawaited(request.response.close());
    });
    return status;
  }

  Future<void> close() async {
    repository.close();
    await server.close(force: true);
  }
}
