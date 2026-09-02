import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/onboarding/data/api_onboarding_repository.dart';
import 'package:craftsky_app/onboarding/data/onboarding_api_client.dart';
import 'package:craftsky_app/onboarding/data/onboarding_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_repository_provider.g.dart';

@riverpod
Future<OnboardingRepository> onboardingRepository(
  Ref ref,
  AccountSessionLease lease,
) async {
  final dio = await ref.watch(accountDioProvider(lease.account).future);
  return ApiOnboardingRepository(OnboardingApiClient(dio));
}
