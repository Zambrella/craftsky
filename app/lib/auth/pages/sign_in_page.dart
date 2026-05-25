import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignInPage extends ConsumerStatefulWidget {
  const SignInPage({super.key});

  @override
  ConsumerState<SignInPage> createState() => _SignInPageState();
}

class _SignInPageState extends ConsumerState<SignInPage> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(authControllerProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncError(:final error)):
          context.showError(_messageFor(l10n, error));
        case _:
          break;
      }
    });

    final state = ref.watch(authControllerProvider);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();

    return Scaffold(
      appBar: AppBar(title: Text(l10n.signInTitle)),
      body: Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            BrandTextField(
              label: l10n.signInHandleLabel,
              hintText: 'alice.bsky.social',
              controller: _controller,
              onSubmitted: (_) => _submit(),
            ),
            SizedBox(height: spacing.sp5),
            ChunkyButton(
              onPressed: state is AsyncLoading ? null : _submit,
              child: state is AsyncLoading
                  ? const StitchProgressIndicator(size: 18)
                  : Text(l10n.signInContinueAction),
            ),
          ],
        ),
      ),
    );
  }

  void _submit() {
    unawaited(
      ref
          .read(authControllerProvider.notifier)
          .signIn(handle: _controller.text),
    );
  }

  String _messageFor(AppLocalizations l10n, Object? error) => switch (error) {
    HandleRequired() => l10n.signInHandleRequiredError,
    InvalidHandle() => l10n.signInInvalidHandleError,
    ServerUnavailable() => l10n.signInServerUnavailableError,
    BrowserLaunchFailed() => l10n.signInBrowserLaunchError,
    _ => l10n.signInGenericError,
  };
}
