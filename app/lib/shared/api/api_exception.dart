/// Errors surfaced by the API client. Sealed so call sites can
/// exhaustively switch.
sealed class ApiException implements Exception {
  const ApiException(this.message);
  final String message;

  @override
  String toString() => 'ApiException: $message';
}

/// HTTP 401. The global 401 handler in `_ErrorMappingInterceptor`
/// signs the user out before this is rethrown to the caller.
final class ApiUnauthorized extends ApiException {
  const ApiUnauthorized() : super('unauthorized');

  @override
  String toString() => 'ApiUnauthorized: $message';
}

/// HTTP 4xx (non-401). [code] comes from the server's `{"error": "…"}`
/// body when present, else null.
final class ApiBadRequest extends ApiException {
  const ApiBadRequest(this.code) : super(code ?? 'bad_request');
  final String? code;

  @override
  String toString() => 'ApiBadRequest: $message code:$code';
}

/// HTTP 5xx or any non-mapped error response.
final class ApiServerError extends ApiException {
  const ApiServerError(super.message);

  @override
  String toString() => 'ApiServerError: $message';
}

/// Timeout, connection failure, or socket error. Distinct from
/// [ApiServerError] so the background `whoami` validation can
/// tolerate offline launches without signing the user out.
final class ApiNetworkError extends ApiException {
  const ApiNetworkError(super.message);

  @override
  String toString() => 'ApiNetworkError: $message';
}
