import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';
import 'package:uuid/uuid.dart';

final _canonicalUuid = RegExp(
  r'^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
);

String newPdsMutationOperationKey() => const Uuid().v4();

bool isCanonicalPdsMutationOperationKey(String value) =>
    _canonicalUuid.hasMatch(value);

final class PdsMutationAmbiguousException implements Exception {
  const new({
    required this.retryAfterSeconds,
    this.rateLimited = false,
    this.responsePending = false,
    this.diagnosticCause,
    this.diagnosticStack,
  });

  final int retryAfterSeconds;
  final bool rateLimited;
  final bool responsePending;

  final Object? diagnosticCause;
  final StackTrace? diagnosticStack;

  @override
  String toString() => 'PdsMutationAmbiguousException(<retryable>)';
}

/// The automatic retry budget ended before the PDS outcome became definite.
/// Repeating the same feature action retries the frozen command unchanged.
final class PdsMutationUnresolvedException implements Exception {
  const new({this.diagnosticCause, this.diagnosticStack});

  final Object? diagnosticCause;
  final StackTrace? diagnosticStack;

  @override
  String toString() => 'PdsMutationUnresolvedException(<retryable>)';
}

/// A missing or unparseable response cannot establish whether the keyed write
/// reached AppView. Keep the operation frozen and retry the original endpoint.
Future<T> unwrapPdsMutationApi<T>(Future<T> Function() request) async {
  try {
    return await unwrapApi(request);
  } on ApiBadRequest catch (error, stack) {
    if (error.details.statusCode != 429 || error.code != 'pds_rate_limited') {
      rethrow;
    }
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: error.details.retryAfterSeconds ?? 5,
      rateLimited: true,
      diagnosticCause: error,
      diagnosticStack: stack,
    );
  } on ApiNetworkError catch (error, stack) {
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: 1,
      diagnosticCause: error,
      diagnosticStack: stack,
    );
  } on ApiServerError catch (error, stack) {
    if (error.details.appViewError == 'video_blob_missing') rethrow;
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: 1,
      diagnosticCause: error,
      diagnosticStack: stack,
    );
  } on FormatException catch (error, stack) {
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: 1,
      diagnosticCause: error,
      diagnosticStack: stack,
    );
    // A malformed success payload may throw a TypeError after AppView accepted
    // the write; the same-key retry must recover the saved response.
    // ignore: avoid_catching_errors
  } on TypeError catch (error, stack) {
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: 1,
      diagnosticCause: error,
      diagnosticStack: stack,
    );
  }
}

T parsePdsMutationResponse<T>(
  Response<Object?> response, {
  required T Function(Object? data) accepted,
  bool requireEmptyNoContent = false,
}) {
  if (response.statusCode == 202) {
    final data = response.data;
    final retryAfter = response.headers.value('retry-after');
    if (data is! Map ||
        data.length != 1 ||
        data['status'] != 'ambiguous' ||
        retryAfter == null ||
        !RegExp(r'^\d+$').hasMatch(retryAfter)) {
      throw const FormatException('Invalid ambiguous mutation response');
    }
    final seconds = int.parse(retryAfter).clamp(1, 5);
    throw PdsMutationAmbiguousException(
      retryAfterSeconds: seconds,
      responsePending: true,
    );
  }
  if (requireEmptyNoContent &&
      (response.statusCode != 204 || response.data != null)) {
    throw const FormatException('Expected an empty 204 mutation response');
  }
  return accepted(response.data);
}
