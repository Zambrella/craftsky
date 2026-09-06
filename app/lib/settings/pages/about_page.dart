import 'dart:async';

import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/about_version.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/settings/settings_links.dart';
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/settings_row_tile.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AboutPage extends ConsumerWidget {
  const AboutPage({
    this.linkLauncher = launchExternalLink,
    this.version,
    this.buildNumber,
    super.key,
  });

  final ExternalLinkLauncher linkLauncher;
  final String? version;
  final String? buildNumber;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final packageInfo = ref.exists(appDependenciesProvider)
        ? ref.watch(packageInfoProvider)
        : null;
    final visibleVersion = buildVersionLabel(
      l10n,
      version: version ?? packageInfo?.version,
      buildNumber: buildNumber ?? packageInfo?.buildNumber,
    );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.settingsAbout)),
      body: ListView(
        children: [
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.terms,
              kind: SettingsRowKind.externalLink,
            ),
            label: l10n.settingsTerms,
            leading: CraftskyIcons.document,
            onTap: () => unawaited(
              _open(context, settingsTermsUri, l10n.navigationLinkOpenError),
            ),
          ),
          SettingsRowTile(
            descriptor: const SettingsRowDescriptor(
              id: SettingsRowId.privacyPolicy,
              kind: SettingsRowKind.externalLink,
            ),
            label: l10n.settingsPrivacyPolicy,
            leading: CraftskyIcons.privacy,
            onTap: () => unawaited(
              _open(context, settingsPrivacyUri, l10n.navigationLinkOpenError),
            ),
          ),
          const ClearImageCacheTile(),
          if (visibleVersion != null)
            SettingsRowTile(
              descriptor: const SettingsRowDescriptor(
                id: SettingsRowId.version,
                kind: SettingsRowKind.readOnly,
              ),
              label: l10n.settingsVersion,
              subtitle: visibleVersion,
              leading: CraftskyIcons.info,
            ),
        ],
      ),
    );
  }

  Future<void> _open(BuildContext context, Uri uri, String errorMessage) async {
    if (await tryLaunchSettingsLink(uri, linkLauncher) || !context.mounted) {
      return;
    }
    context.showError(errorMessage);
  }
}
