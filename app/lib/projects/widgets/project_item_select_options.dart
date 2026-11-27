import 'package:craftsky_app/projects/options/project_option.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';

List<CraftskySelectOption<String>> projectItemSelectOptions(
  String craft, {
  String? retainedType,
  String? retainedSubtype,
}) {
  final types = ProjectOptionCatalogs.projectTypesForCraft(craft);
  final parents = alphabetizedSelectOptions([
    for (final type in types) _itemOption(type),
    if (retainedType != null &&
        !types.any((type) => type.value == retainedType))
      _itemOption(_retainedOption(retainedType)),
    if (retainedType == null && retainedSubtype != null)
      _itemOption(_retainedOption(retainedSubtype, subtype: true)),
  ], lastValue: ProjectOptionCatalogs.otherProjectTypeToken);

  return [
    for (final type in parents) ...[
      type,
      ..._subtypeOptions(craft, type, retainedType, retainedSubtype),
    ],
  ];
}

List<CraftskySelectOption<String>> _subtypeOptions(
  String craft,
  CraftskySelectOption<String> type,
  String? retainedType,
  String? retainedSubtype,
) {
  if (type.value == ProjectOptionCatalogs.otherProjectTypeToken) {
    return const [];
  }
  final subtypes = ProjectOptionCatalogs.projectSubtypesFor(
    craftToken: craft,
    projectTypeToken: type.value,
  );
  return alphabetizedSelectOptions([
    for (final subtype in subtypes) _itemOption(subtype, parent: type),
    if (retainedType == type.value &&
        retainedSubtype != null &&
        !subtypes.any((subtype) => subtype.value == retainedSubtype))
      _itemOption(
        _retainedOption(retainedSubtype, subtype: true),
        parent: type,
      ),
  ]);
}

String? selectedProjectItem(
  List<CraftskySelectOption<String>> options,
  String? type,
  String? subtype,
) {
  if (subtype == null || type == ProjectOptionCatalogs.otherProjectTypeToken) {
    return type;
  }
  if (type != null &&
      options.any(
        (option) => option.parentValue == type && option.value == subtype,
      )) {
    return subtype;
  }
  return _retainedSubtypeValue(subtype);
}

CraftskySelectOption<String> _itemOption(
  ProjectOption option, {
  CraftskySelectOption<String>? parent,
}) => CraftskySelectOption(
  value: option.value,
  label: option.label,
  description: option.alternativeNameDescription,
  aliases: option.aliases,
  parentValue: parent?.value,
  selectedLabel: parent == null ? null : '${parent.label} · ${option.label}',
);

ProjectOption _retainedOption(String value, {bool subtype = false}) =>
    ProjectOption(
      value: subtype ? _retainedSubtypeValue(value) : value,
      label: ProjectOptionCatalogs.projectValueLabel(value).split('#').last,
    );

String _retainedSubtypeValue(String value) => 'retained:$value';

String projectItemFilterLabel(
  List<CraftskySelectOption<String>> options,
  String value,
) {
  final item = options.where((option) => option.value == value).firstOrNull;
  return item?.selectedLabel ??
      item?.label ??
      ProjectOptionCatalogs.projectValueLabel(value).split('#').last;
}
