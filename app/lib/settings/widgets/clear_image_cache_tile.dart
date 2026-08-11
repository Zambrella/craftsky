import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
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
          context.showInfo(l10n.settingsImageCacheCleared);
        case (AsyncLoading(), AsyncError(:final error)):
          final appError = AppErrorMapper.map(
            error,
            fallbackKind: AppErrorKind.actionFailed,
            source: 'action',
          );
          context.showError(appError.message(l10n));
        case _:
          break;
      }
    });

    return SettingsRowTile(
      descriptor: const SettingsRowDescriptor(
        id: SettingsRowId.clearImageCache,
        kind: SettingsRowKind.action,
      ),
      label: l10n.settingsClearImageCache,
      leading: Icons.cleaning_services_outlined,
      onTap: state is AsyncLoading
          ? null
          : () => ref.read(clearImageCacheProvider.notifier).clear(),
    );
  }
}
