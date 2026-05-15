import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:intl/intl.dart' as intl;

import 'app_localizations_en.dart';

// ignore_for_file: type=lint

/// Callers can lookup localized strings with an instance of AppLocalizations
/// returned by `AppLocalizations.of(context)`.
///
/// Applications need to include `AppLocalizations.delegate()` in their app's
/// `localizationDelegates` list, and the locales they support in the app's
/// `supportedLocales` list. For example:
///
/// ```dart
/// import 'generated/app_localizations.dart';
///
/// return MaterialApp(
///   localizationsDelegates: AppLocalizations.localizationsDelegates,
///   supportedLocales: AppLocalizations.supportedLocales,
///   home: MyApplicationHome(),
/// );
/// ```
///
/// ## Update pubspec.yaml
///
/// Please make sure to update your pubspec.yaml to include the following
/// packages:
///
/// ```yaml
/// dependencies:
///   # Internationalization support.
///   flutter_localizations:
///     sdk: flutter
///   intl: any # Use the pinned version from flutter_localizations
///
///   # Rest of dependencies
/// ```
///
/// ## iOS Applications
///
/// iOS applications define key application metadata, including supported
/// locales, in an Info.plist file that is built into the application bundle.
/// To configure the locales supported by your app, you’ll need to edit this
/// file.
///
/// First, open your project’s ios/Runner.xcworkspace Xcode workspace file.
/// Then, in the Project Navigator, open the Info.plist file under the Runner
/// project’s Runner folder.
///
/// Next, select the Information Property List item, select Add Item from the
/// Editor menu, then select Localizations from the pop-up menu.
///
/// Select and expand the newly-created Localizations item then, for each
/// locale your application supports, add a new item and select the locale
/// you wish to add from the pop-up menu in the Value field. This list should
/// be consistent with the languages listed in the AppLocalizations.supportedLocales
/// property.
abstract class AppLocalizations {
  AppLocalizations(String locale)
    : localeName = intl.Intl.canonicalizedLocale(locale.toString());

  final String localeName;

  static AppLocalizations of(BuildContext context) {
    return Localizations.of<AppLocalizations>(context, AppLocalizations)!;
  }

  static const LocalizationsDelegate<AppLocalizations> delegate =
      _AppLocalizationsDelegate();

  /// A list of this localizations delegate along with the default localizations
  /// delegates.
  ///
  /// Returns a list of localizations delegates containing this delegate along with
  /// GlobalMaterialLocalizations.delegate, GlobalCupertinoLocalizations.delegate,
  /// and GlobalWidgetsLocalizations.delegate.
  ///
  /// Additional delegates can be added by appending to this list in
  /// MaterialApp. This list does not have to be used at all if a custom list
  /// of delegates is preferred or required.
  static const List<LocalizationsDelegate<dynamic>> localizationsDelegates =
      <LocalizationsDelegate<dynamic>>[
        delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ];

  /// A list of this localizations delegate's supported locales.
  static const List<Locale> supportedLocales = <Locale>[Locale('en')];

  /// The app's title, used in MaterialApp.title and as an AppBar title.
  ///
  /// In en, this message translates to:
  /// **'Craftsky'**
  String get appTitle;

  /// Muted subtitle on the placeholder HomePage.
  ///
  /// In en, this message translates to:
  /// **'Scaffold ready'**
  String get homeSubtitle;

  /// Renders the running app version below the subtitle on HomePage.
  ///
  /// In en, this message translates to:
  /// **'v{version}'**
  String homeVersionLabel(String version);

  /// Default label for the primary action button on a CraftskyDialog confirm helper when the caller does not provide one.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get dialogConfirmDefault;

  /// Default label for the secondary action button on a CraftskyDialog confirm helper when the caller does not provide one.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get dialogCancelDefault;

  /// Default label for the dismiss button on a CraftskyDialog alert helper when the caller does not provide one.
  ///
  /// In en, this message translates to:
  /// **'OK'**
  String get dialogOkDefault;

  /// Generic accessibility label announced by the StitchProgressIndicator while content is loading.
  ///
  /// In en, this message translates to:
  /// **'Loading'**
  String get loading;

  /// Headline on InitializationErrorScreen when appDependenciesProvider fails.
  ///
  /// In en, this message translates to:
  /// **'Initialization Failed'**
  String get initializationFailedTitle;

  /// Retry-action button label on InitializationErrorScreen.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get retryButton;

  /// Semantics label and tooltip on the close icon shown on sticky warning/error messages dispatched via AppMessenger.
  ///
  /// In en, this message translates to:
  /// **'Dismiss'**
  String get messengerDismiss;

  /// Headline on ErrorScreen (from GoRouter.errorBuilder).
  ///
  /// In en, this message translates to:
  /// **'Something went wrong'**
  String get routingErrorTitle;

  /// Button label on routing ErrorScreen returning to HomeRoute.
  ///
  /// In en, this message translates to:
  /// **'Go home'**
  String get goHomeButton;

  /// Label on the primary action button shown on a self-profile, opens the edit-profile flow.
  ///
  /// In en, this message translates to:
  /// **'Edit profile'**
  String get profileEditAction;

  /// Tooltip on the settings icon button shown next to Edit profile on a self-profile, and on the collapsed-bar trailing action.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get profileSettingsAction;

  /// Tooltip on the share icon button shown on a visitor profile, and on the collapsed-bar trailing action.
  ///
  /// In en, this message translates to:
  /// **'Share'**
  String get profileShareAction;

  /// Label on the follow button on a visitor profile when the viewer is not yet following them.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get profileFollowAction;

  /// Label on the follow button on a visitor profile when the viewer is already following them.
  ///
  /// In en, this message translates to:
  /// **'Following'**
  String get profileFollowingAction;

  /// Tab label for the Posts tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'Posts'**
  String get profileTabPosts;

  /// Tab label for the Comments tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'Comments'**
  String get profileTabComments;

  /// Tab label for the Projects tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'Projects'**
  String get profileTabProjects;

  /// Tab label for the Saved tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get profileTabSaved;

  /// Tab label for the Reposts tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'Reposts'**
  String get profileTabReposts;

  /// Tab label for the About tab on the profile screen.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get profileTabAbout;

  /// Lower-case label paired with the following-count on the profile stats row.
  ///
  /// In en, this message translates to:
  /// **'following'**
  String get profileStatsFollowing;

  /// Lower-case label paired with the follower-count on the profile stats row.
  ///
  /// In en, this message translates to:
  /// **'followers'**
  String get profileStatsFollowers;

  /// Lower-case label paired with the project-count on the profile stats row.
  ///
  /// In en, this message translates to:
  /// **'projects'**
  String get profileStatsProjects;

  /// Headline on the full-screen profile-page error fallback when the profile fetch fails.
  ///
  /// In en, this message translates to:
  /// **'That didn\'t load.'**
  String get profileLoadErrorTitle;

  /// Retry-action button label on the full-screen profile-page error fallback.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get profileLoadErrorRetry;

  /// Muted placeholder shown in the About tab when the profile has no bio.
  ///
  /// In en, this message translates to:
  /// **'Nothing here yet.'**
  String get profileAboutEmpty;

  /// Section heading above the craft chips on the About tab.
  ///
  /// In en, this message translates to:
  /// **'Crafts'**
  String get profileAboutCraftsHeading;

  /// Section heading above the join-date on the About tab.
  ///
  /// In en, this message translates to:
  /// **'Joined'**
  String get profileAboutJoinedHeading;

  /// Muted placeholder shown in the Projects tab while project data isn't wired.
  ///
  /// In en, this message translates to:
  /// **'No projects yet.'**
  String get profileEmptyProjects;

  /// Muted placeholder shown in the Saved tab while saved-item data isn't wired.
  ///
  /// In en, this message translates to:
  /// **'Nothing saved yet.'**
  String get profileEmptySaved;

  /// Muted placeholder shown in the Reposts tab while repost data isn't wired.
  ///
  /// In en, this message translates to:
  /// **'No reposts yet.'**
  String get profileEmptyReposts;

  /// Muted placeholder shown in the profile Posts tab when the user has not posted.
  ///
  /// In en, this message translates to:
  /// **'No posts yet.'**
  String get profilePostsEmpty;

  /// Error title shown in the profile Posts tab when post fetching fails.
  ///
  /// In en, this message translates to:
  /// **'Posts didn\'t load.'**
  String get profilePostsLoadError;

  /// Button label for loading the next page in the profile Posts tab.
  ///
  /// In en, this message translates to:
  /// **'Load more posts'**
  String get profilePostsLoadMore;

  /// Muted placeholder shown in the profile Comments tab when the user has not commented.
  ///
  /// In en, this message translates to:
  /// **'No comments yet.'**
  String get profileCommentsEmpty;

  /// Error title shown in the profile Comments tab when comment fetching fails.
  ///
  /// In en, this message translates to:
  /// **'Comments didn\'t load.'**
  String get profileCommentsLoadError;

  /// Button label for loading the next page in the profile Comments tab.
  ///
  /// In en, this message translates to:
  /// **'Load more comments'**
  String get profileCommentsLoadMore;

  /// Title of the post thread screen.
  ///
  /// In en, this message translates to:
  /// **'Thread'**
  String get postThreadTitle;

  /// Empty state shown on a post thread when the post has no direct replies.
  ///
  /// In en, this message translates to:
  /// **'No replies yet.'**
  String get postThreadEmptyReplies;

  /// Label shown when a thread response has additional replies that are not loaded yet.
  ///
  /// In en, this message translates to:
  /// **'Read more replies'**
  String get postThreadReadMoreReplies;

  /// Button label shown under a reply that has multiple hidden child replies.
  ///
  /// In en, this message translates to:
  /// **'Show more replies'**
  String get postThreadShowMoreReplies;

  /// Button label shown under a reply that continues into one hidden child reply.
  ///
  /// In en, this message translates to:
  /// **'Continue thread'**
  String get postThreadContinueThread;

  /// Button label for replying from the post thread screen.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get postThreadReplyAction;

  /// Button label for commenting on a root post from the post thread screen.
  ///
  /// In en, this message translates to:
  /// **'Comment'**
  String get postCommentAction;

  /// Accessibility label for a reply button on the thread screen. The author placeholder includes a display name or handle.
  ///
  /// In en, this message translates to:
  /// **'Reply to {author}'**
  String postThreadReplyToAuthor(String author);

  /// Accessibility label for a comment button on the thread screen. The author placeholder includes a display name or handle.
  ///
  /// In en, this message translates to:
  /// **'Comment on {author}'**
  String postCommentOnAuthor(String author);

  /// Accessibility label for the show-more-replies continuation button on the thread screen. The author placeholder identifies the post being continued.
  ///
  /// In en, this message translates to:
  /// **'Show more replies to {author}'**
  String postThreadShowMoreRepliesForAuthor(String author);

  /// Accessibility label for the continue-thread button on the thread screen. The author placeholder identifies the post being continued.
  ///
  /// In en, this message translates to:
  /// **'Continue thread from {author}'**
  String postThreadContinueThreadFromAuthor(String author);

  /// Comment-section sort option for oldest-first comment ordering.
  ///
  /// In en, this message translates to:
  /// **'Oldest'**
  String get postCommentsSortOldest;

  /// Helper text for the oldest-first comment sort option.
  ///
  /// In en, this message translates to:
  /// **'Conversation order'**
  String get postCommentsSortOldestDescription;

  /// Comment-section sort option for newest-first comment ordering.
  ///
  /// In en, this message translates to:
  /// **'Newest'**
  String get postCommentsSortNewest;

  /// Helper text for the newest-first comment sort option.
  ///
  /// In en, this message translates to:
  /// **'Most recent on top'**
  String get postCommentsSortNewestDescription;

  /// Comment-section sort option for follows-based ordering. Until follow ranking exists, this behaves like oldest-first.
  ///
  /// In en, this message translates to:
  /// **'Follows'**
  String get postCommentsSortFollows;

  /// Helper text for the follows-first comment sort option.
  ///
  /// In en, this message translates to:
  /// **'People you follow first'**
  String get postCommentsSortFollowsDescription;

  /// Control label shown under a comment before its replies are loaded.
  ///
  /// In en, this message translates to:
  /// **'View replies'**
  String get postCommentsViewReplies;

  /// Control label shown under a comment before its replies are loaded, including the total reply count.
  ///
  /// In en, this message translates to:
  /// **'Show {count, plural, =1{1 reply} other{{count} replies}}'**
  String postCommentsViewReplyCount(int count);

  /// Control label for loading another page of replies under an expanded comment.
  ///
  /// In en, this message translates to:
  /// **'Load more replies'**
  String get postCommentsLoadMoreReplies;

  /// Control label for collapsing an expanded comment reply list.
  ///
  /// In en, this message translates to:
  /// **'Hide replies'**
  String get postCommentsHideReplies;

  /// Message shown when a focused comment/reply link points to an item the AppView has not indexed or can no longer find.
  ///
  /// In en, this message translates to:
  /// **'That reply isn\'t available yet.'**
  String get postCommentsFocusNotFound;

  /// Message shown when a focused comment/reply link does not belong under the route's root post.
  ///
  /// In en, this message translates to:
  /// **'That reply belongs to a different post.'**
  String get postCommentsFocusMismatchedRoot;

  /// Button label that opens the text-only post composer.
  ///
  /// In en, this message translates to:
  /// **'New post'**
  String get postComposeAction;

  /// Title of the text-only post composer sheet.
  ///
  /// In en, this message translates to:
  /// **'New post'**
  String get postComposeTitle;

  /// Label above the text field in the text-only post composer.
  ///
  /// In en, this message translates to:
  /// **'What are you making?'**
  String get postComposeHint;

  /// Title of the reply composer sheet.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get postComposeReplyTitle;

  /// Label above the text field in reply mode.
  ///
  /// In en, this message translates to:
  /// **'Write your reply'**
  String get postComposeReplyHint;

  /// Submit button label in the text-only post composer.
  ///
  /// In en, this message translates to:
  /// **'Post'**
  String get postComposeSubmit;

  /// Submit button label in reply mode.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get postComposeReplySubmit;

  /// Validation error shown when the text-only post composer exceeds the post text limit.
  ///
  /// In en, this message translates to:
  /// **'Posts must be 2000 characters or fewer'**
  String get postComposeTooLong;

  /// Snackbar shown after successfully creating a post.
  ///
  /// In en, this message translates to:
  /// **'Posted.'**
  String get postCreateSuccess;

  /// Snackbar shown when creating a post fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t post.'**
  String get postCreateError;

  /// Tooltip and menu label for deleting a post.
  ///
  /// In en, this message translates to:
  /// **'Delete post'**
  String get postDeleteAction;

  /// Tooltip for opening a post, comment, or reply context menu when no destructive action label applies.
  ///
  /// In en, this message translates to:
  /// **'More actions'**
  String get postMoreActions;

  /// Menu label for deleting a comment.
  ///
  /// In en, this message translates to:
  /// **'Delete comment'**
  String get commentDeleteAction;

  /// Menu label for deleting a reply.
  ///
  /// In en, this message translates to:
  /// **'Delete reply'**
  String get replyDeleteAction;

  /// Title of the delete-post confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'Delete post?'**
  String get postDeleteTitle;

  /// Title of the delete-comment confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'Delete comment?'**
  String get commentDeleteTitle;

  /// Title of the delete-reply confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'Delete reply?'**
  String get replyDeleteTitle;

  /// Body text of the delete-post confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This removes the post from Craftsky.'**
  String get postDeleteMessage;

  /// Body text of the delete-comment confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This removes the comment from Craftsky.'**
  String get commentDeleteMessage;

  /// Body text of the delete-reply confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This removes the reply from Craftsky.'**
  String get replyDeleteMessage;

  /// Confirm button label in the delete-post confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get postDeleteConfirm;

  /// Snackbar shown after successfully deleting a post.
  ///
  /// In en, this message translates to:
  /// **'Post deleted.'**
  String get postDeleteSuccess;

  /// Snackbar shown when deleting a post fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t delete post.'**
  String get postDeleteError;

  /// Snackbar shown when tapping Follow while follow wiring isn't implemented yet.
  ///
  /// In en, this message translates to:
  /// **'Follow coming soon.'**
  String get profileFollowComingSoon;

  /// Snackbar shown when tapping Share while share wiring isn't implemented yet.
  ///
  /// In en, this message translates to:
  /// **'Share coming soon.'**
  String get profileShareComingSoon;

  /// App-bar title on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Edit profile'**
  String get editProfileTitle;

  /// Label on the save action in the profile-edit app bar.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get editProfileSave;

  /// Tooltip on the close (back) action in the profile-edit app bar.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get editProfileCancel;

  /// Label above the display-name field on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Display name'**
  String get editProfileDisplayNameLabel;

  /// Hint text inside the display-name field on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'How your name appears on your profile'**
  String get editProfileDisplayNameHint;

  /// Label above the bio field on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Bio'**
  String get editProfileBioLabel;

  /// Hint text inside the bio field on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Tell people about yourself'**
  String get editProfileBioHint;

  /// Form-validation error shown below the display-name field on the profile-edit page when the user enters more than 64 characters (the AppView profile lexicon's grapheme limit).
  ///
  /// In en, this message translates to:
  /// **'Display name must be 64 characters or fewer'**
  String get editProfileDisplayNameTooLong;

  /// Form-validation error shown below the bio field on the profile-edit page when the user enters more than 256 characters (the AppView profile lexicon's grapheme limit).
  ///
  /// In en, this message translates to:
  /// **'Bio must be 256 characters or fewer'**
  String get editProfileBioTooLong;

  /// Section heading above the crafts picker on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Crafts'**
  String get editProfileCraftsLabel;

  /// Helper text below the crafts heading on the profile-edit page, hinting that the user should pick from the list.
  ///
  /// In en, this message translates to:
  /// **'Pick the crafts you make'**
  String get editProfileCraftsHelper;

  /// Caption shown over the avatar/banner area on the profile-edit page, indicating that photo upload isn't wired yet.
  ///
  /// In en, this message translates to:
  /// **'Photo uploads coming soon'**
  String get editProfilePhotosComingSoon;

  /// Snackbar shown when the profile-edit save request fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save your profile.'**
  String get editProfileSaveError;

  /// Title of the confirm-discard dialog shown when leaving the profile-edit page with unsaved changes.
  ///
  /// In en, this message translates to:
  /// **'Discard changes?'**
  String get editProfileDiscardTitle;

  /// Body of the confirm-discard dialog on the profile-edit page.
  ///
  /// In en, this message translates to:
  /// **'Your edits won\'t be saved.'**
  String get editProfileDiscardMessage;

  /// Confirm action in the confirm-discard dialog — closes the edit page without saving.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get editProfileDiscardConfirm;

  /// Cancel action in the confirm-discard dialog — returns the user to the edit form.
  ///
  /// In en, this message translates to:
  /// **'Keep editing'**
  String get editProfileDiscardCancel;

  /// Display label for the 'sewing' craft option in the crafts picker.
  ///
  /// In en, this message translates to:
  /// **'Sewing'**
  String get craftSewing;

  /// Display label for the 'quilting' craft option.
  ///
  /// In en, this message translates to:
  /// **'Quilting'**
  String get craftQuilting;

  /// Display label for the 'knitting' craft option.
  ///
  /// In en, this message translates to:
  /// **'Knitting'**
  String get craftKnitting;

  /// Display label for the 'crochet' craft option.
  ///
  /// In en, this message translates to:
  /// **'Crochet'**
  String get craftCrochet;

  /// Display label for the 'embroidery' craft option.
  ///
  /// In en, this message translates to:
  /// **'Embroidery'**
  String get craftEmbroidery;

  /// Display label for the 'cross-stitch' craft option.
  ///
  /// In en, this message translates to:
  /// **'Cross-stitch'**
  String get craftCrossStitch;

  /// Display label for the 'weaving' craft option.
  ///
  /// In en, this message translates to:
  /// **'Weaving'**
  String get craftWeaving;

  /// Display label for the 'spinning' craft option.
  ///
  /// In en, this message translates to:
  /// **'Spinning'**
  String get craftSpinning;

  /// Display label for the 'felting' craft option.
  ///
  /// In en, this message translates to:
  /// **'Felting'**
  String get craftFelting;

  /// Display label for the 'macrame' craft option.
  ///
  /// In en, this message translates to:
  /// **'Macramé'**
  String get craftMacrame;

  /// Display label for the 'pottery' craft option.
  ///
  /// In en, this message translates to:
  /// **'Pottery'**
  String get craftPottery;

  /// Display label for the 'woodworking' craft option.
  ///
  /// In en, this message translates to:
  /// **'Woodworking'**
  String get craftWoodworking;

  /// Display label for the 'leatherwork' craft option.
  ///
  /// In en, this message translates to:
  /// **'Leatherwork'**
  String get craftLeatherwork;

  /// Display label for the 'jewellery' craft option.
  ///
  /// In en, this message translates to:
  /// **'Jewellery'**
  String get craftJewellery;

  /// Display label for the 'bookbinding' craft option.
  ///
  /// In en, this message translates to:
  /// **'Bookbinding'**
  String get craftBookbinding;

  /// Display label for the 'calligraphy' craft option.
  ///
  /// In en, this message translates to:
  /// **'Calligraphy'**
  String get craftCalligraphy;

  /// Display label for the 'printmaking' craft option.
  ///
  /// In en, this message translates to:
  /// **'Printmaking'**
  String get craftPrintmaking;

  /// Display label for the 'papercraft' craft option (covers origami, kirigami, card-making).
  ///
  /// In en, this message translates to:
  /// **'Paper craft'**
  String get craftPapercraft;

  /// Display label for the 'painting' craft option.
  ///
  /// In en, this message translates to:
  /// **'Painting'**
  String get craftPainting;

  /// Display label for the 'drawing' craft option.
  ///
  /// In en, this message translates to:
  /// **'Drawing'**
  String get craftDrawing;

  /// Display label for the 'candlemaking' craft option.
  ///
  /// In en, this message translates to:
  /// **'Candle making'**
  String get craftCandleMaking;

  /// Display label for the 'soapmaking' craft option.
  ///
  /// In en, this message translates to:
  /// **'Soap making'**
  String get craftSoapMaking;
}

class _AppLocalizationsDelegate
    extends LocalizationsDelegate<AppLocalizations> {
  const _AppLocalizationsDelegate();

  @override
  Future<AppLocalizations> load(Locale locale) {
    return SynchronousFuture<AppLocalizations>(lookupAppLocalizations(locale));
  }

  @override
  bool isSupported(Locale locale) =>
      <String>['en'].contains(locale.languageCode);

  @override
  bool shouldReload(_AppLocalizationsDelegate old) => false;
}

AppLocalizations lookupAppLocalizations(Locale locale) {
  // Lookup logic when only language code is specified.
  switch (locale.languageCode) {
    case 'en':
      return AppLocalizationsEn();
  }

  throw FlutterError(
    'AppLocalizations.delegate failed to load unsupported locale "$locale". This is likely '
    'an issue with the localizations generation tool. Please file an issue '
    'on GitHub with a reproducible sample app and the gen-l10n configuration '
    'that was used.',
  );
}
