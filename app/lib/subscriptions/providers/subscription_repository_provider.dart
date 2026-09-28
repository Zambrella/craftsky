import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'subscription_repository_provider.g.dart';

@riverpod
Future<SubscriptionApi> subscriptionRepository(
  Ref ref,
  AccountKey account,
) async {
  final dio = await ref.watch(accountDioProvider(account).future);
  return SubscriptionApiClient(dio);
}
