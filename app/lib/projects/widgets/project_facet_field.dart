import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/rich_text/facet_autocomplete_controller.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:material_ui/material_ui.dart';

class ProjectFacetField extends StatelessWidget {
  const new({
    required this.name,
    required this.label,
    required this.controller,
    required this.limit,
    super.key,
    this.editorKey,
    this.focusNode,
    this.hintText,
    this.enabled = true,
    this.textCapitalization = TextCapitalization.words,
    this.allowedTokenKinds,
    this.precedingHashtags,
    this.normalizeValue,
    this.onChanged,
  });

  final RecordTextLimit limit;
  final String name;
  final String label;
  final FacetTextEditingController controller;
  final Key? editorKey;
  final FocusNode? focusNode;
  final String? hintText;
  final bool enabled;
  final TextCapitalization textCapitalization;
  final Set<ActiveFacetTokenKind>? allowedTokenKinds;
  final List<String> Function()? precedingHashtags;
  final String? Function(String value)? normalizeValue;
  final ValueChanged<String>? onChanged;

  @override
  Widget build(BuildContext context) {
    return FormBuilderField<String>(
      name: name,
      enabled: enabled,
      validator: (value) => limit.accepts(value ?? '')
          ? null
          : AppLocalizations.of(context).textInputTooLong,
      builder: (field) {
        return FacetAutocompleteEditor(
          key: editorKey,
          label: label,
          maxLength: limit.graphemes,
          maxBytes: limit.bytes,
          errorText: field.errorText,
          hintText: hintText,
          controller: controller,
          focusNode: focusNode,
          textCapitalization: textCapitalization,
          enabled: field.widget.enabled,
          textInputAction: TextInputAction.next,
          allowedTokenKinds: allowedTokenKinds,
          precedingHashtags: precedingHashtags,
          onChanged: (value) {
            field.didChange(normalizeValue?.call(value) ?? value);
            onChanged?.call(value);
          },
        );
      },
    );
  }
}
