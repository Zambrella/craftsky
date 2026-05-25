import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.welcomeTitle)),
      body: const Center(child: _WelcomePageBody()),
    );
  }
}

class _WelcomePageBody extends ConsumerWidget {
  const _WelcomePageBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Text(l10n.welcomeTitle),
        SizedBox(height: spacing.sp5),
        ChunkyButton(
          onPressed: () => const SignInRoute().go(context),
          child: Text(l10n.welcomeSignInAction),
        ),
        SizedBox(height: spacing.sp2),
        TextButton(
          onPressed: () => const SignInRoute().go(context),
          child: Text(l10n.welcomeCreateAccountAction),
        ),
      ],
    );
  }
}
