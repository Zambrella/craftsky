import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

/// Opens an external link.
typedef ExternalLinkLauncher = Future<bool> Function(Uri uri);

/// Confirms whether an external link should open.
typedef ExternalLinkConfirmer =
    Future<bool> Function(
      BuildContext context,
      Uri uri,
    );

/// Parses user-provided link text into an HTTP(S) URI CraftSky can open.
Uri? normalizeExternalLinkUri(String? value) {
  final trimmed = value?.trim();
  if (trimmed == null || trimmed.isEmpty) return null;
  final parsed = Uri.tryParse(trimmed);
  if (parsed == null) return null;
  final withScheme = parsed.hasScheme
      ? parsed
      : Uri.tryParse('https://$trimmed');
  if (withScheme == null) return null;
  if (withScheme.scheme != 'http' && withScheme.scheme != 'https') return null;
  if (withScheme.host.isEmpty) return null;
  return withScheme;
}

/// User-facing link label that hides scheme, query strings, and fragments.
String displayExternalLink(Uri uri) {
  final buffer = StringBuffer(uri.host);
  var path = uri.path;
  if (path.endsWith('/') && path.length > 1) {
    path = path.substring(0, path.length - 1);
  }
  if (path.isNotEmpty && path != '/') {
    buffer.write(path);
  }
  return buffer.toString();
}

/// Opens [uri] in the platform browser or app handler.
Future<bool> launchExternalLink(Uri uri) {
  return url_launcher.launchUrl(
    uri,
    mode: url_launcher.LaunchMode.externalApplication,
  );
}

/// Shows the standard CraftSky external-link confirmation dialog.
Future<bool> showOpenLinkDialog(BuildContext context, Uri uri) async {
  final theme = Theme.of(context);
  final l10n = AppLocalizations.of(context);
  final spacing = theme.extension<SpacingTheme>()!;
  final durations = theme.extension<DurationTheme>()!;
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (dialogContext, _, _) => CraftskyDialog(
      title: l10n.externalLinkConfirmTitle,
      body: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.externalLinkConfirmBody),
          SizedBox(height: spacing.sp3),
          SelectableText(
            uri.toString(),
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(dialogContext).pop(false),
          child: Text(l10n.actionCancel),
        ),
        ChunkyButton(
          onPressed: () => Navigator.of(dialogContext).pop(true),
          child: Text(l10n.externalLinkConfirmAction),
        ),
      ],
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: durations.easePop,
      );
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1).animate(curved),
          child: child,
        ),
      );
    },
  );
  return result ?? false;
}

/// Confirms and then launches an external HTTP(S) link.
Future<void> confirmAndLaunchExternalLink(
  BuildContext context, {
  required Uri uri,
  required ExternalLinkLauncher launchUrl,
  ExternalLinkConfirmer confirmOpenLink = showOpenLinkDialog,
}) async {
  try {
    final confirmed = await confirmOpenLink(context, uri);
    if (!confirmed) return;
    final launched = await launchUrl(uri);
    if (!launched && context.mounted) _showExternalLinkFailure(context);
  } on Object {
    if (context.mounted) _showExternalLinkFailure(context);
  }
}

/// Accepts an AppView-hydrated destination without trimming or rewriting it.
Uri? hydratedExternalActionUri(String? value, {bool allowMailto = false}) {
  if (value == null || value.isEmpty) return null;
  final uri = Uri.tryParse(value);
  if (uri == null) return null;
  if (uri.scheme == 'https' && uri.host.isNotEmpty) return uri;
  if (allowMailto && uri.scheme == 'mailto' && uri.path.isNotEmpty) return uri;
  return null;
}

/// Confirms and launches an already validated hydrated action with safe,
/// localized failure feedback.
Future<void> confirmAndLaunchExternalAction(
  BuildContext context, {
  required Uri uri,
  required ExternalLinkLauncher launchUrl,
  ExternalLinkConfirmer confirmOpenLink = showOpenLinkDialog,
}) async {
  final hydrated = hydratedExternalActionUri(
    uri.toString(),
    allowMailto: true,
  );
  if (hydrated != uri) return;
  try {
    final confirmed = await confirmOpenLink(context, uri);
    if (!confirmed) return;
    final launched = await launchUrl(uri);
    if (!launched && context.mounted) {
      _showExternalLinkFailure(context);
    }
  } on Object {
    if (context.mounted) _showExternalLinkFailure(context);
  }
}

void _showExternalLinkFailure(BuildContext context) {
  try {
    context.showError(AppLocalizations.of(context).navigationLinkOpenError);
  } on Object {
    // Isolated widget hosts may not install the app-level messenger.
  }
}
