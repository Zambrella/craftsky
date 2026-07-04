import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/errors/app_error_presenter.dart';
import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Settings tile that empties both image caches. The action is reversible
/// (images re-download on next view) so there is no confirmation dialog.
class ClearImageCacheTile extends ConsumerWidget {
  const ClearImageCacheTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(clearImageCacheProvider);

    ref.listen(clearImageCacheProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncData()):
          context.showInfo('Image cache cleared');
        case (AsyncLoading(), AsyncError(:final error)):
          final appError = AppErrorMapper.map(
            error,
            source: AppErrorSource.action,
          );
          context.showError(AppErrorPresenter.message(l10n, appError));
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
