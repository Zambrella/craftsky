import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/widgets/project_facet_field.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_text_field.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:material_ui/material_ui.dart';

class ProjectPatternEditor extends StatefulWidget {
  const new({
    required this.pattern,
    required this.savedPattern,
    required this.nameLabel,
    required this.onChanged,
    required this.precedingHashtags,
    super.key,
  });

  final ProjectPattern pattern;
  final ProjectPattern? savedPattern;
  final String nameLabel;
  final ValueChanged<ProjectPattern> onChanged;
  final List<String> Function(String) precedingHashtags;

  @override
  State<ProjectPatternEditor> createState() => _ProjectPatternEditorState();
}

class _ProjectPatternEditorState extends State<ProjectPatternEditor> {
  late ProjectPattern _pattern = widget.pattern;
  late final _name = FacetTextEditingController(text: _pattern.name ?? '');
  late final _designer = FacetTextEditingController(
    text: _pattern.designer ?? '',
  );
  late final _publisher = FacetTextEditingController(
    text: _pattern.publisher ?? '',
  );

  @override
  void dispose() {
    _name.dispose();
    _designer.dispose();
    _publisher.dispose();
    super.dispose();
  }

  String? _text(String? value, {String? saved, String? placeholder}) {
    if (value == saved) return saved;
    final trimmed = value?.trim();
    return trimmed == null || trimmed.isEmpty || trimmed == placeholder
        ? null
        : trimmed;
  }

  void _change(ProjectPattern pattern) {
    _pattern = pattern;
    widget.onChanged(pattern);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return FormBuilder(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        spacing: 12,
        children: [
          ProjectFacetField(
            name: 'name',
            label: widget.nameLabel,
            hintText: l10n.projectComposerPatternNameHint,
            controller: _name,
            limit: RecordTextLimits.patternText,
            allowedTokenKinds: const {ActiveFacetTokenKind.hashtag},
            precedingHashtags: () => widget.precedingHashtags('name'),
            onChanged: (value) {
              final name = _text(
                value,
                saved: widget.savedPattern?.name,
                placeholder: '#',
              );
              _change(
                _pattern.copyWith(
                  name: name,
                  nameFacets: name == _pattern.name
                      ? _pattern.nameFacets
                      : null,
                ),
              );
            },
          ),
          ProjectFacetField(
            name: 'designer',
            label: l10n.projectComposerPatternDesignerLabel,
            hintText: l10n.projectComposerPatternDesignerHint,
            controller: _designer,
            limit: RecordTextLimits.patternText,
            allowedTokenKinds: const {ActiveFacetTokenKind.mention},
            precedingHashtags: () => widget.precedingHashtags('designer'),
            onChanged: (value) {
              final designer = _text(
                value,
                saved: widget.savedPattern?.designer,
              );
              _change(
                _pattern.copyWith(
                  designer: designer,
                  designerFacets: designer == _pattern.designer
                      ? _pattern.designerFacets
                      : null,
                ),
              );
            },
          ),
          ProjectFacetField(
            name: 'publisher',
            label: l10n.projectComposerPatternPublisherLabel,
            hintText: l10n.projectComposerPatternPublisherHint,
            controller: _publisher,
            limit: RecordTextLimits.patternText,
            allowedTokenKinds: const {ActiveFacetTokenKind.mention},
            precedingHashtags: () => widget.precedingHashtags('publisher'),
            onChanged: (value) {
              final publisher = _text(
                value,
                saved: widget.savedPattern?.publisher,
              );
              _change(
                _pattern.copyWith(
                  publisher: publisher,
                  publisherFacets: publisher == _pattern.publisher
                      ? _pattern.publisherFacets
                      : null,
                ),
              );
            },
          ),
          CraftskyFormBuilderTextField(
            name: 'url',
            label: l10n.projectComposerPatternUrlLabel,
            hintText: l10n.projectComposerPatternUrlHint,
            keyboardType: TextInputType.url,
            initialValue: _pattern.url,
            onChanged: (value) => _change(_pattern.copyWith(url: _text(value))),
          ),
          CraftskyFormBuilderDropdownField<String>(
            name: 'difficulty',
            label: l10n.projectComposerPatternDifficultyLabel,
            options: alphabetizedSelectOptions([
              for (final option in ProjectOptionCatalogs.patternDifficulties)
                CraftskySelectOption(value: option.value, label: option.label),
            ]),
            initialValue: _pattern.difficulty,
            onChanged: (value) => _change(_pattern.copyWith(difficulty: value)),
          ),
          FormBuilderField<bool>(
            name: 'selfDrafted',
            initialValue: _pattern.selfDrafted,
            builder: (field) => SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(
                l10n.projectComposerPatternSelfDraftedTitle,
                style: const TextStyle(fontWeight: FontWeight.bold),
              ),
              subtitle: Text(l10n.projectComposerPatternSelfDraftedDescription),
              value: field.value ?? false,
              onChanged: (value) {
                field.didChange(value);
                _change(_pattern.copyWith(selfDrafted: value));
              },
            ),
          ),
        ],
      ),
    );
  }
}
