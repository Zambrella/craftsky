import 'package:craftsky_app/profile/widgets/profile_presentation_page.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';

/// Opens a profile route in its compact summary-card presentation.
///
/// The URL changes immediately. Expanding the summary is an internal state
/// change of that same route, so Back always returns directly to the origin.
Future<void> showUserProfileCard(
  BuildContext context, {
  required String handleOrDid,
}) {
  return UserProfileRoute(
    handle: handleOrDid,
    $extra: const ProfilePresentationRequest.compact(),
  ).push<void>(context);
}
