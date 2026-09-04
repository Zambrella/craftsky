import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/scheduled_posts/providers/scheduled_media_provider.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ScheduledMediaThumbnail extends ConsumerWidget {
  const ScheduledMediaThumbnail({
    required this.account,
    required this.mediaId,
    this.size = 48,
    super.key,
  });

  final AccountKey account;
  final String mediaId;
  final double size;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final bytes = ref.watch(
      scheduledMediaBytesProvider(
        ScheduledMediaKey(account: account, mediaId: mediaId),
      ),
    );
    return SizedBox.square(
      dimension: size,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: switch (bytes) {
          AsyncData(:final value) => Image.memory(
            value,
            fit: BoxFit.cover,
            semanticLabel: l10n.scheduledPostsThumbnailSemantics,
          ),
          AsyncError() => const Icon(CraftskyIcons.brokenImage),
          _ => const Center(child: CircularProgressIndicator()),
        },
      ),
    );
  }
}

List<String> scheduledPayloadMediaIDs(Map<String, dynamic>? payload) {
  final media = payload?['media'];
  if (media is! List<dynamic>) return const [];
  return [
    for (final item in media)
      if (item is Map<dynamic, dynamic> && item['id'] is String)
        item['id']! as String,
  ];
}
