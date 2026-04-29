import 'package:flutter/material.dart';

/// Profile bio text. Renders nothing when [description] is null or
/// empty — empty-state copy belongs to the visitor-facing About tab,
/// not the header.
class ProfileBio extends StatelessWidget {
  const ProfileBio({required this.description, super.key});

  final String? description;

  @override
  Widget build(BuildContext context) {
    final text = description;
    if (text == null || text.isEmpty) return const SizedBox.shrink();
    final theme = Theme.of(context);
    return Text(
      text,
      style: theme.textTheme.bodyMedium?.copyWith(
        color: theme.colorScheme.onSurface,
      ),
    );
  }
}
