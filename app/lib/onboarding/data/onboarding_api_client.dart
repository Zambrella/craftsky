import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:dio/dio.dart';

final class OnboardingApiClient {
  const OnboardingApiClient(this._dio);

  final Dio _dio;

  Future<OnboardingCompletion> readStatus() => unwrapApi(() async {
    final response = await _dio.get<Map<String, dynamic>>(
      '/v1/onboarding/status',
    );
    return OnboardingCompletion.fromJson(response.data!);
  });

  Future<OnboardingCompletion> complete() => unwrapApi(() async {
    final response = await _dio.post<Map<String, dynamic>>(
      '/v1/onboarding/completion',
    );
    return OnboardingCompletion.fromJson(response.data!);
  });
}
