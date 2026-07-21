import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class InstagramMigrationApiClient {
  const InstagramMigrationApiClient(this._dio);

  final Dio _dio;

  Future<InstagramVerificationAttempt> createVerification() => unwrapApi(
    () async {
      final response = await _dio.post<Object?>(
        '/v1/migrations/instagram/verifications',
        data: <String, Object?>{},
      );
      _requireStatus(response, 201);
      return _decodeMap(
        response.data,
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
    return _decodeMap(response.data, InstagramVerificationAttempt.fromMap);
  });

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
    return _decodeMap(
      response.data,
      InstagramVerificationConfirmation.fromMap,
    );
  });

  Future<InstagramAccountStatus> getAccount() => unwrapApi(() async {
    final response = await _dio.get<Object?>(
      '/v1/migrations/instagram/account',
    );
    _requireStatus(response, 200);
    return _decodeMap(response.data, InstagramAccountStatus.fromMap);
  });

  Future<InstagramAccountStatus> updateSettings(
    InstagramAccountSettingsPatch patch,
  ) => unwrapApi(() async {
    final response = await _dio.patch<Object?>(
      '/v1/migrations/instagram/settings',
      data: patch.toMap(),
    );
    _requireStatus(response, 200);
    return _decodeMap(response.data, InstagramAccountStatus.fromMap);
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
    return _decodeMap(response.data, InstagramImportCreateResult.fromMap);
  });

  Future<InstagramImportPage> listImports({int? limit, String? cursor}) =>
      unwrapApi(() async {
        final response = await _dio.get<Object?>(
          '/v1/migrations/instagram/imports',
          queryParameters: {'limit': ?limit, 'cursor': ?cursor},
        );
        _requireStatus(response, 200);
        return _decodeMap(response.data, InstagramImportPage.fromMap);
      });

  Future<InstagramImportSummary> getImport(String importId) => unwrapApi(
    () async {
      final response = await _dio.get<Object?>(
        '/v1/migrations/instagram/imports/${Uri.encodeComponent(importId)}',
      );
      _requireStatus(response, 200);
      return _decodeMap(response.data, InstagramImportSummary.fromMap);
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
    return _decodeMap(response.data, InstagramImportSummary.fromMap);
  });

  Future<void> deleteImport(String importId) => unwrapApi(() async {
    final response = await _dio.delete<void>(
      '/v1/migrations/instagram/imports/${Uri.encodeComponent(importId)}',
    );
    _requireStatus(response, 204);
  });

  Future<InstagramSuggestionPage> listSuggestions({
    int? limit,
    String? cursor,
  }) => unwrapApi(() async {
    final response = await _dio.get<Object?>(
      '/v1/migrations/instagram/suggestions',
      queryParameters: {'limit': ?limit, 'cursor': ?cursor},
    );
    _requireStatus(response, 200);
    return _decodeMap(response.data, InstagramSuggestionPage.fromMap);
  });

  Future<InstagramSuggestionActionResult> acceptSuggestion(
    String suggestionId,
  ) => unwrapApi(() async {
    final response = await _dio.post<Object?>(
      '/v1/migrations/instagram/suggestions/'
      '${Uri.encodeComponent(suggestionId)}/accept',
    );
    _requireStatus(response, 200);
    return _decodeMap(response.data, InstagramSuggestionActionResult.fromMap);
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

  T _decode<T>(T Function() decode) {
    try {
      return decode();
    } on ApiException {
      rethrow;
    } on Object {
      throw const ApiServerError('invalid_instagram_response');
    }
  }

  T _decodeMap<T>(
    Object? data,
    T Function(Map<String, dynamic>) decode,
  ) => _decode(() {
    if (data is! Map<String, dynamic>) {
      throw const FormatException('invalid_instagram_response_map');
    }
    return decode(data);
  });
}
