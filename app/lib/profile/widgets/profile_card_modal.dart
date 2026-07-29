import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_card.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum _ProfileCardAction { visitProfile }

typedef _ProfileCardResult = ({_ProfileCardAction action, String handle});

/// Loads and presents a compact profile card without adding a route.
///
/// Identity surfaces should use this helper. Explicit navigation affordances,
/// deep links, and the card's own "Visit profile" action continue to use
/// [UserProfileRoute].
Future<void> showUserProfileCard(
  BuildContext context, {
  required String handleOrDid,
  Color? primaryColor,
  ProfileBackgroundIllustration? backgroundIllustration,
  ProfileAvatarFrame? avatarFrame,
}) async {
  final result = await showCraftskyModal<_ProfileCardResult>(
    context,
    builder: (dialogContext) => _ProfileCardModalHost(
      handleOrDid: handleOrDid,
      primaryColor: primaryColor,
      backgroundIllustration: backgroundIllustration,
      avatarFrame: avatarFrame,
    ),
  );
  if (!context.mounted || result == null) return;

  switch (result.action) {
    case _ProfileCardAction.visitProfile:
      await UserProfileRoute(handle: result.handle).push<void>(context);
  }
}

class _ProfileCardModalHost extends ConsumerWidget {
  const _ProfileCardModalHost({
    required this.handleOrDid,
    required this.primaryColor,
    required this.backgroundIllustration,
    required this.avatarFrame,
  });

  final String handleOrDid;
  final Color? primaryColor;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(userProfileProvider(handleOrDid));
    final toggleState = ref.watch(toggleFollowProfileProvider);
    final auth = ref.watch(authSessionProvider).value;
    ref.listen(toggleFollowProfileProvider, (previous, next) {
      if (previous is AsyncLoading && next is AsyncError) {
        context.showError(
          AppLocalizations.of(context).profileFollowToggleError,
        );
        ref.read(toggleFollowProfileProvider.notifier).reset();
      }
    });

    return switch (profileAsync) {
      AsyncValue(:final value?) => ProfileCard(
        profile: value,
        primaryColor: primaryColor,
        backgroundIllustration: backgroundIllustration,
        avatarFrame: avatarFrame,
        isOwnProfile: _isOwnProfile(auth, value.did.toString()),
        isPrimaryActionBusy: toggleState.isLoading,
        onClose: () => Navigator.of(context).pop(),
        onVisitProfile: () => Navigator.of(context).pop((
          action: _ProfileCardAction.visitProfile,
          handle: value.handle.toString(),
        )),
        onPrimaryAction: () {
          unawaited(
            ref
                .read(toggleFollowProfileProvider.notifier)
                .toggle(cacheKey: handleOrDid, profile: value),
          );
        },
      ),
      AsyncError() => CraftskyDialog(
        title: AppLocalizations.of(context).profileLoadErrorTitle,
        body: const SizedBox.shrink(),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(MaterialLocalizations.of(context).closeButtonLabel),
          ),
          ChunkyButton(
            onPressed: () => ref.invalidate(userProfileProvider(handleOrDid)),
            child: Text(AppLocalizations.of(context).profileLoadErrorRetry),
          ),
        ],
      ),
      _ => const Center(child: StitchProgressIndicator()),
    };
  }

  bool _isOwnProfile(AuthState? auth, String profileDid) {
    return switch (auth) {
      SignedIn(:final did) => did.toString() == profileDid,
      _ => false,
    };
  }
}
