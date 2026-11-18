import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/providers/account_hosting_provider.dart';
import 'package:craftsky_app/settings/settings_links.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class AccountProviderSection extends ConsumerWidget {
  const new({
    required this.did,
    required this.handle,
    this.linkLauncher = launchExternalLink,
    super.key,
  });

  final String did;
  final String handle;
  final ExternalLinkLauncher linkLauncher;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(accountHostingProvider);
    final loaded = state.value;
    final info = loaded?.did == did ? loaded : null;
    final manage = hydratedExternalActionUri(info?.accountManagementUrl);
    final password = hydratedExternalActionUri(info?.passwordChangeUrl);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.accountProviderTitle,
                style: Theme.of(context).textTheme.titleSmall,
              ),
              const SizedBox(height: 8),
              Text(l10n.accountProviderExplanation),
              const SizedBox(height: 12),
              _detail(context, l10n.accountHandleLabel, info?.handle ?? handle),
              _detail(context, l10n.accountDidLabel, did),
              if (info?.hostingProvider case final provider?)
                _detail(context, l10n.accountHostingLabel, provider),
              if (info?.pdsUrl case final pds?) SelectableText(pds),
              if (state.isLoading) const LinearProgressIndicator(),
              if (state.hasError ||
                  (!state.isLoading && info?.pdsUrl == null)) ...[
                Text(l10n.accountProviderUnavailable),
                TextButton(
                  onPressed: () => ref.invalidate(accountHostingProvider),
                  child: Text(l10n.retryButton),
                ),
              ],
            ],
          ),
        ),
        if (manage != null)
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.manageProvider,
              kind: SettingsRowKind.externalLink,
            ),
            label: l10n.accountManageProvider,
            leading: CraftskyIcons.accountSettings,
            onTap: () => _open(context, manage),
          ),
        if (password != null)
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.changeProviderPassword,
              kind: SettingsRowKind.externalLink,
            ),
            label: l10n.accountChangePassword,
            leading: CraftskyIcons.externalLink,
            onTap: () => _open(context, password),
          ),
        if (manage != null)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(
              password == null
                  ? l10n.accountProviderAvailabilityGeneric
                  : l10n.accountProviderAvailability,
            ),
          ),
        const Divider(),
      ],
    );
  }

  Widget _detail(BuildContext context, String label, String value) => Row(
    children: [
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: Theme.of(context).textTheme.labelMedium),
            SelectableText(value),
          ],
        ),
      ),
      IconButton(
        tooltip: '${AppLocalizations.of(context).accountCopyDetail}: $label',
        icon: const Icon(CraftskyIcons.copy),
        onPressed: () async {
          await Clipboard.setData(ClipboardData(text: value));
          if (context.mounted) {
            context.showInfo(AppLocalizations.of(context).moderationCopied);
          }
        },
      ),
    ],
  );

  Future<void> _open(BuildContext context, Uri uri) async {
    if (await tryLaunchSettingsLink(uri, linkLauncher) || !context.mounted) {
      return;
    }
    context.showError(AppLocalizations.of(context).navigationLinkOpenError);
  }
}
