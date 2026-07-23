import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/saved_posts/models/saved_post_folder.dart';
import 'package:craftsky_app/saved_posts/providers/saved_post_folders_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<SavedPostFolder?> showCreateSavedPostFolderDialog(
  BuildContext context, {
  required AccountKey account,
}) => showDialog<SavedPostFolder>(
  context: context,
  builder: (_) => _FolderNameDialog(account: account),
);

Future<SavedPostFolder?> showRenameSavedPostFolderDialog(
  BuildContext context, {
  required AccountKey account,
  required SavedPostFolder folder,
}) => showDialog<SavedPostFolder>(
  context: context,
  builder: (_) => _FolderNameDialog(account: account, folder: folder),
);

Future<bool> showDeleteSavedPostFolderDialog(
  BuildContext context, {
  required AccountKey account,
  required SavedPostFolder folder,
}) async {
  final deleteSaves = await showDialog<bool>(
    context: context,
    builder: (context) {
      final l10n = AppLocalizations.of(context);
      return AlertDialog(
        title: Text(l10n.savedPostDeleteFolder),
        content: Text(l10n.savedPostDeleteFolderBody),
        actions: [
          TextButton(
            autofocus: true,
            onPressed: () => Navigator.of(context).pop(),
            child: Text(l10n.actionCancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(l10n.savedPostKeepSaves),
          ),
          Semantics(
            button: true,
            label: l10n.savedPostDeleteSaves,
            hint: l10n.destructiveActionHint,
            excludeSemantics: true,
            child: FilledButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: Text(l10n.savedPostDeleteSaves),
            ),
          ),
        ],
      );
    },
  );
  if (deleteSaves == null || !context.mounted) return false;
  final container = ProviderScope.containerOf(context);
  final deleted = await container
      .read(savedPostFoldersProvider(account).notifier)
      .delete(folder.id, deleteSaves: deleteSaves);
  if (!deleted && context.mounted) {
    final failure = container
        .read(savedPostFoldersProvider(account))
        .value
        ?.mutationFailure;
    if (failure?.shouldPresent ?? false) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            failure!.localizedMessage(AppLocalizations.of(context)),
          ),
        ),
      );
    }
  }
  return deleted;
}

class _FolderNameDialog extends ConsumerStatefulWidget {
  const _FolderNameDialog({required this.account, this.folder});

  final AccountKey account;
  final SavedPostFolder? folder;

  @override
  ConsumerState<_FolderNameDialog> createState() => _FolderNameDialogState();
}

class _FolderNameDialogState extends ConsumerState<_FolderNameDialog> {
  late final _controller = TextEditingController(text: widget.folder?.name);
  bool _pending = false;
  bool _hasError = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AlertDialog(
      title: Text(
        widget.folder == null
            ? l10n.savedPostNewFolder
            : l10n.savedPostRenameFolder,
      ),
      content: TextField(
        controller: _controller,
        enabled: !_pending,
        autofocus: true,
        decoration: InputDecoration(
          labelText: l10n.savedPostFolderNameHint,
          errorText: _hasError ? l10n.savedPostCreateFolderError : null,
        ),
        onChanged: (_) {
          if (_hasError) setState(() => _hasError = false);
        },
      ),
      actions: [
        TextButton(
          onPressed: _pending ? null : () => Navigator.of(context).pop(),
          child: Text(l10n.actionCancel),
        ),
        FilledButton(
          onPressed: _pending ? null : _submit,
          child: _pending
              ? const SizedBox.square(
                  dimension: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(
                  widget.folder == null
                      ? l10n.savedPostCreateFolderAction
                      : l10n.savedPostRenameFolder,
                ),
        ),
      ],
    );
  }

  Future<void> _submit() async {
    String name;
    try {
      name = normalizeSavedPostFolderName(_controller.text);
    } on SavedPostFolderNameException {
      setState(() => _hasError = true);
      return;
    }
    setState(() {
      _pending = true;
      _hasError = false;
    });
    final notifier = ref.read(
      savedPostFoldersProvider(widget.account).notifier,
    );
    final folder = widget.folder == null
        ? await notifier.create(name)
        : await notifier.rename(widget.folder!.id, name);
    if (!mounted) return;
    if (folder == null) {
      final failure = ref
          .read(savedPostFoldersProvider(widget.account))
          .value
          ?.mutationFailure;
      setState(() {
        _pending = false;
        _hasError = failure?.shouldPresent ?? true;
      });
      return;
    }
    Navigator.of(context).pop(folder);
  }
}
