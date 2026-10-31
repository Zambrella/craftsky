import 'package:craftsky_app/router/router.dart';
import 'package:go_router/go_router.dart';

String statusDestinationLocation(String destination) => switch (destination) {
  'feed' => const FeedRoute().location,
  'projects' => const FeedRoute().location,
  'search' => const SearchRoute().location,
  'notifications' => const NotificationsRoute().location,
  'profile' => const ProfileRoute().location,
  'settings' => const SettingsRoute().location,
  _ => throw const FormatException('Status destination'),
};

/// The status host lives in MaterialApp.router.builder, above the Router
/// inherited widget, so navigate directly using the typed route's location.
bool activateStatusDestination(GoRouter router, String destination) {
  router.go(statusDestinationLocation(destination));
  return true;
}
