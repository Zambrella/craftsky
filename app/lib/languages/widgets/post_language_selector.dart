import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
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
            avatar: const Icon(CraftskyIconsBold.add, size: 18),
            label: Text(l10n.postLanguageAdd),
            onPressed: enabled && selection.values.length < 3
                ? () async {
                    final language = await showCraftskyModal<String>(
                      context,
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
  final _focusNode = FocusNode();
  String _query = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _focusNode.requestFocus();
    });
  }

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
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
    final viewportHeight =
        MediaQuery.sizeOf(context).height -
        MediaQuery.viewInsetsOf(context).bottom;
    final contentHeight = (viewportHeight - 300).clamp(180.0, 420.0);

    return CraftskyDialog(
      title: l10n.postLanguageDialogTitle,
      body: SizedBox(
        height: contentHeight,
        child: Column(
          children: [
            CraftskyTextInput(
              label: l10n.languageSearchHint,
              focusNode: _focusNode,
              textInputAction: TextInputAction.search,
              onChanged: (value) => setState(() => _query = value),
            ),
            SizedBox(height: spacing.sp3),
            Expanded(
              child: ListView.builder(
                padding: EdgeInsets.zero,
                itemCount: available.length,
                itemBuilder: (context, index) {
                  final language = available[index];
                  return Material(
                    color: Colors.transparent,
                    child: ListTile(
                      title: Text(languageLabel(language)),
                      subtitle: Text(language),
                      onTap: () => Navigator.of(context).pop(language),
                    ),
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
