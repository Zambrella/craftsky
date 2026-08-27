import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'follower_growth_provider.g.dart';

@riverpod
Future<FollowerGrowth> followerGrowth(
  Ref ref,
  AccountKey account,
  FollowerGrowthPeriod period,
) async {
  final repository = await ref.watch(
    accountFollowerGrowthRepositoryProvider(account).future,
  );
  return repository.fetchFollowerGrowth(period);
}
