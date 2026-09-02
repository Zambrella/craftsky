import 'dart:async';

import 'package:craftsky_app/auth/models/auth_error.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class WelcomePage extends ConsumerStatefulWidget {
  const WelcomePage({this.linkLauncher = launchExternalLink, super.key});

  final ExternalLinkLauncher linkLauncher;

  @override
  ConsumerState<WelcomePage> createState() => _WelcomePageState();
}

class _WelcomePageState extends ConsumerState<WelcomePage> {
  static final Uri _termsUri = Uri.parse('https://craftsky.social/terms');
  static final Uri _privacyUri = Uri.parse('https://craftsky.social/privacy');

  final _handleController = TextEditingController();

  @override
  void dispose() {
    _handleController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    ref.listen(authControllerProvider, (previous, next) {
      if (previous case AsyncLoading()) {
        if (next case AsyncError(:final error)) {
          context.showError(_messageFor(l10n, error));
        }
      }
    });
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    final radius =
        Theme.of(context).extension<RadiusTheme>() ?? const RadiusTheme();
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final busy = ref.watch(authControllerProvider) is AsyncLoading;

    return Scaffold(
      body: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) => SingleChildScrollView(
            padding: EdgeInsets.all(spacing.sp4),
            child: ConstrainedBox(
              constraints: BoxConstraints(
                minHeight: constraints.maxHeight - spacing.sp4 * 2,
              ),
              child: Center(
                child: Container(
                  width: double.infinity,
                  constraints: const BoxConstraints(maxWidth: 560),
                  padding: EdgeInsets.symmetric(
                    horizontal: constraints.maxWidth < 420
                        ? spacing.sp4
                        : spacing.sp6,
                    vertical: spacing.sp6,
                  ),
                  decoration: BoxDecoration(
                    color: swatches.paper2,
                    border: Border.all(color: colors.onSurface, width: 1.5),
                    borderRadius: BorderRadius.circular(radius.r3),
                    boxShadow: shadows.drop,
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(
                        l10n.welcomeJoinTitle,
                        textAlign: TextAlign.center,
                        style: theme.textTheme.displaySmall,
                      ),
                      SizedBox(height: spacing.sp2),
                      Text(
                        l10n.welcomeSubtitle,
                        textAlign: TextAlign.center,
                        style: theme.textTheme.bodyLarge?.copyWith(
                          color: colors.onSurfaceVariant,
                        ),
                      ),
                      SizedBox(height: spacing.sp5),
                      ChunkyButton(
                        onPressed: busy ? null : _startRegistration,
                        child: busy
                            ? Text(l10n.welcomeRedirectingAction)
                            : Text(l10n.welcomeRegisterAction),
                      ),
                      SizedBox(height: spacing.sp2),
                      Text(
                        l10n.welcomeRegistrationHandoff,
                        textAlign: TextAlign.center,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: colors.onSurfaceVariant,
                        ),
                      ),
                      SizedBox(height: spacing.sp5),
                      _OrDivider(label: l10n.welcomeOr),
                      SizedBox(height: spacing.sp5),
                      BrandTextField(
                        label: l10n.signInHandleLabel,
                        hintText: 'your-handle.bsky.social',
                        controller: _handleController,
                        enabled: !busy,
                        keyboardType: TextInputType.url,
                        textInputAction: TextInputAction.done,
                        autofillHints: const [AutofillHints.username],
                        autocorrect: false,
                        onSubmitted: (_) => _signIn(),
                      ),
                      SizedBox(height: spacing.sp4),
                      ChunkyButton(
                        variant: ChunkyButtonVariant.secondary,
                        onPressed: busy ? null : _signIn,
                        child: Text(
                          busy
                              ? l10n.welcomeRedirectingAction
                              : l10n.welcomeSignInAction,
                        ),
                      ),
                      SizedBox(height: spacing.sp5),
                      _AtmosphereExplainer(l10n: l10n),
                      SizedBox(height: spacing.sp5),
                      _LegalLinks(
                        l10n: l10n,
                        onTerms: () => _openLink(_termsUri),
                        onPrivacy: () => _openLink(_privacyUri),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _signIn() {
    unawaited(
      ref
          .read(authControllerProvider.notifier)
          .signIn(handle: _handleController.text),
    );
  }

  void _startRegistration() {
    unawaited(ref.read(authControllerProvider.notifier).startRegistration());
  }

  Future<void> _openLink(Uri uri) async {
    var opened = false;
    try {
      opened = await widget.linkLauncher(uri);
    } on Object {
      // Platform link handlers can fail by returning false or by throwing.
    }
    if (!mounted || opened) return;
    context.showError(AppLocalizations.of(context).navigationLinkOpenError);
  }

  String _messageFor(AppLocalizations l10n, Object? error) => switch (error) {
    HandleRequired() => l10n.signInHandleRequiredError,
    InvalidHandle() => l10n.signInInvalidHandleError,
    ServerUnavailable() => l10n.signInServerUnavailableError,
    BrowserLaunchFailed() => l10n.signInBrowserLaunchError,
    RegistrationFailure.canceled => l10n.authRegistrationCanceledError,
    RegistrationFailure.providerUnavailable =>
      l10n.authRegistrationProviderUnavailableError,
    RegistrationFailure.registrationIncomplete =>
      l10n.authRegistrationIncompleteError,
    AccountLimitReached() => l10n.accountSwitcherMaximum,
    _ => l10n.signInGenericError,
  };
}

class _OrDivider extends StatelessWidget {
  const _OrDivider({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Row(
      children: [
        const Expanded(child: CraftskyDivider()),
        Padding(
          padding: EdgeInsets.symmetric(horizontal: spacing.sp3),
          child: Text(label, style: Theme.of(context).textTheme.labelSmall),
        ),
        const Expanded(child: CraftskyDivider()),
      ],
    );
  }
}

class _AtmosphereExplainer extends StatelessWidget {
  const _AtmosphereExplainer({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return ExpansionTile(
      key: const Key('atmosphere-explainer'),
      tilePadding: EdgeInsets.zero,
      childrenPadding: EdgeInsets.zero,
      shape: const Border(),
      collapsedShape: const Border(),
      title: Text(
        l10n.welcomeAtmosphereTitle,
        style: theme.textTheme.titleSmall,
      ),
      children: [
        Container(
          width: double.infinity,
          padding: EdgeInsets.all(spacing.sp4),
          color: swatches.paper3,
          child: Text(
            l10n.welcomeAtmosphereBody,
            style: theme.textTheme.bodyMedium,
          ),
        ),
      ],
    );
  }
}

class _LegalLinks extends StatefulWidget {
  const _LegalLinks({
    required this.l10n,
    required this.onTerms,
    required this.onPrivacy,
  });

  final AppLocalizations l10n;
  final VoidCallback onTerms;
  final VoidCallback onPrivacy;

  @override
  State<_LegalLinks> createState() => _LegalLinksState();
}

class _LegalLinksState extends State<_LegalLinks> {
  late final TapGestureRecognizer _termsRecognizer;
  late final TapGestureRecognizer _privacyRecognizer;

  @override
  void initState() {
    super.initState();
    _termsRecognizer = TapGestureRecognizer()..onTap = widget.onTerms;
    _privacyRecognizer = TapGestureRecognizer()..onTap = widget.onPrivacy;
  }

  @override
  void didUpdateWidget(covariant _LegalLinks oldWidget) {
    super.didUpdateWidget(oldWidget);
    _termsRecognizer.onTap = widget.onTerms;
    _privacyRecognizer.onTap = widget.onPrivacy;
  }

  @override
  void dispose() {
    _termsRecognizer.dispose();
    _privacyRecognizer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final style = theme.textTheme.bodySmall;
    final linkStyle = style?.copyWith(
      color: theme.colorScheme.primary,
      decoration: TextDecoration.underline,
      decorationColor: theme.colorScheme.primary,
    );
    return Text.rich(
      key: const Key('legal-links'),
      TextSpan(
        style: style,
        children: [
          TextSpan(text: '${widget.l10n.welcomeLegalPrefix} '),
          TextSpan(
            text: widget.l10n.navigationTerms,
            style: linkStyle,
            recognizer: _termsRecognizer,
          ),
          TextSpan(text: ' ${widget.l10n.welcomeLegalAnd} '),
          TextSpan(
            text: widget.l10n.welcomePrivacyAction,
            style: linkStyle,
            recognizer: _privacyRecognizer,
          ),
          const TextSpan(text: '.'),
        ],
      ),
      textAlign: TextAlign.center,
    );
  }
}
