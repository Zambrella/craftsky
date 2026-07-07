import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dio/dio.dart';

/// Stateless, side-effect-free. Both the session Dio and the handoff
/// Dio attach this interceptor; the session Dio additionally installs
/// `_SignOutOn401Interceptor` (see Task 14b).
class ErrorMappingInterceptor extends Interceptor {
  const ErrorMappingInterceptor();

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    handler.next(
      DioException(
        requestOptions: err.requestOptions,
        response: err.response,
        type: err.type,
        error: _mapDioError(err),
        stackTrace: err.stackTrace,
      ),
    );
  }

  ApiException _mapDioError(DioException err) {
    final details = _detailsFor(err);
    return switch (err.type) {
      DioExceptionType.connectionTimeout ||
      DioExceptionType.sendTimeout ||
      DioExceptionType.receiveTimeout ||
      DioExceptionType.connectionError => ApiNetworkError(
        err.message ?? err.type.name,
        details: details,
      ),
      DioExceptionType.badResponse => _mapBadResponse(err),
      DioExceptionType.cancel => ApiCanceled(details: details),
      DioExceptionType.badCertificate || DioExceptionType.unknown =>
        err.error is Exception
            ? ApiNetworkError(err.message ?? 'network_error', details: details)
            : ApiServerError(err.message ?? 'server_error', details: details),
    };
  }

  ApiException _mapBadResponse(DioException err) {
    final status = err.response?.statusCode ?? 0;
    final details = _detailsFor(err);
    if (status == 401) return ApiUnauthorized(details: details);
    if (status >= 400 && status < 500) {
      return ApiBadRequest(details.appViewError, details: details);
    }
    return ApiServerError('http_$status', details: details);
  }

  ApiFailureDetails _detailsFor(DioException err) {
    final data = err.response?.data;
    final appViewError = data is Map && data['error'] is String
        ? data['error'] as String
        : null;
    final requestId = data is Map && data['requestId'] is String
        ? data['requestId'] as String
        : null;
    return ApiFailureDetails(
      statusCode: err.response?.statusCode,
      appViewError: appViewError,
      requestId: requestId,
      endpointCategory: _endpointCategory(err.requestOptions.path),
    );
  }

  String _endpointCategory(String path) {
    final uri = Uri.tryParse(path);
    final pathOnly = uri?.path ?? path.split('?').first;
    final parts = pathOnly
        .split('/')
        .where((part) => part.isNotEmpty && part != 'v1')
        .toList();
    if (parts.isEmpty) return 'appview.unknown';
    return switch (parts) {
      ['auth', 'login'] => 'appview.auth.login',
      ['auth', 'logout'] => 'appview.auth.logout',
      ['whoami'] => 'appview.whoami',
      ['blobs', 'images'] => 'appview.blobs.images',
      ['notifications'] => 'appview.notifications',
      ['projects'] => 'appview.projects',
      ['feed'] => 'appview.feed',
      ['feed', 'timeline'] => 'appview.feed.timeline',
      ['posts'] => 'appview.posts',
      ['posts', _, _] => 'appview.posts.detail',
      ['posts', _, _, 'reports'] => 'appview.posts.reports',
      ['posts', _, _, 'replies'] => 'appview.posts.replies',
      ['posts', _, _, 'comments'] => 'appview.posts.comments',
      ['posts', _, _, 'likes'] => 'appview.posts.likes',
      ['posts', _, _, 'reposts'] => 'appview.posts.reposts',
      ['profiles', 'me'] => 'appview.profiles.me',
      ['profiles', 'me', 'followers'] => 'appview.profiles.me.followers',
      ['profiles', 'me', 'following'] => 'appview.profiles.me.following',
      ['profiles', 'me', ...] => 'appview.profiles.me',
      ['profiles', _] => 'appview.profiles.detail',
      ['profiles', _, 'posts'] => 'appview.profiles.posts',
      ['profiles', _, 'projects'] => 'appview.profiles.projects',
      ['profiles', _, 'comments'] => 'appview.profiles.comments',
      ['profiles', _, 'follows'] => 'appview.profiles.follows',
      ['profiles', _, 'reports'] => 'appview.profiles.reports',
      ['profiles', _, 'mutual-followers'] =>
        'appview.profiles.mutual_followers',
      ['search', 'suggestions'] => 'appview.search.suggestions',
      ['search', 'hashtags'] => 'appview.search.hashtags',
      ['search', 'hashtags', 'top'] => 'appview.search.hashtags.top',
      ['search', 'hashtags', _, 'posts'] => 'appview.search.hashtag_posts',
      ['search', 'profiles'] => 'appview.search.profiles',
      ['search', 'posts'] => 'appview.search.posts',
      ['search', 'projects'] => 'appview.search.projects',
      ['search', 'recent'] => 'appview.search.recent',
      ['search', 'recent', _] => 'appview.search.recent.detail',
      _ => 'appview.unknown',
    };
  }
}
