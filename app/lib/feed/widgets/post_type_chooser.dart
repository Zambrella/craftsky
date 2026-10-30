import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

typedef PostComposerLauncher = Future<Post?> Function(BuildContext context);

Future<Post?> showTopLevelPostComposerChooser(
  BuildContext context, {
  required RelativeRect position,
  PostComposerLauncher showRegularComposer = showPostComposerSheet,
  PostComposerLauncher showProjectComposer = showProjectComposerSheet,
}) async {
  final l10n = AppLocalizations.of(context);
  final container = ProviderScope.containerOf(context, listen: false);
  final account = container.read(authSessionProvider).value;
  if (account is SignedIn) {
    final preferences = container.read(sharedPreferencesProvider);
    final seenKey = 'post_type_explainer.seen.${account.did.value}';
    if (!(preferences.getBool(seenKey) ?? false)) {
      final type = await showCraftskyModal<_PostType>(
        context,
        builder: (dialogContext) => _PostTypeExplainer(
          onSelected: (type) => Navigator.of(dialogContext).pop(type),
        ),
      );
      if (type == null || !context.mounted) return null;
      // An account switch while the modal is open must not launch a composer
      // for a different account or consume that account's first-use explainer.
      final currentAccount = container.read(authSessionProvider).value;
      if (currentAccount is! SignedIn || currentAccount.did != account.did) {
        return null;
      }
      await preferences.setBool(seenKey, true);
      if (!context.mounted) return null;
      final launchAccount = container.read(authSessionProvider).value;
      if (launchAccount is! SignedIn || launchAccount.did != account.did) {
        return null;
      }
      return await switch (type) {
        _PostType.regular => showRegularComposer(context),
        _PostType.project => showProjectComposer(context),
      };
    }
  }
  Post? selectedPost;

  await showCraftskyContextMenu(
    context,
    position: position,
    groups: [
      CraftskyContextMenuGroup(
        items: [
          CraftskyContextMenuItem(
            key: keys.feed.regularPost,
            text: l10n.postTypeRegularLabel,
            description: l10n.postTypeRegularDescription,
            icon: CraftskyIconsBold.textPost,
            onPressed: () async {
              selectedPost = await showRegularComposer(context);
            },
          ),
          CraftskyContextMenuItem(
            text: l10n.postTypeProjectLabel,
            description: l10n.postTypeProjectDescription,
            icon: CraftskyIconsBold.projectPost,
            onPressed: () async {
              selectedPost = await showProjectComposer(context);
            },
          ),
        ],
      ),
    ],
  );

  return selectedPost;
}

enum _PostType { regular, project }

class _PostTypeExplainer extends StatelessWidget {
  const new({required this.onSelected});

  final ValueChanged<_PostType> onSelected;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return CraftskyDialog(
      title: l10n.postTypeExplainerTitle,
      body: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _PostTypeOption(
            key: keys.feed.regularPost,
            title: l10n.postTypeRegularLabel,
            description: l10n.postTypeExplainerRegularDescription,
            icon: CraftskyIconsBold.textPost,
            onPressed: () => onSelected(_PostType.regular),
          ),
          const SizedBox(height: 16),
          _PostTypeOption(
            title: l10n.postTypeProjectLabel,
            description: l10n.postTypeExplainerProjectDescription,
            icon: CraftskyIconsBold.projectPost,
            onPressed: () => onSelected(_PostType.project),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.dialogCancelDefault),
        ),
      ],
    );
  }
}

class _PostTypeOption extends StatelessWidget {
  const new({
    required this.title,
    required this.description,
    required this.icon,
    required this.onPressed,
    super.key,
  });

  final String title;
  final String description;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return OutlinedButton(
      onPressed: onPressed,
      style: OutlinedButton.styleFrom(
        padding: const EdgeInsets.all(16),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: theme.textTheme.titleMedium),
                const SizedBox(height: 4),
                Text(description, style: theme.textTheme.bodyMedium),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
