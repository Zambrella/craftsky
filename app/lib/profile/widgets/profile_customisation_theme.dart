import 'package:flutter/material.dart';

enum ProfileBackgroundIllustration { botanical, yarn, patchwork }

enum ProfileAvatarFrame { stitched, scalloped, braidedYarn }

/// Applies a user's profile colour to every themed element in a profile
/// surface while preserving the app's typography and theme extensions.
class ProfileCustomisationTheme extends StatelessWidget {
  const ProfileCustomisationTheme({
    required this.child,
    this.primaryColor,
    super.key,
  });

  final Widget child;
  final Color? primaryColor;

  @override
  Widget build(BuildContext context) {
    final parentTheme = Theme.of(context);
    final primary = primaryColor ?? parentTheme.colorScheme.primary;
    final generatedScheme = ColorScheme.fromSeed(
      seedColor: primary,
      brightness: parentTheme.brightness,
    );
    return Theme(
      data: parentTheme.copyWith(
        colorScheme: parentTheme.colorScheme.copyWith(
          primary: primary,
          onPrimary: generatedScheme.onPrimary,
          primaryContainer: generatedScheme.primaryContainer,
          onPrimaryContainer: generatedScheme.onPrimaryContainer,
        ),
      ),
      child: child,
    );
  }
}
