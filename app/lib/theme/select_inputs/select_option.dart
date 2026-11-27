part of '../craftsky_select_inputs.dart';

class CraftskySelectOption<T> {
  const new({
    required this.value,
    required this.label,
    this.description,
    this.aliases = const [],
    this.group,
    this.parentValue,
    this.selectedLabel,
    this.leadingBuilder,
  });

  final T value;
  final String label;
  final String? description;
  final List<String> aliases;
  final String? group;
  final T? parentValue;
  final String? selectedLabel;
  final WidgetBuilder? leadingBuilder;
}

List<CraftskySelectOption<T>> alphabetizedSelectOptions<T>(
  Iterable<CraftskySelectOption<T>> options, {
  T? lastValue,
}) {
  final sorted = options.toList()
    ..sort(
      (left, right) =>
          left.label.toLowerCase().compareTo(right.label.toLowerCase()),
    );
  if (lastValue != null) {
    final index = sorted.indexWhere((option) => option.value == lastValue);
    if (index >= 0) sorted.add(sorted.removeAt(index));
  }
  return sorted;
}
