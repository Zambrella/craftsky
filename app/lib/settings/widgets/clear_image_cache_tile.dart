import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Settings tile that empties both image caches. The action is reversible
/// (images re-download on next view) so there is no confirmation dialog.
class ClearImageCacheTile extends ConsumerWidget {
  const ClearImageCacheTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(clearImageCacheProvider);

    ref.listen(clearImageCacheProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncData()):
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Image cache cleared')),
          );
        case (AsyncLoading(), AsyncError(:final error)):
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Could not clear cache: $error')),
          );
        case _:
          break;
      }
    });

    return ListTile(
      leading: const Icon(Icons.cleaning_services_outlined),
      title: const Text('Clear image cache'),
      enabled: state is! AsyncLoading,
      onTap: () => ref.read(clearImageCacheProvider.notifier).clear(),
    );
  }
}
