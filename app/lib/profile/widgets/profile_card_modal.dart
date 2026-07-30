import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// Opens a profile route in its compact summary-card presentation.
///
/// The URL changes immediately. Expanding the summary is an internal state
/// change of that same route, so Back always returns directly to the origin.
Future<void> showUserProfileCard(
  BuildContext context, {
  required String handleOrDid,
  Color? primaryColor,
  ProfileBackgroundIllustration? backgroundIllustration,
  ProfileAvatarFrame? avatarFrame,
}) {
  return GoRouter.of(context).push<void>(
    UserProfileRoute(handle: handleOrDid).location,
    extra: ProfilePresentationRequest.compact(
      primaryColor: primaryColor,
      backgroundIllustration: backgroundIllustration,
      avatarFrame: avatarFrame,
    ),
  );
}
