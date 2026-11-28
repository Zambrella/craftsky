import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_published_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';

final class InstagramMigrationApiClient {
  const new(this._dio);

  final Dio _dio;

  Future<InstagramVerificationAttempt> createVerification() => unwrapApi(
    () async {
      final response = await _dio.post<Object?>(
        '/v1/migrations/instagram/verifications',
        data: <String, Object?>{},
      );
      _requireStatus(response, 201);
      return await _decodeMap(
        response,
        InstagramVerificationAttempt.fromCreationMap,
      );
    },
  );

  Future<InstagramVerificationAttempt> getVerification(
    String verificationId,
  ) => unwrapApi(() async {
    final response = await _dio.get<Object?>(
      '/v1/migrations/instagram/verifications/'
      '${Uri.encodeComponent(verificationId)}',
    );
    _requireStatus(response, 200);
    return await _decodeMap(
      response,
      InstagramVerificationAttempt.fromMap,
    );
  });

  Future<InstagramVerificationAttempt?> getCurrentVerification() => unwrapApi(
    () async {
      final response = await _dio.get<Object?>(
        '/v1/migrations/instagram/verifications/current',
      );
      _requireStatus(response, 200);
      return await _decode(response, () {
        final data = response.data;
        if (data is! Map<String, dynamic>) {
          throw const FormatException('invalid_instagram_current_verification');
        }
        final verification = data['verification'];
        if (verification == null) return null;
        if (verification is! Map<String, dynamic>) {
          throw const FormatException('invalid_instagram_current_verification');
        }
        return InstagramVerificationAttempt.fromMap(verification);
      });
    },
  );

  Future<void> cancelVerification(String verificationId) => unwrapApi(
    () async {
      final response = await _dio.delete<void>(
        '/v1/migrations/instagram/verifications/'
        '${Uri.encodeComponent(verificationId)}',
      );
      _requireStatus(response, 204);
    },
  );

  Future<InstagramVerificationConfirmation> confirmVerification(
    String verificationId, {
    required bool discoverable,
  }) => unwrapApi(() async {
    final response = await _dio.post<Object?>(
      '/v1/migrations/instagram/verifications/'
      '${Uri.encodeComponent(verificationId)}/confirm',
      data: {'discoverable': discoverable},
    );
    _requireStatus(response, 200);
    return await _decodeMap(
      response,
      InstagramVerificationConfirmation.fromMap,
    );
  });

  Future<InstagramAccountStatus> getAccount() => unwrapApi(() async {
    final response = await _dio.get<Object?>(
      '/v1/migrations/instagram/account',
    );
    _requireStatus(response, 200);
    return await _decodeMap(response, InstagramAccountStatus.fromMap);
  });

  Future<InstagramAccountStatus> updateSettings(
    InstagramAccountSettingsPatch patch,
  ) => unwrapApi(() async {
    final response = await _dio.patch<Object?>(
      '/v1/migrations/instagram/settings',
      data: patch.toMap(),
    );
    _requireStatus(response, 200);
    return await _decodeMap(response, InstagramAccountStatus.fromMap);
  });

  Future<void> revokeAccount() => unwrapApi(() async {
    final response = await _dio.delete<void>(
      '/v1/migrations/instagram/account',
    );
    _requireStatus(response, 204);
  });

  Future<InstagramImportCreateResult> createImport(
    InstagramImportRequest request,
  ) => unwrapApi(() async {
    final response = await _dio.post<Object?>(
      '/v1/migrations/instagram/imports',
      data: request.toMap(),
    );
    _requireStatus(response, 201);
    return await _decodeMap(response, InstagramImportCreateResult.fromMap);
  });

  Future<InstagramImportPage> listImports({int? limit, String? cursor}) =>
      unwrapApi(() async {
        final response = await _dio.get<Object?>(
          '/v1/migrations/instagram/imports',
          queryParameters: {'limit': ?limit, 'cursor': ?cursor},
        );
        _requireStatus(response, 200);
        return await _decodeMap(response, InstagramImportPage.fromMap);
      });

  Future<InstagramImportSummary> getImport(String importId) => unwrapApi(
    () async {
      final response = await _dio.get<Object?>(
        '/v1/migrations/instagram/imports/${Uri.encodeComponent(importId)}',
      );
      _requireStatus(response, 200);
      return await _decodeMap(response, InstagramImportSummary.fromMap);
    },
  );

  Future<InstagramImportSummary> updateImport(
    String importId,
    InstagramImportPatch patch,
  ) => unwrapApi(() async {
    final response = await _dio.patch<Object?>(
      '/v1/migrations/instagram/imports/${Uri.encodeComponent(importId)}',
      data: patch.toMap(),
    );
    _requireStatus(response, 200);
    return await _decodeMap(response, InstagramImportSummary.fromMap);
  });

  Future<void> deleteImport(String importId) => unwrapApi(() async {
    final response = await _dio.delete<void>(
      '/v1/migrations/instagram/imports/${Uri.encodeComponent(importId)}',
    );
    _requireStatus(response, 204);
  });

  Future<List<InstagramPublishedImport>> lookupPostImports(
    List<String> sourceKeys,
  ) async {
    if (sourceKeys.isEmpty) return const [];
    if (sourceKeys.length > 100 ||
        !sourceKeys.every(InstagramPublishedImport.isSourceKey)) {
      throw ArgumentError('Expected up to 100 Instagram source keys.');
    }
    final requested = sourceKeys.toSet();
    return await unwrapApi(() async {
      final response = await _dio.post<Object?>(
        '/v1/migrations/instagram/post-imports/lookup',
        data: {'sourceKeys': requested.toList()},
      );
      _requireStatus(response, 200);
      return await _decodeMap(response, (data) {
        final responseItems = data['items'];
        if (responseItems is! List<dynamic>) {
          throw const FormatException('invalid_instagram_import_lookup');
        }
        final items = <InstagramPublishedImport>[];
        for (final entry in responseItems) {
          if (entry is! Map<String, dynamic>) {
            throw const FormatException('invalid_instagram_import_lookup_item');
          }
          final item = InstagramPublishedImport.fromMap(entry);
          if (!requested.remove(item.sourceKey)) {
            throw const FormatException(
              'unexpected_instagram_import_lookup_item',
            );
          }
          items.add(item);
        }
        return List.unmodifiable(items);
      });
    });
  }

  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final response = await _dio.get<Object?>(
      '/v1/migrations/instagram/suggestions',
      queryParameters: {'limit': ?limit, 'cursor': ?cursor},
    );
    _requireStatus(response, 200);
    return await _decodeMap(response, InstagramSuggestionPage.fromMap);
  });

  Future<InstagramSuggestionActionResult> acceptSuggestion(
    String suggestionId, {
    required String operationKey,
  }) => unwrapPdsMutationApi(() async {
    _requireCanonicalOperationKey(operationKey);
    final response = await _dio.post<Object?>(
      '/v1/migrations/instagram/suggestions/'
      '${Uri.encodeComponent(suggestionId)}/accept',
      options: Options(
        headers: {'Idempotency-Key': operationKey},
        validateStatus: (status) =>
            status != null && status >= 200 && status < 300,
      ),
    );
    return await parsePdsMutationResponse(
      response,
      accepted: (data) => _decode(response, () {
        _requireStatus(response, 200);
        if (data is! Map<String, dynamic> ||
            data.length != 2 ||
            !data.containsKey('suggestionId') ||
            !data.containsKey('state')) {
          throw const FormatException(
            'invalid_instagram_suggestion_action',
          );
        }
        final result = InstagramSuggestionActionResult.fromMap(data);
        if (result.suggestionId != suggestionId ||
            (result.state != InstagramSuggestionState.followed &&
                result.state != InstagramSuggestionState.alreadyFollowing)) {
          throw const FormatException(
            'invalid_instagram_suggestion_action_state',
          );
        }
        return result;
      }),
    );
  });

  Future<void> dismissSuggestion(String suggestionId) => unwrapApi(() async {
    final response = await _dio.delete<void>(
      '/v1/migrations/instagram/suggestions/'
      '${Uri.encodeComponent(suggestionId)}',
    );
    _requireStatus(response, 204);
  });

  void _requireStatus(Response<Object?> response, int expected) {
    if (response.statusCode != expected) {
      throw const ApiServerError('unexpected_instagram_status');
    }
  }

  T _decode<T>(Response<Object?> response, T Function() decode) {
    try {
      return decode();
    } on ApiException {
      rethrow;
    } on Object catch (error, stack) {
      Error.throwWithStackTrace(
        ApiServerError(
          'invalid_instagram_response',
          details: ErrorMappingInterceptor.detailsFor(
            DioException(
              requestOptions: response.requestOptions,
              response: response,
              error: error,
              stackTrace: stack,
            ),
          ),
        ),
        stack,
      );
    }
  }

  T _decodeMap<T>(
    Response<Object?> response,
    T Function(Map<String, dynamic>) decode,
  ) => _decode(response, () {
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw const FormatException('invalid_instagram_response_map');
    }
    return decode(data);
  });

  void _requireCanonicalOperationKey(String operationKey) {
    if (!isCanonicalPdsMutationOperationKey(operationKey)) {
      throw ArgumentError.value(
        operationKey,
        'operationKey',
        'must be a canonical UUID',
      );
    }
  }
}
