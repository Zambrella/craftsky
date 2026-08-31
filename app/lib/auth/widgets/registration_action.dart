import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class RegistrationAction extends StatelessWidget {
  const RegistrationAction({
    required this.onPressed,
    this.isLoading = false,
    this.enabled = true,
    super.key,
  });

  final VoidCallback onPressed;
  final bool isLoading;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();

    return Column(
      children: [
        Text(l10n.registrationProviderDisclosure),
        SizedBox(height: spacing.sp2),
        TextButton(
          onPressed: enabled && !isLoading ? onPressed : null,
          child: isLoading
              ? const StitchProgressIndicator(size: 18)
              : Text(l10n.welcomeCreateAccountAction),
        ),
      ],
    );
  }
}
