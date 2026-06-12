part of '../craftsky_select_inputs.dart';

class CraftskySelectOption<T> {
  const CraftskySelectOption({
    required this.value,
    required this.label,
    this.description,
  });

  final T value;
  final String label;
  final String? description;
}
