import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

enum DraftCloseChoice { save, discard, keepEditing }

Future<DraftCloseChoice> showDraftCloseDialog(
  BuildContext context, {
  required bool existingDraft,
  required bool canSave,
}) async {
  final l10n = AppLocalizations.of(context);
  return await showDialog<DraftCloseChoice>(
        context: context,
        barrierDismissible: false,
        builder: (context) => AlertDialog(
          title: Text(l10n.draftCloseTitle),
          content: Text(l10n.draftCloseMessage),
          actions: [
            TextButton(
              onPressed: () =>
                  Navigator.of(context).pop(DraftCloseChoice.keepEditing),
              child: Text(l10n.draftKeepEditingAction),
            ),
            TextButton(
              onPressed: () =>
                  Navigator.of(context).pop(DraftCloseChoice.discard),
              child: Text(
                existingDraft
                    ? l10n.draftDiscardChangesAction
                    : l10n.draftDiscardAction,
              ),
            ),
            FilledButton(
              onPressed: canSave
                  ? () => Navigator.of(context).pop(DraftCloseChoice.save)
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
