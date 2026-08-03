import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_post_repository_provider.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show FutureProviderFamily;

@immutable
final class ScheduledMediaKey {
  const ScheduledMediaKey({required this.account, required this.mediaId});

  final AccountKey account;
  final String mediaId;

  @override
  bool operator ==(Object other) =>
      other is ScheduledMediaKey &&
      other.account == account &&
      other.mediaId == mediaId;

  @override
  int get hashCode => Object.hash(account, mediaId);

  @override
  String toString() => 'ScheduledMediaKey(<redacted>)';
}

final FutureProviderFamily<Uint8List, ScheduledMediaKey>
scheduledMediaBytesProvider = FutureProvider.autoDispose
    .family<Uint8List, ScheduledMediaKey>((ref, key) async {
      final repository = await ref.watch(
        accountScheduledPostRepositoryProvider(key.account).future,
      );
      return repository.mediaBytes(key.mediaId);
    });
