import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:material_ui/material_ui.dart';

class ProjectFacetField extends StatefulWidget {
  const new({
    required this.name,
    required this.label,
    required this.controller,
    super.key,
    this.editorKey,
    this.focusNode,
    this.hintText,
    this.enabled = true,
    this.textCapitalization = TextCapitalization.words,
    this.initialDisplayText,
    this.allowedTokenKinds,
    this.normalizeValue,
    this.onChanged,
  });

  final String name;
  final String label;
  final FacetTextEditingController controller;
  final Key? editorKey;
  final FocusNode? focusNode;
  final String? hintText;
  final bool enabled;
  final TextCapitalization textCapitalization;
  final String? initialDisplayText;
  final Set<ActiveFacetTokenKind>? allowedTokenKinds;
  final String? Function(String value)? normalizeValue;
  final ValueChanged<String>? onChanged;

  @override
  State<ProjectFacetField> createState() => _ProjectFacetFieldState();
}

class _ProjectFacetFieldState extends State<ProjectFacetField> {
  @override
  void initState() {
    super.initState();
    if (widget.initialDisplayText case final initialDisplayText?
        when widget.controller.text.isEmpty) {
      widget.controller.text = initialDisplayText;
      widget.controller.selection = TextSelection.collapsed(
        offset: initialDisplayText.length,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<String>(
      name: widget.name,
      enabled: widget.enabled,
      builder: (field) {
        return FacetAutocompleteEditor(
          key: widget.editorKey,
          label: widget.label,
          hintText: widget.hintText,
          controller: widget.controller,
          focusNode: widget.focusNode,
          textCapitalization: widget.textCapitalization,
          enabled: field.widget.enabled,
          textInputAction: TextInputAction.next,
          allowedTokenKinds: widget.allowedTokenKinds,
          onChanged: (value) {
            field.didChange(widget.normalizeValue?.call(value) ?? value);
            widget.onChanged?.call(value);
          },
        );
      },
    );
  }
}
