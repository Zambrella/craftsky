import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:flutter/material.dart';

/// Generic empty/placeholder body for tabs that don't have data wiring
/// yet (Projects, Reposts, Saved). Returns a [SliverFillRemaining] so
/// it fills whatever scrollable space is left below the header chrome.
/// Per the design-system voice, copy is warm rather than apologetic.
class ProfileEmptyTab extends StatelessWidget {
  const ProfileEmptyTab({
    required this.icon,
    required this.title,
    required this.subtitle,
    super.key,
  });

  final IconData icon;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return SliverFillRemaining(
      hasScrollBody: false,
      child: CraftskyEmptyState(
        icon: icon,
        title: title,
        subtitle: subtitle,
      ),
    );
  }
}
