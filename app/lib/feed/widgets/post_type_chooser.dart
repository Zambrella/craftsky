import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:flutter/material.dart';

typedef PostComposerLauncher = Future<Post?> Function(BuildContext context);

Future<Post?> showTopLevelPostComposerChooser(
  BuildContext context, {
  required RelativeRect position,
  PostComposerLauncher showRegularComposer = showPostComposerSheet,
  PostComposerLauncher showProjectComposer = showProjectComposerSheet,
}) async {
  final l10n = AppLocalizations.of(context);
  Post? selectedPost;

  await showCraftskyContextMenu(
    context,
    position: position,
    groups: [
      CraftskyContextMenuGroup(
        items: [
          CraftskyContextMenuItem(
            text: l10n.postTypeRegularLabel,
            description: l10n.postTypeRegularDescription,
            icon: Icons.notes,
            onPressed: () async {
              selectedPost = await showRegularComposer(context);
            },
          ),
          CraftskyContextMenuItem(
            text: l10n.postTypeProjectLabel,
            description: l10n.postTypeProjectDescription,
            icon: Icons.auto_awesome_mosaic_outlined,
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
