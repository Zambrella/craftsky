/// UI-only option metadata for project composer controls.
///
/// Project DTOs remain string-backed and open-token compatible; this class is
/// only used by Flutter controls to pair a user-facing label with a known token
/// or bounded string value.
class ProjectOption {
  const ProjectOption({
    required this.value,
    required this.label,
    this.description,
    this.group,
    this.parentValue,
  });

  final String value;
  final String label;
  final String? description;
  final String? group;
  final String? parentValue;
}
