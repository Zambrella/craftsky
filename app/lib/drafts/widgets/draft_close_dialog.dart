import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';

enum DraftCloseChoice { save, discard, keepEditing }

Future<DraftCloseChoice> showDraftCloseDialog(
  BuildContext context, {
  required bool existingDraft,
  required bool canSave,
}) async {
  final l10n = AppLocalizations.of(context);
  return await showCraftskyModal<DraftCloseChoice>(
        context,
        barrierDismissible: false,
        builder: (dialogContext) => CraftskyDialog(
          title: l10n.draftCloseTitle,
          body: Text(l10n.draftCloseMessage),
          actions: [
            TextButton(
              onPressed: () =>
                  Navigator.of(dialogContext).pop(DraftCloseChoice.keepEditing),
              child: Text(l10n.draftKeepEditingAction),
            ),
            TextButton(
              style: TextButton.styleFrom(foregroundColor: BrandColors.red),
              onPressed: () =>
                  Navigator.of(dialogContext).pop(DraftCloseChoice.discard),
              child: Text(
                existingDraft
                    ? l10n.draftDiscardChangesAction
                    : l10n.draftDiscardAction,
              ),
            ),
            ChunkyButton(
              onPressed: canSave
                  ? () => Navigator.of(dialogContext).pop(DraftCloseChoice.save)
                  : null,
              child: Text(
                existingDraft
                    ? l10n.draftSaveChangesAction
                    : l10n.draftSaveAction,
              ),
            ),
          ],
        ),
      ) ??
      DraftCloseChoice.keepEditing;
}
