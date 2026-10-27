import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/composer/project_composer_fields.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ProjectMaterialsField extends StatelessWidget {
  const new({
    required this.name,
    required this.label,
    required this.inputHintText,
    required this.addButtonLabel,
    required this.disabledText,
    required this.maxSelectedErrorText,
    required this.maxLengthErrorText,
    required this.enabled,
    super.key,
  });

  final String name;
  final String label;
  final String inputHintText;
  final String addButtonLabel;
  final String disabledText;
  final String maxSelectedErrorText;
  final String maxLengthErrorText;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<List<ProjectMaterial>>(
      name: name,
      enabled: enabled,
      builder: (field) {
        return _MaterialsInput(
          label: label,
          values: List<ProjectMaterial>.from(field.value ?? const []),
          inputHintText: inputHintText,
          addButtonLabel: addButtonLabel,
          disabledText: disabledText,
          maxSelectedErrorText: maxSelectedErrorText,
          maxLengthErrorText: maxLengthErrorText,
          enabled: field.widget.enabled,
          onChanged: field.didChange,
        );
      },
    );
  }
}

class _MaterialsInput extends ConsumerStatefulWidget {
  const new({
    required this.label,
    required this.values,
    required this.inputHintText,
    required this.addButtonLabel,
    required this.disabledText,
    required this.maxSelectedErrorText,
    required this.maxLengthErrorText,
    required this.enabled,
    required this.onChanged,
  });

  static const maxSelected = 10;

  final String label;
  final List<ProjectMaterial> values;
  final String inputHintText;
  final String addButtonLabel;
  final String disabledText;
  final String maxSelectedErrorText;
  final String maxLengthErrorText;
  final bool enabled;
  final ValueChanged<List<ProjectMaterial>> onChanged;

  @override
  ConsumerState<_MaterialsInput> createState() => _MaterialsInputState();
}

class _MaterialsInputState extends ConsumerState<_MaterialsInput> {
  final _controller = FacetTextEditingController();
  final _focusNode = FocusNode(debugLabel: 'projectMaterials');
  String? _errorText;
  ProjectMaterial? _editingMaterial;

  bool get _canAdd => widget.enabled && _controller.text.trim().isNotEmpty;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleTextChanged);
  }

  @override
  void dispose() {
    _controller
      ..removeListener(_handleTextChanged)
      ..dispose();
    _focusNode.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant _MaterialsInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.values.length < _MaterialsInput.maxSelected &&
        oldWidget.values.length != widget.values.length) {
      _errorText = null;
    }
  }

  void _handleTextChanged() {
    if (mounted) setState(() {});
  }

  bool _isAdding = false;

  Future<void> _addCurrent() async {
    if (!widget.enabled || _isAdding) return;
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    if (!RecordTextLimits.material.accepts(text)) {
      setState(() => _errorText = widget.maxLengthErrorText);
      _focusNode.requestFocus();
      return;
    }
    if (widget.values.length >= _MaterialsInput.maxSelected) {
      setState(() => _errorText = widget.maxSelectedErrorText);
      _focusNode.requestFocus();
      return;
    }

    setState(() => _isAdding = true);
    try {
      final facets = await ref
          .read(facetGeneratorProvider)
          .generate(
            text,
            includeLinks: false,
          );
      if (!mounted) return;

      widget.onChanged([
        ...widget.values,
        ProjectMaterial(
          text: text,
          facets: facets.isEmpty ? null : facets,
          extraFields: _editingMaterial?.extraFields ?? const {},
        ),
      ]);
      _editingMaterial = null;
      _controller.clear();
      setState(() => _errorText = null);
      _focusNode.requestFocus();
    } finally {
      if (mounted) setState(() => _isAdding = false);
    }
  }

  void _remove(ProjectMaterial material) {
    if (!widget.enabled) return;
    widget.onChanged(
      List<ProjectMaterial>.from(widget.values)..remove(material),
    );
    setState(() => _errorText = null);
  }

  void _edit(ProjectMaterial material) {
    if (!widget.enabled || _isAdding || _controller.text.isNotEmpty) return;
    _editingMaterial = material;
    _controller.value = TextEditingValue(
      text: material.text,
      selection: TextSelection.collapsed(offset: material.text.length),
    );
    _remove(material);
    _focusNode.requestFocus();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        FacetAutocompleteEditor(
          maxLength: RecordTextLimits.material.graphemes,
          maxBytes: RecordTextLimits.material.bytes,
          key: const Key('${ProjectComposerFields.materials}-custom-input'),
          label: widget.label,
          hintText: widget.inputHintText,
          controller: _controller,
          textCapitalization: TextCapitalization.words,
          focusNode: _focusNode,
          enabled: widget.enabled && !_isAdding,
          errorText: _errorText,
          betweenLabelAndField: widget.values.isEmpty
              ? null
              : _MaterialEntryList(
                  values: widget.values,
                  enabled: widget.enabled && !_isAdding,
                  hasPendingText: _controller.text.isNotEmpty,
                  onEdit: _edit,
                  onRemove: _remove,
                ),
          suffixIcon: Padding(
            padding: const EdgeInsetsDirectional.only(end: 8),
            child: TextButton(
              key: const Key('${ProjectComposerFields.materials}-add-custom'),
              style: TextButton.styleFrom(
                minimumSize: Size.zero,
                padding: const EdgeInsets.symmetric(horizontal: 8),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              onPressed: _canAdd && !_isAdding ? _addCurrent : null,
              child: ActionProgress(
                isLoading: _isAdding,
                child: Text(widget.addButtonLabel),
              ),
            ),
          ),
          allowedTokenKinds: const {
            ActiveFacetTokenKind.mention,
            ActiveFacetTokenKind.hashtag,
          },
          textInputAction: TextInputAction.done,
          onSubmitted: (_) => unawaited(_addCurrent()),
        ),
        if (!widget.enabled)
          Padding(
            padding: EdgeInsets.only(top: spacing.sp2),
            child: Text(
              widget.disabledText,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
          ),
      ],
    );
  }
}

class _MaterialEntryList extends StatelessWidget {
  const new({
    required this.values,
    required this.enabled,
    required this.hasPendingText,
    required this.onEdit,
    required this.onRemove,
  });

  final List<ProjectMaterial> values;
  final bool enabled;
  final bool hasPendingText;
  final ValueChanged<ProjectMaterial> onEdit;
  final ValueChanged<ProjectMaterial> onRemove;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final colors = theme.colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final (index, material) in values.indexed) ...[
          DecoratedBox(
            decoration: BoxDecoration(
              color: colors.surface,
              border: Border.all(color: colors.outline),
              borderRadius: BorderRadius.circular(radii.r3),
            ),
            child: Padding(
              padding: EdgeInsetsDirectional.only(
                start: spacing.sp4,
                top: spacing.sp3,
                bottom: spacing.sp3,
                end: spacing.sp2,
              ),
              child: Row(
                children: [
                  Expanded(
                    child: FacetedText(
                      text: material.text,
                      facets: material.facets,
                      style: theme.textTheme.bodyLarge?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.onSurface,
                      ),
                    ),
                  ),
                  IconButton(
                    key: Key(
                      '${ProjectComposerFields.materials}-edit-'
                      '${material.text}',
                    ),
                    icon: const Icon(CraftskyIconsBold.edit),
                    tooltip: hasPendingText
                        ? AppLocalizations.of(
                            context,
                          ).projectComposerMaterialsEditBlockedHint
                        : AppLocalizations.of(
                            context,
                          ).projectComposerMaterialsEditAction,
                    visualDensity: VisualDensity.compact,
                    padding: EdgeInsets.zero,
                    constraints: const BoxConstraints.tightFor(
                      width: 32,
                      height: 32,
                    ),
                    onPressed: enabled && !hasPendingText
                        ? () => onEdit(material)
                        : null,
                  ),
                  IconButton(
                    key: Key(
                      '${ProjectComposerFields.materials}-remove-'
                      '${material.text}',
                    ),
                    icon: const Icon(CraftskyIconsBold.close),
                    tooltip: 'Remove material',
                    visualDensity: VisualDensity.compact,
                    padding: EdgeInsets.zero,
                    constraints: const BoxConstraints.tightFor(
                      width: 32,
                      height: 32,
                    ),
                    onPressed: enabled ? () => onRemove(material) : null,
                  ),
                ],
              ),
            ),
          ),
          if (index != values.length - 1) SizedBox(height: spacing.sp2),
        ],
      ],
    );
  }
}
