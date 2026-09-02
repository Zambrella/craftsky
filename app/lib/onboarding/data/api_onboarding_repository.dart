import 'package:craftsky_app/onboarding/data/onboarding_api_client.dart';
import 'package:craftsky_app/onboarding/data/onboarding_repository.dart';
import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';

final class ApiOnboardingRepository implements OnboardingRepository {
  const ApiOnboardingRepository(this._client);

  final OnboardingApiClient _client;

  @override
  Future<OnboardingCompletion> readStatus() => _client.readStatus();

  @override
  Future<OnboardingCompletion> complete() => _client.complete();
}
