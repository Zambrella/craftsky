import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// A branded confirm/alert dialog. Paper-cutout aesthetic: thick ink border,
/// chunky `r3` corners, hard-offset drop shadow drawn via stacked layers (the
/// same approach used by `ChunkyButton`).
///
/// Most callers should reach for [showCraftskyConfirmDialog],
/// [showCraftskyDestructiveConfirmDialog], or [showCraftskyAlertDialog]
/// rather than constructing this widget directly.
class CraftskyDialog extends StatelessWidget {
  const CraftskyDialog({
    required this.title,
    required this.body,
    required this.actions,
    super.key,
  });

  final String title;
  final Widget body;
  final List<Widget> actions;

  /// Maximum width on wide screens. Below this, the dialog tracks the
  /// available width minus [_horizontalInset] on each side.
  static const double _maxWidth = 360;

  /// Horizontal inset reserved on small screens so the 10px shadow never
  /// touches the edge.
  static const double _horizontalInset = 24;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    final shadowOffset = shadows.dropLg.first.offset;
    final shadowColor = shadows.dropLg.first.color;
    final radius = BorderRadius.circular(radii.r3);

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: _horizontalInset,
          vertical: _horizontalInset,
        ),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: _maxWidth),
          child: IntrinsicHeight(
            child: Stack(
              children: [
                Positioned.fill(
                  child: Transform.translate(
                    offset: shadowOffset,
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: shadowColor,
                        borderRadius: radius,
                      ),
                    ),
                  ),
                ),
                Material(
                  color: Colors.transparent,
                  child: Container(
                    decoration: BoxDecoration(
                      color: swatches.paper3,
                      borderRadius: radius,
                      border: Border.all(color: colors.onSurface, width: 1.5),
                    ),
                    padding: EdgeInsets.all(spacing.sp5),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(title, style: theme.textTheme.titleLarge),
                        SizedBox(height: spacing.sp4),
                        DefaultTextStyle.merge(
                          style: theme.textTheme.bodyMedium,
                          child: body,
                        ),
                        SizedBox(height: spacing.sp5),
                        Wrap(
                          alignment: WrapAlignment.end,
                          spacing: spacing.sp2,
                          runSpacing: spacing.sp2,
                          children: actions,
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Configuration for a single button on a [CraftskyDialog]. Used by the
/// `show…Dialog` helpers; not consumed by [CraftskyDialog] itself.
class CraftskyDialogAction {
  const CraftskyDialogAction({
    required this.label,
    this.onPressed,
    this.isPrimary = false,
    this.isDestructive = false,
  });

  /// Visible button label.
  final String label;

  /// Tap handler. May be sync or async. `null` disables the button.
  final FutureOr<void> Function()? onPressed;

  /// Renders as a filled [ChunkyButton] when true; otherwise a [TextButton].
  final bool isPrimary;

  /// When [isPrimary] is also true, swaps the surface to [BrandColors.red]
  /// for delete/sign-out style actions.
  final bool isDestructive;
}

/// Shows a neutral two-button confirm dialog. Resolves to `true` if the user
/// taps the confirm action, `false` if they cancel, dismiss the barrier, or
/// hit the system back button.
///
/// If [onConfirm] is provided, the primary button shows a loading spinner
/// while the future completes; on success, the dialog pops with `true`. If
/// [onConfirm] throws, the dialog stays open, both buttons re-enable, and
/// the error rethrows so the caller's existing error-handling path runs.
Future<bool> showCraftskyConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
}) {
  return _showConfirmDialog(
    context,
    title: title,
    message: message,
    confirmLabel: confirmLabel,
    cancelLabel: cancelLabel,
    onConfirm: onConfirm,
    isDestructive: false,
  );
}

/// Shows a single-button informational dialog. Resolves when the user taps
/// the dismiss button or the modal barrier.
Future<void> showCraftskyAlertDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? dismissLabel,
}) async {
  final l10n = AppLocalizations.of(context);
  final theme = Theme.of(context);
  final durations = theme.extension<DurationTheme>()!;
  await showGeneralDialog<void>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (dialogContext, _, _) => CraftskyDialog(
      title: title,
      body: Text(message),
      actions: [
        ChunkyButton(
          backgroundColor: theme.colorScheme.primary,
          onPressed: () => Navigator.of(dialogContext).pop(),
          child: Text(dismissLabel ?? l10n.dialogOkDefault),
        ),
      ],
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: durations.easePop,
      );
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1).animate(curved),
          child: child,
        ),
      );
    },
  );
}

/// Shows a destructive two-button confirm dialog. Identical to
/// [showCraftskyConfirmDialog] except the primary action surface is
/// [BrandColors.red] for delete-style flows.
Future<bool> showCraftskyDestructiveConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
}) {
  return _showConfirmDialog(
    context,
    title: title,
    message: message,
    confirmLabel: confirmLabel,
    cancelLabel: cancelLabel,
    onConfirm: onConfirm,
    isDestructive: true,
  );
}

Future<bool> _showConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  required String? confirmLabel,
  required String? cancelLabel,
  required Future<void> Function()? onConfirm,
  required bool isDestructive,
}) async {
  final l10n = AppLocalizations.of(context);
  final durations = Theme.of(context).extension<DurationTheme>()!;
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (_, _, _) => _AsyncConfirmDialogHost(
      title: title,
      message: message,
      confirmLabel: confirmLabel ?? l10n.dialogConfirmDefault,
      cancelLabel: cancelLabel ?? l10n.dialogCancelDefault,
      onConfirm: onConfirm,
      isDestructive: isDestructive,
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: durations.easePop,
      );
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1).animate(curved),
          child: child,
        ),
      );
    },
  );
  return result ?? false;
}

class _AsyncConfirmDialogHost extends StatefulWidget {
  const _AsyncConfirmDialogHost({
    required this.title,
    required this.message,
    required this.confirmLabel,
    required this.cancelLabel,
    required this.onConfirm,
    required this.isDestructive,
  });

  final String title;
  final String message;
  final String confirmLabel;
  final String cancelLabel;
  final Future<void> Function()? onConfirm;
  final bool isDestructive;

  @override
  State<_AsyncConfirmDialogHost> createState() =>
      _AsyncConfirmDialogHostState();
}

class _AsyncConfirmDialogHostState extends State<_AsyncConfirmDialogHost> {
  bool _isConfirming = false;

  Future<void> _handleConfirm() async {
    final onConfirm = widget.onConfirm;
    if (onConfirm == null) {
      Navigator.of(context).pop(true);
      return;
    }

    setState(() => _isConfirming = true);
    var threw = false;
    try {
      await onConfirm();
    } on Object catch (_) {
      // Swallow the error here so it doesn't crash the widget tree as an
      // unhandled async exception. The contract is that the caller observes
      // failure via their existing channel (typically a Riverpod `ref.listen`
      // on the mutation provider being invoked) — this widget's only job is
      // to keep the dialog open and re-enable interaction so the user can
      // retry or cancel.
      threw = true;
    }

    if (!mounted) return;

    if (threw) {
      setState(() => _isConfirming = false);
      return;
    }

    Navigator.of(context).pop(true);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primaryBackground = widget.isDestructive
        ? BrandColors.red
        : theme.colorScheme.primary;

    return PopScope(
      canPop: !_isConfirming,
      child: CraftskyDialog(
        title: widget.title,
        body: Text(widget.message),
        actions: [
          TextButton(
            onPressed: _isConfirming
                ? null
                : () => Navigator.of(context).pop(false),
            child: Text(widget.cancelLabel),
          ),
          ChunkyButton(
            backgroundColor: primaryBackground,
            onPressed: _isConfirming ? null : _handleConfirm,
            child: _isConfirming
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                    ),
                  )
                : Text(widget.confirmLabel),
          ),
        ],
      ),
    );
  }
}
