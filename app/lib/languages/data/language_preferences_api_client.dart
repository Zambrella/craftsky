import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class LanguagePreferencesApiClient {
  const LanguagePreferencesApiClient(this._dio);

  final Dio _dio;

  Future<LanguagePreferences> get() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/languages/preferences',
    );
    return LanguagePreferences.fromJson(response.data!);
  });

  Future<LanguagePreferences> replace(LanguagePreferences preferences) =>
      unwrapApi(() async {
        final response = await _dio.put<Map<String, dynamic>>(
          '/v1/languages/preferences',
          data: preferences.toJson(),
        );
        return LanguagePreferences.fromJson(response.data!);
      });

  Future<LanguagePreferences> initialize(LanguagePreferences proposal) =>
      unwrapApi(() async {
        final response = await _dio.post<Map<String, dynamic>>(
          '/v1/languages/preferences/initialize',
          data: proposal.toJson(),
        );
        return LanguagePreferences.fromJson(response.data!);
      });
}
