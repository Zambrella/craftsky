import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';

/// A public actor result; identity resolution still belongs to sign-in.
class HandleSuggestion {
  const HandleSuggestion({
    required this.handle,
    this.displayName,
    this.avatarUrl,
  });

  final Handle handle;
  final String? displayName;
  final String? avatarUrl;
}

/// Public discovery, deliberately independent of AppView session transports.
class HandleTypeaheadService {
  HandleTypeaheadService({Dio? dio}) : _dio = dio ?? createDio();

  static const endpoint =
      'https://typeahead.waow.tech/xrpc/tech.waow.typeahead.searchActors';

  final Dio _dio;

  /// Creates a new credential-free transport, never a shared AppView Dio.
  static Dio createDio() => Dio(
    BaseOptions(
      connectTimeout: const Duration(seconds: 5),
      sendTimeout: const Duration(seconds: 5),
      receiveTimeout: const Duration(seconds: 5),
      followRedirects: false,
      headers: {'X-Client': 'craftsky.social'},
    ),
  );

  /// Errors (including cancellation) propagate for the widget to handle.
  Future<List<HandleSuggestion>> search(
    String query, {
    CancelToken? cancelToken,
  }) async {
    final q = query.trim();
    if (q.isEmpty) return const [];
    final response = await _dio.get<Object?>(
      endpoint,
      queryParameters: {'q': q, 'limit': 5},
      cancelToken: cancelToken,
    );
    final data = response.data;
    if (data is! Map || data['actors'] is! List) return const [];
    final suggestions = <HandleSuggestion>[];
    final seen = <String>{};
    for (final actor in data['actors'] as List) {
      if (actor is! Map || actor['handle'] is! String) continue;
      final candidate = (actor['handle'] as String).trim().toLowerCase();
      if (candidate.isEmpty || candidate == 'handle.invalid') continue;
      final Handle handle;
      try {
        handle = Handle.parse(candidate);
      } on Object {
        continue;
      }
      if (!seen.add(handle)) continue;
      final name = actor['displayName'];
      final displayName = name is String ? name.trim() : null;
      suggestions.add(
        HandleSuggestion(
          handle: handle,
          displayName: displayName == '' ? null : displayName,
          avatarUrl: _avatarUrl(actor['avatar']),
        ),
      );
      if (suggestions.length == 5) break;
    }
    return List.unmodifiable(suggestions);
  }

  static String? _avatarUrl(Object? value) {
    if (value is! String) return null;
    final url = value.trim();
    final uri = Uri.tryParse(url);
    if (uri == null ||
        uri.scheme != 'https' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty) {
      return null;
    }
    return url;
  }

  void close() => _dio.close(force: true);
}
