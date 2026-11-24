/// UI-only option metadata for project composer controls.
///
/// Project DTOs remain string-backed and open-token compatible; this class is
/// only used by Flutter controls to pair a user-facing label with a known token
/// or bounded string value.
class ProjectOption {
  const new({
    required this.value,
    required this.label,
    this.aliases = const [],
    this.group,
    this.parentValue,
  });

  final String value;
  final String label;
  final List<String> aliases;
  final String? group;
  final String? parentValue;

  static final _labelSeparator = RegExp(
    r'\s+(?:or|and)\s+|/',
    caseSensitive: false,
  );
  static final _aliasSeparator = RegExp(r'\s+or\s+|/', caseSensitive: false);

  String? get alternativeNameDescription {
    final visibleNames = {
      label.trim().toLowerCase(),
      for (final part in label.split(_labelSeparator))
        part.trim().toLowerCase(),
    };
    final names = <String>[];
    for (final alias in aliases) {
      for (final part in alias.split(_aliasSeparator)) {
        final name = part.trim();
        if (name.isNotEmpty && visibleNames.add(name.toLowerCase())) {
          names.add(name[0].toUpperCase() + name.substring(1));
        }
      }
    }
    if (names.isEmpty) return null;
    names.sort(
      (left, right) => left.toLowerCase().compareTo(right.toLowerCase()),
    );
    return names.join(' · ');
  }
}
