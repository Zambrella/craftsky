import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/scheduled_posts/data/api_scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_api_client.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'scheduled_post_repository_provider.g.dart';

@riverpod
Future<ScheduledPostRepository> accountScheduledPostRepository(
  Ref ref,
  AccountKey account,
) async {
  final dio = await ref.watch(accountDioProvider(account).future);
  return ApiScheduledPostRepository(ScheduledPostApiClient(dio));
}
