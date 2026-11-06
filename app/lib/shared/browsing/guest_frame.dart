import 'dart:async';

import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/browsing/guest_session.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showGuestSignIn(BuildContext context, WidgetRef ref) async {
  final session = ref.read(guestSessionProvider);
  await showDialog<void>(
    context: context,
    builder: (dialogContext) => Dialog(
      clipBehavior: Clip.antiAlias,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxWidth: 640,
          maxHeight: MediaQuery.sizeOf(context).height * .9,
        ),
        child: WelcomePage(
          onExplore: () => Navigator.of(dialogContext).pop(),
          onStartAuth: () => session.prepareSignIn(
            GoRouter.of(context).routeInformationProvider.value.uri.toString(),
          ),
        ),
      ),
    ),
  );
  session.explore();
}

/// Public chrome has no account runtime, drawers, notifications or composer.
class GuestFrame extends ConsumerStatefulWidget {
  const new({required this.child, super.key});
  final Widget child;
  @override
  ConsumerState<GuestFrame> createState() => _GuestFrameState();
}

class _GuestFrameState extends ConsumerState<GuestFrame> {
  bool _scheduled = false;
  bool _authScheduled = false;
  @override
  Widget build(BuildContext context) {
    final delegate = GoRouter.of(context).routerDelegate;
    return ListenableBuilder(
      listenable: delegate,
      builder: (context, _) =>
          _buildLocation(context, delegate.currentConfiguration.uri),
    );
  }

  Widget _buildLocation(BuildContext context, Uri uri) {
    final explicit = uri.queryParameters['auth'] == '1';
    if (!explicit) _authScheduled = false;
    if ((!_scheduled &&
            ref.read(guestSessionProvider).shouldPrompt(uri.path)) ||
        (explicit && !_authScheduled)) {
      _scheduled = true;
      _authScheduled = explicit;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (explicit) const FeedRoute().go(context);
        unawaited(showGuestSignIn(context, ref));
      });
    }
    return Scaffold(
      body: Column(
        children: [
          Expanded(child: widget.child),
          SafeArea(
            top: false,
            child: Semantics(
              container: true,
              label: 'Sign in to CraftSky',
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 8,
                ),
                child: Wrap(
                  alignment: WrapAlignment.center,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  spacing: 12,
                  children: [
                    const Text('Join the crafting community'),
                    FilledButton(
                      onPressed: () => showGuestSignIn(context, ref),
                      child: const Text('Sign in / Register'),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
