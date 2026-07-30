import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:flutter/material.dart';

class PostLanguageSelector extends StatelessWidget {
  const PostLanguageSelector({
    required this.selection,
    required this.onChanged,
    this.enabled = true,
    super.key,
  });

  final PostLanguageSelection selection;
  final ValueChanged<PostLanguageSelection> onChanged;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Semantics(
      label: l10n.postLanguagesSemantics,
      container: true,
      explicitChildNodes: true,
      child: Wrap(
        spacing: 8,
        runSpacing: 4,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          for (final language in selection.values)
            InputChip(
              label: Text(languageLabel(language)),
              selected: true,
              onDeleted: enabled && selection.values.length > 1
                  ? () => onChanged(selection.remove(language))
                  : null,
            ),
          ActionChip(
            avatar: const Icon(Icons.add, size: 18),
            label: Text(l10n.postLanguageAdd),
            onPressed: enabled && selection.values.length < 3
                ? () async {
                    final language = await showDialog<String>(
                      context: context,
                      builder: (context) => _LanguageSearchDialog(
                        excluded: selection.values.toSet(),
                      ),
                    );
                    if (language != null) {
                      onChanged(selection.add(language));
                    }
                  }
                : null,
          ),
          if (selection.values.length == 3) Text(l10n.postLanguageLimit),
        ],
      ),
    );
  }
}

class _LanguageSearchDialog extends StatefulWidget {
  const _LanguageSearchDialog({required this.excluded});

  final Set<String> excluded;

  @override
  State<_LanguageSearchDialog> createState() => _LanguageSearchDialogState();
}

class _LanguageSearchDialogState extends State<_LanguageSearchDialog> {
  String _query = '';

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final query = _query.trim().toLowerCase();
    final available =
        supportedLanguageTags.where((tag) {
          if (widget.excluded.contains(tag)) return false;
          if (query.isEmpty) return true;
          return tag.contains(query) ||
              languageLabel(tag).toLowerCase().contains(query);
        }).toList()..sort(
          (left, right) => languageLabel(left).compareTo(languageLabel(right)),
        );
    return AlertDialog(
      title: Text(l10n.postLanguageDialogTitle),
      content: SizedBox(
        width: 420,
        height: 420,
        child: Column(
          children: [
            TextField(
              autofocus: true,
              decoration: InputDecoration(
                labelText: l10n.languageSearchHint,
                prefixIcon: const Icon(Icons.search),
              ),
              onChanged: (value) => setState(() => _query = value),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: ListView.builder(
                itemCount: available.length,
                itemBuilder: (context, index) {
                  final language = available[index];
                  return ListTile(
                    title: Text(languageLabel(language)),
                    subtitle: Text(language),
                    onTap: () => Navigator.of(context).pop(language),
                  );
                },
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.languageCancel),
        ),
      ],
    );
  }
}
