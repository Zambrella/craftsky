import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/error_reporting_provider.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ErrorReportingConsentTile extends ConsumerWidget {
  const ErrorReportingConsentTile({this.onboarding = false, super.key});

  final bool onboarding;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final preference = ref.watch(errorReportingPreferenceProvider);
    final value =
        preference.value ??
        ref.watch(errorReportingControllerProvider).value?.optedIn ??
        false;
    Future<void> change({required bool enabled}) async {
      try {
        await ref
            .read(errorReportingPreferenceProvider.notifier)
            .setEnabled(enabled: enabled);
      } on Object {
        if (context.mounted) context.showError(l10n.errorReportingSaveError);
      }
    }

    // The nullable conditional closure needs a context type for bool inference.
    // ignore: omit_local_variable_types
    final ValueChanged<bool>? onChanged =
        preference.isLoading || preference.hasError
        ? null
        : (enabled) => unawaited(change(enabled: enabled));
    final title = Text(
      onboarding ? l10n.errorReportingOptIn : l10n.errorReportingTitle,
    );
    final subtitle = Text(l10n.errorReportingDescription);
    if (onboarding) {
      return CheckboxListTile(
        key: const Key('onboarding-error-reporting'),
        contentPadding: EdgeInsets.zero,
        controlAffinity: ListTileControlAffinity.leading,
        value: value,
        onChanged: onChanged == null
            ? null
            : (enabled) => onChanged(enabled ?? false),
        title: title,
        subtitle: subtitle,
        secondary: preference.isLoading
            ? const ActionProgress(isLoading: true, child: SizedBox.shrink())
            : null,
      );
    }
    return SwitchListTile(
      key: const Key('settings-error-reporting'),
      secondary: ActionProgress(
        isLoading: preference.isLoading,
        child: const Icon(CraftskyIcons.privacy),
      ),
      title: title,
      subtitle: subtitle,
      value: value,
      onChanged: onChanged,
    );
  }
}
