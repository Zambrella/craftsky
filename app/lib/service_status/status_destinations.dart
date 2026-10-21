import 'package:craftsky_app/router/router.dart';
import 'package:go_router/go_router.dart';

String statusDestinationLocation(String destination) => switch (destination) {
  'feed' => const FeedRoute().location,
  'projects' => const ProjectsRoute().location,
  'search' => const SearchRoute().location,
  'notifications' => const NotificationsRoute().location,
  'profile' => const ProfileRoute().location,
  'settings' => const SettingsRoute().location,
  _ => throw const FormatException('Status destination'),
};

bool activateStatusDestination(GoRouter router, String destination) {
  router.go(statusDestinationLocation(destination));
  return true;
}
