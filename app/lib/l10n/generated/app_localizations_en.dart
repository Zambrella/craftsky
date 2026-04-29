// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'Craftsky';

  @override
  String get homeSubtitle => 'Scaffold ready';

  @override
  String homeVersionLabel(String version) {
    return 'v$version';
  }

  @override
  String get initializationFailedTitle => 'Initialization Failed';

  @override
  String get retryButton => 'Retry';

  @override
  String get routingErrorTitle => 'Something went wrong';

  @override
  String get goHomeButton => 'Go home';

  @override
  String get profileEditAction => 'Edit profile';

  @override
  String get profileSettingsAction => 'Settings';

  @override
  String get profileShareAction => 'Share';

  @override
  String get profileFollowAction => 'Follow';

  @override
  String get profileFollowingAction => 'Following';

  @override
  String get profileTabPosts => 'Posts';

  @override
  String get profileTabProjects => 'Projects';

  @override
  String get profileTabSaved => 'Saved';

  @override
  String get profileTabReposts => 'Reposts';

  @override
  String get profileTabAbout => 'About';

  @override
  String get profileStatsFollowing => 'following';

  @override
  String get profileStatsFollowers => 'followers';

  @override
  String get profileStatsProjects => 'projects';

  @override
  String get profileLoadErrorTitle => 'That didn\'t load.';

  @override
  String get profileLoadErrorRetry => 'Try again';

  @override
  String get profileAboutEmpty => 'Nothing here yet.';

  @override
  String get profileAboutCraftsHeading => 'Crafts';

  @override
  String get profileAboutJoinedHeading => 'Joined';

  @override
  String get profileEmptyProjects => 'No projects yet.';

  @override
  String get profileEmptySaved => 'Nothing saved yet.';

  @override
  String get profileEmptyReposts => 'No reposts yet.';

  @override
  String get profileEditComingSoon => 'Profile editing coming soon.';

  @override
  String get profileFollowComingSoon => 'Follow coming soon.';

  @override
  String get profileShareComingSoon => 'Share coming soon.';
}
