part of '../craftsky_select_inputs.dart';

class CraftskySelectOption<T> {
  const CraftskySelectOption({
    required this.value,
    required this.label,
    this.description,
    this.leadingBuilder,
  });

  final T value;
  final String label;
  final String? description;
  final WidgetBuilder? leadingBuilder;
}

List<CraftskySelectOption<T>> alphabetizedSelectOptions<T>(
  Iterable<CraftskySelectOption<T>> options,
) => options.toList()..sort((left, right) => left.label.compareTo(right.label));
