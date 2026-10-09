import 'dart:async';
import 'dart:js_interop';

import 'package:craftsky_app/service_status/data/service_status_body.dart';
import 'package:craftsky_app/service_status/data/service_status_repository.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:web/web.dart' as web;

typedef StatusFetch =
    Future<web.Response> Function(String uri, web.RequestInit init);

ServiceStatusRepository createServiceStatusRepository(Uri uri) =>
    WebServiceStatusRepository(uri);
bool isExpectedStatusPlatformFailure(Object error) => false;

final class WebServiceStatusRepository extends ServiceStatusRepository {
  WebServiceStatusRepository(this.uri, {StatusFetch? fetch})
    : _fetch =
          fetch ?? ((uri, init) => web.window.fetch(uri.toJS, init).toDart);
  final Uri uri;
  final StatusFetch _fetch;
  final _active = <web.AbortController>{};

  @override
  StatusRequest start() {
    final abort = web.AbortController();
    _active.add(abort);
    Future<ServiceStatusDocument> read() async {
      web.ReadableStreamDefaultReader? reader;
      try {
        final response = await _fetch(
          uri.toString(),
          web.RequestInit(
            method: 'GET',
            credentials: 'omit',
            redirect: 'error',
            cache: 'no-store',
            referrerPolicy: 'no-referrer',
            signal: abort.signal,
            headers: web.Headers()..set('Accept', 'application/json'),
          ),
        );
        if (response.status != 200 ||
            response.headers
                    .get('content-type')
                    ?.split(';')
                    .first
                    .trim()
                    .toLowerCase() !=
                'application/json' ||
            response.body == null) {
          throw const FormatException('Status response');
        }
        reader = web.ReadableStreamDefaultReader(response.body!);
        Stream<List<int>> chunks() async* {
          while (true) {
            final chunk = await reader!.read().toDart;
            if (chunk.done) break;
            yield (chunk.value! as JSUint8Array).toDart;
          }
        }

        return ServiceStatusDocument.decode(await readStatusBody(chunks()));
      } on Object catch (error, stack) {
        if (error is Error || error is FormatException) rethrow;
        throw StatusNetworkFailure(error, stack);
      } finally {
        abort.abort();
        reader?.releaseLock();
        _active.remove(abort);
      }
    }

    // JS interop members cannot be torn off by the Dart web compiler.
    // ignore: unnecessary_lambdas
    return StatusRequest(read(), () => abort.abort());
  }

  @override
  void close() {
    for (final abort in _active) {
      abort.abort();
    }
    _active.clear();
  }
}
