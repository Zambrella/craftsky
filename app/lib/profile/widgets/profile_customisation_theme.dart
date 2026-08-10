import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:flutter/material.dart';

/// Applies one fixed, audited profile colour bundle without generating hues
/// at runtime. The caller controls the boundary by choosing where to insert
/// this inherited theme.
class ProfileCustomisationTheme extends StatelessWidget {
  const ProfileCustomisationTheme({
    required this.child,
    this.customisation = ProfileCustomisation.defaults,
    super.key,
  });

  final Widget child;
  final ProfileCustomisation customisation;

  @override
  Widget build(BuildContext context) {
    final parent = Theme.of(context);
    final bundle =
        profileColourBundles[customisation.colour] ??
        profileColourBundles[ProfileCustomisation.defaults.colour]!;
    final base = profileColour(bundle.base);
    final foreground = profileColour(bundle.foreground);
    final hover = profileColour(bundle.hover);
    final pressed = profileColour(bundle.pressed);
    final soft = profileColour(bundle.softContainer);
    final ink = profileColour('#111318');
    final buttonBackground = WidgetStateProperty.resolveWith<Color?>((states) {
      if (states.contains(WidgetState.disabled)) {
        return base.withValues(alpha: 0.38);
      }
      if (states.contains(WidgetState.pressed)) return pressed;
      if (states.contains(WidgetState.hovered) ||
          states.contains(WidgetState.focused)) {
        return hover;
      }
      return base;
    });
    final accentForeground = WidgetStateProperty.resolveWith<Color?>((states) {
      if (states.contains(WidgetState.disabled)) {
        return foreground.withValues(alpha: 0.5);
      }
      return foreground;
    });

    return Theme(
      data: parent.copyWith(
        extensions: [
          ...parent.extensions.values.where(
            (extension) => extension is! ChunkyButtonColourTheme,
          ),
          ChunkyButtonColourTheme(
            base: base,
            foreground: foreground,
            hover: hover,
            pressed: pressed,
            softContainer: soft,
            onSoftContainer: ink,
          ),
        ],
        colorScheme: parent.colorScheme.copyWith(
          primary: base,
          onPrimary: foreground,
          primaryContainer: soft,
          onPrimaryContainer: ink,
        ),
        filledButtonTheme: FilledButtonThemeData(
          style: ButtonStyle(
            backgroundColor: buttonBackground,
            foregroundColor: accentForeground,
          ),
        ),
        elevatedButtonTheme: ElevatedButtonThemeData(
          style: ButtonStyle(
            backgroundColor: buttonBackground,
            foregroundColor: accentForeground,
          ),
        ),
        textButtonTheme: TextButtonThemeData(
          style: ButtonStyle(
            foregroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.pressed)) return pressed;
              if (states.contains(WidgetState.hovered) ||
                  states.contains(WidgetState.focused)) {
                return hover;
              }
              return base;
            }),
          ),
        ),
        iconButtonTheme: IconButtonThemeData(
          style: ButtonStyle(
            backgroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.disabled)) {
                return soft.withValues(alpha: 0.38);
              }
              if (states.contains(WidgetState.pressed)) return pressed;
              if (states.contains(WidgetState.hovered) ||
                  states.contains(WidgetState.focused)) {
                return hover;
              }
              return soft;
            }),
            foregroundColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.pressed) ||
                  states.contains(WidgetState.hovered) ||
                  states.contains(WidgetState.focused)) {
                return foreground;
              }
              return ink;
            }),
          ),
        ),
      ),
      child: child,
    );
  }
}

Color profileColour(String hex) =>
    Color(int.parse(hex.substring(1), radix: 16) | 0xFF000000);
