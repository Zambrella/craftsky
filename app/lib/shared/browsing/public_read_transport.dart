import 'package:dio/dio.dart';

/// Closed GET map for the existing public-capable repositories.
final class PublicReadRequest {
  factory resolve(
    String path,
    String method,
    Map<String, dynamic> parameters,
  ) {
    if (method != 'GET' ||
        !path.startsWith('/v1/') ||
        path.contains('..') ||
        path.contains('?') ||
        path.contains('#')) {
      throw const FormatException('Public read unavailable');
    }
    var route = path.substring(4);
    final parts = route.split('/').map(Uri.decodeComponent).toList();
    if (parts.any(
      (part) => part.isEmpty || part.contains('/') || part.contains(r'\'),
    )) {
      throw const FormatException('Invalid public route');
    }
    final area = parts.first;
    final feed = route == 'projects';
    final search =
        parts.length == 2 &&
        area == 'search' &&
        const ['profiles', 'posts', 'projects', 'hashtags'].contains(parts[1]);
    final tag =
        parts.length == 4 &&
        area == 'search' &&
        parts[1] == 'hashtags' &&
        parts[3] == 'posts';
    final profile =
        area == 'profiles' &&
        parts.length >= 2 &&
        !const ['me', 'recent'].contains(parts[1].replaceFirst('@', '')) &&
        (parts.length == 2 ||
            (parts.length == 3 &&
                const [
                  'posts',
                  'projects',
                  'comments',
                  'reposts',
                  'events',
                ].contains(parts[2])));
    final post =
        area == 'posts' &&
        parts.length >= 3 &&
        parts[1].startsWith('did:') &&
        (parts.length == 3 ||
            (parts.length == 4 &&
                const [
                  'thread',
                  'comments',
                  'replies',
                  'likes',
                  'reposts',
                  'quotes',
                ].contains(parts[3])));
    final caption =
        area == 'posts' &&
        parts.length == 5 &&
        parts[1].startsWith('did:') &&
        parts[3] == 'video-captions';
    final event =
        area == 'events' && parts.length == 3 && parts[1].startsWith('did:');
    if (!feed && !search && !tag && !profile && !post && !event && !caption) {
      throw const FormatException('Public read unavailable');
    }
    final query = Map<String, dynamic>.of(parameters);
    String? section;
    if (post &&
        parts.length == 4 &&
        const ['comments', 'replies'].contains(parts[3])) {
      section = parts[3];
      route = route.replaceFirst(RegExp(r'/(comments|replies)$'), '/thread');
      query
        ..remove('sort')
        ..remove('focus');
    }
    final paged =
        feed ||
        search ||
        tag ||
        (profile && parts.length == 3) ||
        (post && parts.length == 4);
    if (paged) {
      query['limit'] = (int.tryParse('${query['limit'] ?? 20}') ?? 20)
          .clamp(1, 20)
          .toString();
    }
    final allowed = <String>{
      if (paged) ...['limit', 'cursor'],
      if (search) 'q',
      if (feed) ...[
        'sort',
        'craftType',
        'status',
        'projectType',
        'projectSubtype',
        'patternDifficulty',
        'color',
        'designTag',
        'yarnWeight',
        'piecingTechnique',
        'quiltingMethod',
        'selfDrafted',
      ],
    };
    // Member search defaults are presentation options, never public queries.
    if (search || tag) query.removeWhere((key, _) => !allowed.contains(key));
    if (query.keys.any((key) => !allowed.contains(key)) ||
        (search && '${query['q'] ?? ''}'.trim().isEmpty)) {
      throw const FormatException('Invalid public query');
    }
    return PublicReadRequest._('/v1/public/$route', query, section);
  }
  new _(this.path, this.query, this.section);
  final String path;
  final Map<String, dynamic> query;
  final String? section;
  Object? adapt(Object? value) {
    if (path.contains('/video-captions/')) return value;
    if (section != null && value is Map && value['replies'] is Map) {
      final replies = value['replies'] as Map;
      final items = (replies['items'] as List? ?? [])
          .map((item) => {'post': adaptPublicDisplay(item), 'flattened': false})
          .toList();
      if (section == 'replies') {
        return {'items': items, 'loaded': true, 'cursor': replies['cursor']};
      }
      return {
        'post': adaptPublicDisplay(value['post']),
        'sort': 'oldest',
        'ancestors': [
          for (final ancestor in value['ancestors'] as List? ?? [])
            {
              'state': (ancestor as Map)['state'],
              if (ancestor['post'] != null)
                'post': adaptPublicDisplay(ancestor['post']),
            },
        ],
        'comments': {
          'items': items
              .map(
                (item) => {
                  ...item,
                  'placement': 'normal',
                  'replies': {'items': <Object>[], 'loaded': false},
                },
              )
              .toList(),
          'cursor': replies['cursor'],
        },
      };
    }
    return adaptPublicDisplay(value);
  }
}

Object? adaptPublicDisplay(Object? value) {
  if (value is List) return value.map(adaptPublicDisplay).toList();
  if (value is! Map) return value;
  final result = Map<String, dynamic>.from(value);
  if (value['startsAt'] is String && value['endsAt'] is String) {
    result['publicSuppressionReasons'] = <String>[];
    result['upcomingExclusionReasons'] = <String>[];
  } else if (value['text'] is String && value['author'] is Map) {
    result['author'] = adaptPublicDisplay(value['author']);
    result['indexedAt'] = value['createdAt'];
    for (final name in [
      'viewerHasLiked',
      'viewerHasReposted',
      'viewerHasReplied',
      'viewerHasSaved',
    ]) {
      result[name] = false;
    }
    if (value['quoteView'] case final Map<String, dynamic> quote) {
      result['quoteView'] = {
        ...Map<String, dynamic>.from(quote),
        'revealable': false,
        if (quote['post'] != null) 'post': adaptPublicDisplay(quote['post']),
      };
    }
  } else if (value['did'] is String && value['handle'] is String) {
    final business = result.remove('business');
    result['accountType'] ??= business == true ? 'business' : 'regular';
    if (value['links'] is List) result['business'] = value['businessProfile'];
    result['isCraftskyProfile'] = true;
  } else if (value['items'] is List) {
    result['items'] = adaptPublicDisplay(value['items']);
    result['totalCount'] ??= (value['items'] as List).length;
  }
  return result;
}

final class PublicReadInterceptor extends Interceptor {
  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    try {
      final request = PublicReadRequest.resolve(
        options.path,
        options.method,
        options.queryParameters,
      );
      options
        ..path = request.path
        ..queryParameters = request.query
        ..headers = {'Accept': 'application/json'}
        ..followRedirects = false;
      options.extra['publicReadRequest'] = request;
      handler.next(options);
    } on FormatException {
      handler.reject(
        DioException(
          requestOptions: options,
          type: DioExceptionType.badResponse,
          response: Response<Object?>(
            requestOptions: options,
            statusCode: 403,
            data: {
              'error': 'authentication_required',
              'message': 'Sign in to use this feature.',
            },
          ),
        ),
      );
    }
  }

  @override
  void onResponse(
    Response<dynamic> response,
    ResponseInterceptorHandler handler,
  ) {
    response.data =
        (response.requestOptions.extra['publicReadRequest']
                as PublicReadRequest)
            .adapt(response.data);
    handler.next(response);
  }
}
