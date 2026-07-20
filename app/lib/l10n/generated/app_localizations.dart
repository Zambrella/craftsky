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
  /// **'CraftSky'**
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

  /// Title for the main chronological feed page.
  ///
  /// In en, this message translates to:
  /// **'Feed'**
  String get feedTitle;

  /// Title for the in-app notifications page.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get notificationsTitle;

  /// Empty state shown on the notifications tab when there is no social activity.
  ///
  /// In en, this message translates to:
  /// **'No notifications yet.'**
  String get notificationsEmpty;

  /// Error title shown on the notifications tab when notification fetching fails.
  ///
  /// In en, this message translates to:
  /// **'Notifications didn\'t load.'**
  String get notificationsLoadError;

  /// Button label for loading the next page of notifications.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get notificationsLoadMore;

  /// Notification row title for a follow activity.
  ///
  /// In en, this message translates to:
  /// **'{actor} followed you'**
  String notificationFollowRow(String actor);

  /// Notification row title for a like activity on the viewer's post.
  ///
  /// In en, this message translates to:
  /// **'{actor} liked your post'**
  String notificationLikeRow(String actor);

  /// Notification row title for a like activity on the viewer's direct comment.
  ///
  /// In en, this message translates to:
  /// **'{actor} liked your comment'**
  String notificationLikeCommentRow(String actor);

  /// Notification row title for a like activity on the viewer's nested reply.
  ///
  /// In en, this message translates to:
  /// **'{actor} liked your reply'**
  String notificationLikeReplyRow(String actor);

  /// Notification row title for a repost activity on the viewer's post.
  ///
  /// In en, this message translates to:
  /// **'{actor} reposted your post'**
  String notificationRepostRow(String actor);

  /// Notification row title for a repost activity on the viewer's direct comment.
  ///
  /// In en, this message translates to:
  /// **'{actor} reposted your comment'**
  String notificationRepostCommentRow(String actor);

  /// Notification row title for a repost activity on the viewer's nested reply.
  ///
  /// In en, this message translates to:
  /// **'{actor} reposted your reply'**
  String notificationRepostReplyRow(String actor);

  /// Notification row title for a direct comment activity on the viewer's root post.
  ///
  /// In en, this message translates to:
  /// **'{actor} commented on your post'**
  String notificationReplyRow(String actor);

  /// Notification row title for a response to the viewer's direct comment.
  ///
  /// In en, this message translates to:
  /// **'{actor} replied to your comment'**
  String notificationReplyToCommentRow(String actor);

  /// Notification row title for a response to the viewer's nested reply.
  ///
  /// In en, this message translates to:
  /// **'{actor} replied to your reply'**
  String notificationReplyToReplyRow(String actor);

  /// Notification row title for a mention activity.
  ///
  /// In en, this message translates to:
  /// **'{actor} mentioned you'**
  String notificationMentionRow(String actor);

  /// Notification row title for a quote.
  ///
  /// In en, this message translates to:
  /// **'{actor} quoted your post'**
  String notificationQuoteRow(String actor);

  /// No description provided for @notificationGenericRow.
  ///
  /// In en, this message translates to:
  /// **'New activity'**
  String get notificationGenericRow;

  /// No description provided for @notificationUnavailableRow.
  ///
  /// In en, this message translates to:
  /// **'Activity unavailable'**
  String get notificationUnavailableRow;

  /// No description provided for @notificationSettingsAction.
  ///
  /// In en, this message translates to:
  /// **'Notification settings'**
  String get notificationSettingsAction;

  /// No description provided for @notificationSettingsIntro.
  ///
  /// In en, this message translates to:
  /// **'Category preferences apply to all devices signed in to this account.'**
  String get notificationSettingsIntro;

  /// No description provided for @notificationDeviceDisabled.
  ///
  /// In en, this message translates to:
  /// **'Notifications are disabled on this device'**
  String get notificationDeviceDisabled;

  /// No description provided for @notificationDeviceDisabledDescription.
  ///
  /// In en, this message translates to:
  /// **'Account preferences still apply. Enable alerts in system settings.'**
  String get notificationDeviceDisabledDescription;

  /// No description provided for @notificationOpenSettings.
  ///
  /// In en, this message translates to:
  /// **'Open settings'**
  String get notificationOpenSettings;

  /// No description provided for @notificationCategoryLikes.
  ///
  /// In en, this message translates to:
  /// **'Likes'**
  String get notificationCategoryLikes;

  /// No description provided for @notificationCategoryFollows.
  ///
  /// In en, this message translates to:
  /// **'Follows'**
  String get notificationCategoryFollows;

  /// No description provided for @notificationCategoryReplies.
  ///
  /// In en, this message translates to:
  /// **'Replies'**
  String get notificationCategoryReplies;

  /// No description provided for @notificationCategoryMentions.
  ///
  /// In en, this message translates to:
  /// **'Mentions'**
  String get notificationCategoryMentions;

  /// No description provided for @notificationCategoryQuotes.
  ///
  /// In en, this message translates to:
  /// **'Quotes'**
  String get notificationCategoryQuotes;

  /// No description provided for @notificationCategoryReposts.
  ///
  /// In en, this message translates to:
  /// **'Reposts'**
  String get notificationCategoryReposts;

  /// No description provided for @notificationCategoryEverythingElse.
  ///
  /// In en, this message translates to:
  /// **'Everything else'**
  String get notificationCategoryEverythingElse;

  /// No description provided for @notificationPreferenceFrom.
  ///
  /// In en, this message translates to:
  /// **'From'**
  String get notificationPreferenceFrom;

  /// No description provided for @notificationScopeEveryone.
  ///
  /// In en, this message translates to:
  /// **'Everyone'**
  String get notificationScopeEveryone;

  /// No description provided for @notificationScopePeopleIFollow.
  ///
  /// In en, this message translates to:
  /// **'People I follow'**
  String get notificationScopePeopleIFollow;

  /// No description provided for @notificationPushEnabled.
  ///
  /// In en, this message translates to:
  /// **'Push notifications'**
  String get notificationPushEnabled;

  /// No description provided for @notificationPreferenceSaveError.
  ///
  /// In en, this message translates to:
  /// **'Could not save notification preference'**
  String get notificationPreferenceSaveError;

  /// No description provided for @notificationBannerOpen.
  ///
  /// In en, this message translates to:
  /// **'Open'**
  String get notificationBannerOpen;

  /// Accessible in-app notification badge label.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 new activity} other{{count} new activities}}'**
  String notificationNewActivityCount(int count);

  /// Title and main heading on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'Welcome'**
  String get welcomeTitle;

  /// Primary button label on the welcome page that opens sign-in.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get welcomeSignInAction;

  /// Secondary action on the welcome page for users who need a PDS account.
  ///
  /// In en, this message translates to:
  /// **'Create account on a PDS'**
  String get welcomeCreateAccountAction;

  /// App-bar title on the sign-in page.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get signInTitle;

  /// App-bar title when adding another retained account.
  ///
  /// In en, this message translates to:
  /// **'Add account'**
  String get addAccountTitle;

  /// Explains that adding an account preserves the current account.
  ///
  /// In en, this message translates to:
  /// **'Sign in to another account. Your current account stays signed in.'**
  String get addAccountDescription;

  /// Action for starting another account sign-in.
  ///
  /// In en, this message translates to:
  /// **'Add account'**
  String get accountSwitcherAdd;

  /// Helper shown when the retained-account limit is reached.
  ///
  /// In en, this message translates to:
  /// **'Maximum of 5 accounts'**
  String get accountSwitcherMaximum;

  /// Accessible label for opening the account switcher.
  ///
  /// In en, this message translates to:
  /// **'Switch account'**
  String get accountSwitcherTooltip;

  /// Compact navigation hint for opening the account switcher.
  ///
  /// In en, this message translates to:
  /// **'Long press to switch account'**
  String get accountSwitcherLongPressHint;

  /// Accessible progress label during an account transition.
  ///
  /// In en, this message translates to:
  /// **'Switching account'**
  String get accountSwitchingLabel;

  /// Identity fallback when cached account metadata is unavailable.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get accountIdentityFallback;

  /// Label for the handle input on the sign-in page.
  ///
  /// In en, this message translates to:
  /// **'Handle'**
  String get signInHandleLabel;

  /// Primary button label on the sign-in page.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get signInContinueAction;

  /// Snackbar error when submitting sign-in without a handle.
  ///
  /// In en, this message translates to:
  /// **'Please enter a handle.'**
  String get signInHandleRequiredError;

  /// Snackbar error when the sign-in handle is malformed or cannot be resolved.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t recognise that handle.'**
  String get signInInvalidHandleError;

  /// Snackbar error when the auth server cannot be reached.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t reach the server. Please try again.'**
  String get signInServerUnavailableError;

  /// Snackbar error when OAuth sign-in cannot open the system browser.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t open the browser. Check that you have one installed.'**
  String get signInBrowserLaunchError;

  /// Fallback snackbar error for sign-in failures.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong. Please try again.'**
  String get signInGenericError;

  /// Loading message shown while completing OAuth sign-in from a deep link.
  ///
  /// In en, this message translates to:
  /// **'Signing in…'**
  String get authCompleteSigningIn;

  /// Error shown when the OAuth completion token has expired.
  ///
  /// In en, this message translates to:
  /// **'That sign-in link expired. Please sign in again.'**
  String get authCompleteTimedOutError;

  /// Error shown when the OAuth callback has no matching pending sign-in.
  ///
  /// In en, this message translates to:
  /// **'No sign-in is in progress. Please sign in again.'**
  String get authCompleteNoPendingSignInError;

  /// Error shown when the completed OAuth session cannot be saved.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save your session securely. Please sign in again.'**
  String get authCompleteStorageError;

  /// Fallback error shown when OAuth completion fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t complete sign-in. Please sign in again.'**
  String get authCompleteGenericError;

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

  /// Generic button label for returning to the previous screen.
  ///
  /// In en, this message translates to:
  /// **'Back'**
  String get backButton;

  /// Title shown when a post or profile opened from a notification is permanently unavailable.
  ///
  /// In en, this message translates to:
  /// **'This is no longer available'**
  String get notificationDestinationUnavailableTitle;

  /// Safe explanation shown when a notification destination is permanently unavailable.
  ///
  /// In en, this message translates to:
  /// **'This post or profile may have been deleted or hidden.'**
  String get notificationDestinationUnavailableBody;

  /// Action returning from an unavailable notification destination to the notifications list.
  ///
  /// In en, this message translates to:
  /// **'View notifications'**
  String get notificationDestinationViewNotifications;

  /// Title shown when a notification destination fails for a retryable reason.
  ///
  /// In en, this message translates to:
  /// **'That didn\'t load'**
  String get notificationDestinationRetryTitle;

  /// Safe explanation shown when loading a notification destination can be retried.
  ///
  /// In en, this message translates to:
  /// **'Check your connection and try again.'**
  String get notificationDestinationRetryBody;

  /// Empty state shown on the main chronological Feed tab when the home timeline has no posts.
  ///
  /// In en, this message translates to:
  /// **'Your feed is quiet.'**
  String get feedEmpty;

  /// Error title shown on the main chronological Feed tab when timeline fetching fails.
  ///
  /// In en, this message translates to:
  /// **'Feed didn\'t load.'**
  String get feedLoadError;

  /// Semantics label and tooltip on the close icon shown on sticky warning/error messages dispatched via AppMessenger.
  ///
  /// In en, this message translates to:
  /// **'Dismiss'**
  String get messengerDismiss;

  /// Transient confirmation shown after the member signs out their final retained account.
  ///
  /// In en, this message translates to:
  /// **'Signed out successfully.'**
  String get signOutSuccess;

  /// Transient confirmation shown after sign-out activates another retained account.
  ///
  /// In en, this message translates to:
  /// **'Signed out successfully. Now signed in as @{handle}.'**
  String signOutSuccessWithAccount(String handle);

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

  /// Safe generic error message shown when the app cannot reach the network.
  ///
  /// In en, this message translates to:
  /// **'You\'re offline. Check your connection and try again.'**
  String get errorNetworkUnavailable;

  /// Safe generic error message shown when the CraftSky service is unavailable.
  ///
  /// In en, this message translates to:
  /// **'CraftSky is having trouble right now. Please try again.'**
  String get errorServiceUnavailable;

  /// Safe generic error message shown when the user's session is no longer valid.
  ///
  /// In en, this message translates to:
  /// **'Please sign in again.'**
  String get errorSessionExpired;

  /// Safe generic error message shown when the user cannot access an action or resource.
  ///
  /// In en, this message translates to:
  /// **'You don\'t have permission to do that.'**
  String get errorPermissionDenied;

  /// Safe generic error message shown when a post, project, profile, or other content cannot be found.
  ///
  /// In en, this message translates to:
  /// **'That content is no longer available.'**
  String get errorContentUnavailable;

  /// Safe generic error message shown when local secure storage cannot be read or written.
  ///
  /// In en, this message translates to:
  /// **'CraftSky couldn\'t access secure storage. Please try again.'**
  String get errorStorageUnavailable;

  /// Safe generic error message shown on the initialization error screen.
  ///
  /// In en, this message translates to:
  /// **'CraftSky couldn\'t finish starting. Please try again.'**
  String get errorInitializationFailed;

  /// Safe generic error message shown on the routing error screen.
  ///
  /// In en, this message translates to:
  /// **'That page couldn\'t be opened.'**
  String get errorNavigationFailed;

  /// Safe generic error message shown when a user action fails.
  ///
  /// In en, this message translates to:
  /// **'That didn\'t work. Please try again.'**
  String get errorActionFailed;

  /// Safe generic error message shown for inline background-load failures.
  ///
  /// In en, this message translates to:
  /// **'This didn\'t load. Please try again.'**
  String get errorBackgroundLoadFailed;

  /// Safe generic fallback error message for unexpected failures.
  ///
  /// In en, this message translates to:
  /// **'Something went wrong. Please try again.'**
  String get errorUnexpected;

  /// Action label shown when an error requires the user to sign in again.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get errorActionSignIn;

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

  /// Accessible tooltip for the visitor profile More menu.
  ///
  /// In en, this message translates to:
  /// **'More profile actions'**
  String get profileMoreActions;

  /// Action that privately mutes a profile.
  ///
  /// In en, this message translates to:
  /// **'Mute account'**
  String get profileMuteAction;

  /// Action that removes a private profile mute.
  ///
  /// In en, this message translates to:
  /// **'Unmute account'**
  String get profileUnmuteAction;

  /// Destructive action that publicly blocks a profile.
  ///
  /// In en, this message translates to:
  /// **'Block account'**
  String get profileBlockAction;

  /// Action that removes the viewer's public profile block.
  ///
  /// In en, this message translates to:
  /// **'Unblock account'**
  String get profileUnblockAction;

  /// Viewer-only annotation on a muted profile.
  ///
  /// In en, this message translates to:
  /// **'Muted account'**
  String get profileMuteAnnotation;

  /// Annotation on a profile the viewer has blocked.
  ///
  /// In en, this message translates to:
  /// **'Blocked by you'**
  String get profileBlockingAnnotation;

  /// Annotation on a profile whose owner blocked the viewer.
  ///
  /// In en, this message translates to:
  /// **'This account has blocked you'**
  String get profileBlockedByAnnotation;

  /// Annotation when both accounts own a block.
  ///
  /// In en, this message translates to:
  /// **'You have blocked each other'**
  String get profileMutualBlockAnnotation;

  /// Feedback after a mute or block mutation rolls back.
  ///
  /// In en, this message translates to:
  /// **'Could not update account relationship.'**
  String get profileRelationshipError;

  /// Feedback after muting a profile.
  ///
  /// In en, this message translates to:
  /// **'Account muted.'**
  String get profileMuteSuccess;

  /// Feedback after unmuting a profile.
  ///
  /// In en, this message translates to:
  /// **'Account unmuted.'**
  String get profileUnmuteSuccess;

  /// Feedback after blocking a profile.
  ///
  /// In en, this message translates to:
  /// **'Account blocked.'**
  String get profileBlockSuccess;

  /// Feedback after unblocking a profile.
  ///
  /// In en, this message translates to:
  /// **'Account unblocked.'**
  String get profileUnblockSuccess;

  /// Title of the public block confirmation.
  ///
  /// In en, this message translates to:
  /// **'Block this account?'**
  String get profileBlockConfirmTitle;

  /// Consequences and public visibility warning in the block confirmation.
  ///
  /// In en, this message translates to:
  /// **'Blocking is public on the AT Protocol. You will no longer see or interact with each other\'s content.'**
  String get profileBlockConfirmBody;

  /// Title of the unblock confirmation.
  ///
  /// In en, this message translates to:
  /// **'Unblock this account?'**
  String get profileUnblockConfirmTitle;

  /// Consequences shown in the unblock confirmation.
  ///
  /// In en, this message translates to:
  /// **'You may see and interact with each other\'s content again.'**
  String get profileUnblockConfirmBody;

  /// Generic cancel action.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get actionCancel;

  /// Generic confirmation action.
  ///
  /// In en, this message translates to:
  /// **'Confirm'**
  String get actionConfirm;

  /// Accessibility hint for actions that can remove content or a public relationship.
  ///
  /// In en, this message translates to:
  /// **'Destructive action'**
  String get destructiveActionHint;

  /// Settings entry and page title for private mutes.
  ///
  /// In en, this message translates to:
  /// **'Muted accounts'**
  String get settingsMutedAccounts;

  /// Settings entry and page title for public blocks.
  ///
  /// In en, this message translates to:
  /// **'Blocked accounts'**
  String get settingsBlockedAccounts;

  /// Empty state for the muted-account list.
  ///
  /// In en, this message translates to:
  /// **'You have not muted any accounts.'**
  String get settingsMutedAccountsEmpty;

  /// Empty state for the blocked-account list.
  ///
  /// In en, this message translates to:
  /// **'You have not blocked any accounts.'**
  String get settingsBlockedAccountsEmpty;

  /// Safe load error for the muted-account list.
  ///
  /// In en, this message translates to:
  /// **'Could not load muted accounts.'**
  String get settingsMutedAccountsError;

  /// Safe load error for the blocked-account list.
  ///
  /// In en, this message translates to:
  /// **'Could not load blocked accounts.'**
  String get settingsBlockedAccountsError;

  /// Retry action for relationship-list load failures.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get relationshipListRetry;

  /// Pagination action for relationship lists.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get relationshipListLoadMore;

  /// Row action in the muted-account list.
  ///
  /// In en, this message translates to:
  /// **'Unmute'**
  String get relationshipListUnmute;

  /// Row action in the blocked-account list.
  ///
  /// In en, this message translates to:
  /// **'Unblock'**
  String get relationshipListUnblock;

  /// Safe row-level relationship mutation error.
  ///
  /// In en, this message translates to:
  /// **'Could not update this account.'**
  String get relationshipListMutationError;

  /// Content-free placeholder for a muted post or quote.
  ///
  /// In en, this message translates to:
  /// **'Post from a muted account'**
  String get postMutedPlaceholder;

  /// Generic content-free placeholder for blocked or unavailable content.
  ///
  /// In en, this message translates to:
  /// **'Post unavailable'**
  String get postUnavailablePlaceholder;

  /// Temporary reveal action for muted content.
  ///
  /// In en, this message translates to:
  /// **'Show post'**
  String get postRevealAction;

  /// Label on the follow button on a visitor profile when the viewer is not yet following them.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get profileFollowAction;

  /// Label on the follow button on a visitor profile when the viewer is already following them.
  ///
  /// In en, this message translates to:
  /// **'Unfollow'**
  String get profileFollowingAction;

  /// Marker shown on a profile page for an atproto account that does not have a CraftSky profile record.
  ///
  /// In en, this message translates to:
  /// **'Non CraftSky profile'**
  String get profileNonCraftskyMarker;

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

  /// Muted placeholder shown in the Projects tab when the user has no project posts.
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
  /// **'Post'**
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

  /// Label for the regular-post option in the top-level post-type chooser.
  ///
  /// In en, this message translates to:
  /// **'Regular post'**
  String get postTypeRegularLabel;

  /// Brief description for the regular-post option in the top-level post-type chooser.
  ///
  /// In en, this message translates to:
  /// **'Share a quick update, thought or question.'**
  String get postTypeRegularDescription;

  /// Label for the project-post option in the top-level post-type chooser.
  ///
  /// In en, this message translates to:
  /// **'Project post'**
  String get postTypeProjectLabel;

  /// Brief description for the project-post option in the top-level post-type chooser.
  ///
  /// In en, this message translates to:
  /// **'Add photos and structured project details.'**
  String get postTypeProjectDescription;

  /// Title of the project composer sheet.
  ///
  /// In en, this message translates to:
  /// **'Project post'**
  String get projectComposerTitle;

  /// Button label for advancing to the next project composer page.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get projectComposerNextAction;

  /// Small marker shown beside project composer field labels when the field is required.
  ///
  /// In en, this message translates to:
  /// **'required'**
  String get projectComposerRequiredLabel;

  /// Short helper text shown above the project title on the first project composer page.
  ///
  /// In en, this message translates to:
  /// **'Fill in the details about your project'**
  String get projectComposerDetailsPrompt;

  /// Short helper text shown at the top of the optional details page in the project composer.
  ///
  /// In en, this message translates to:
  /// **'This information is optional but will help others find your project'**
  String get projectComposerOptionalDetailsPrompt;

  /// Label for the optional project-title field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Project title'**
  String get projectComposerProjectTitleLabel;

  /// Placeholder text for the optional project-title field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add a short project title'**
  String get projectComposerProjectTitleHint;

  /// Label above the main project description text field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Project description'**
  String get projectComposerDescriptionLabel;

  /// Hint text inside the main project description text field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Tell everyone about your project'**
  String get projectComposerDescriptionHint;

  /// Label for the project composer craft-type field.
  ///
  /// In en, this message translates to:
  /// **'Craft type'**
  String get projectComposerCraftTypeLabel;

  /// Label for the project composer status field.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get projectComposerStatusLabel;

  /// Label for the project composer materials field.
  ///
  /// In en, this message translates to:
  /// **'Materials'**
  String get projectComposerMaterialsLabel;

  /// Hint text for the free-text materials entry in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add material'**
  String get projectComposerMaterialsAddHint;

  /// Button label for adding a typed material in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get projectComposerMaterialsAddAction;

  /// Validation error for a project composer material entry that is too long.
  ///
  /// In en, this message translates to:
  /// **'Keep each material to {max} characters or fewer.'**
  String projectComposerMaterialsMaxLengthError(int max);

  /// Small helper label shown by reusable project fields while create is loading and controls are disabled.
  ///
  /// In en, this message translates to:
  /// **'Disabled'**
  String get projectComposerFieldDisabledLabel;

  /// Validation helper shown when a project multi-select field reaches its configured maximum.
  ///
  /// In en, this message translates to:
  /// **'You can choose up to {maxSelected}.'**
  String projectComposerMultiSelectMaxSelectedError(int maxSelected);

  /// Label for the project composer colours field.
  ///
  /// In en, this message translates to:
  /// **'Colours'**
  String get projectComposerColoursLabel;

  /// Placeholder text for searching known colour options in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Search colours'**
  String get projectComposerColoursSearchHint;

  /// Label for the project composer design-tags field.
  ///
  /// In en, this message translates to:
  /// **'Design tags'**
  String get projectComposerDesignTagsLabel;

  /// Placeholder text for searching known design-tag options in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Search design tags'**
  String get projectComposerDesignTagsSearchHint;

  /// Action that reveals optional project pattern fields in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add pattern'**
  String get projectComposerAddPatternAction;

  /// Disclosure label for the optional pattern section in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Pattern'**
  String get projectComposerPatternSectionLabel;

  /// Section title shown above optional pattern details after a pattern tag or name is entered.
  ///
  /// In en, this message translates to:
  /// **'Pattern info'**
  String get projectComposerPatternInfoSectionLabel;

  /// Disclosure label for optional craft-specific project detail fields.
  ///
  /// In en, this message translates to:
  /// **'More project details'**
  String get projectComposerMoreDetailsLabel;

  /// Empty-state text shown in the craft-specific project details section before a craft type is selected.
  ///
  /// In en, this message translates to:
  /// **'Select Craft Type'**
  String get projectComposerSelectCraftTypeEmptyState;

  /// Label for the sewing project-type field in project details.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get projectComposerSewingProjectTypeLabel;

  /// Label for the craft-specific project subtype field in project details.
  ///
  /// In en, this message translates to:
  /// **'Project subtype'**
  String get projectComposerProjectSubtypeLabel;

  /// Label for the sewing size-made detail field.
  ///
  /// In en, this message translates to:
  /// **'Size made'**
  String get projectComposerSizeMadeLabel;

  /// Placeholder text for the sewing size-made detail field.
  ///
  /// In en, this message translates to:
  /// **'e.g. Medium or custom measurements'**
  String get projectComposerSizeMadeHint;

  /// Label for the sewing fit-notes detail field.
  ///
  /// In en, this message translates to:
  /// **'Fit notes'**
  String get projectComposerFitNotesLabel;

  /// Placeholder text for the sewing fit-notes detail field.
  ///
  /// In en, this message translates to:
  /// **'Add fit notes'**
  String get projectComposerFitNotesHint;

  /// Label for the knitting project-type field in project details.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get projectComposerKnittingProjectTypeLabel;

  /// Label for the crochet project-type field in project details.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get projectComposerCrochetProjectTypeLabel;

  /// Label for the quilting project-type field in project details.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get projectComposerQuiltingProjectTypeLabel;

  /// Label for yarn-weight detail fields.
  ///
  /// In en, this message translates to:
  /// **'Yarn weight'**
  String get projectComposerYarnWeightLabel;

  /// Label for the knitting needle-size detail field.
  ///
  /// In en, this message translates to:
  /// **'Needle size'**
  String get projectComposerNeedleSizeLabel;

  /// Label for the crochet hook-size detail field.
  ///
  /// In en, this message translates to:
  /// **'Hook size'**
  String get projectComposerHookSizeLabel;

  /// Label for gauge stitches input in project details.
  ///
  /// In en, this message translates to:
  /// **'Gauge stitches'**
  String get projectComposerGaugeStitchesLabel;

  /// Placeholder text for gauge stitch count fields.
  ///
  /// In en, this message translates to:
  /// **'Stitches'**
  String get projectComposerGaugeStitchesHint;

  /// Label for optional gauge rows input in project details.
  ///
  /// In en, this message translates to:
  /// **'Gauge rows'**
  String get projectComposerGaugeRowsLabel;

  /// Placeholder text for optional gauge row count fields.
  ///
  /// In en, this message translates to:
  /// **'Rows'**
  String get projectComposerGaugeRowsHint;

  /// Label for gauge measurement input in project details.
  ///
  /// In en, this message translates to:
  /// **'Gauge measurement'**
  String get projectComposerGaugeMeasurementLabel;

  /// Placeholder text for gauge measurement fields.
  ///
  /// In en, this message translates to:
  /// **'Measurement'**
  String get projectComposerGaugeMeasurementHint;

  /// Label for gauge unit selection in project details.
  ///
  /// In en, this message translates to:
  /// **'Gauge unit'**
  String get projectComposerGaugeUnitLabel;

  /// Label for finished-size detail fields.
  ///
  /// In en, this message translates to:
  /// **'Finished size'**
  String get projectComposerFinishedSizeLabel;

  /// Placeholder text for finished-size detail fields.
  ///
  /// In en, this message translates to:
  /// **'Add finished size'**
  String get projectComposerFinishedSizeHint;

  /// Label for the quilting size detail field.
  ///
  /// In en, this message translates to:
  /// **'Size'**
  String get projectComposerSizeLabel;

  /// Label for the quilting piecing-technique detail field.
  ///
  /// In en, this message translates to:
  /// **'Piecing technique'**
  String get projectComposerPiecingTechniqueLabel;

  /// Label for the quilting method detail field.
  ///
  /// In en, this message translates to:
  /// **'Quilting method'**
  String get projectComposerQuiltingMethodLabel;

  /// Validation error shown when submitting a project post without body text.
  ///
  /// In en, this message translates to:
  /// **'Add body text.'**
  String get projectComposerBodyRequiredError;

  /// Validation error shown when submitting a project post without selecting a craft type.
  ///
  /// In en, this message translates to:
  /// **'Choose a craft type.'**
  String get projectComposerCraftRequiredError;

  /// Validation error shown when submitting a project post without a photo.
  ///
  /// In en, this message translates to:
  /// **'Add at least one photo.'**
  String get projectComposerPhotoRequiredError;

  /// Validation error shown when gauge values are partial, missing a unit, or not positive whole numbers.
  ///
  /// In en, this message translates to:
  /// **'Complete the gauge or clear it.'**
  String get projectComposerGaugeInvalidError;

  /// Label for the optional pattern tag-or-name field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Pattern tag or name'**
  String get projectComposerPatternNameLabel;

  /// Placeholder text for the optional pattern-name field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add pattern name'**
  String get projectComposerPatternNameHint;

  /// Label for the optional pattern URL field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Link'**
  String get projectComposerPatternUrlLabel;

  /// Placeholder text for the optional pattern URL field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'https://example.com/pattern'**
  String get projectComposerPatternUrlHint;

  /// Label for the optional pattern difficulty field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Difficulty'**
  String get projectComposerPatternDifficultyLabel;

  /// Label for the optional pattern designer field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Designer'**
  String get projectComposerPatternDesignerLabel;

  /// Placeholder text for the optional pattern designer field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add pattern designer'**
  String get projectComposerPatternDesignerHint;

  /// Label for the optional pattern publisher field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Publisher'**
  String get projectComposerPatternPublisherLabel;

  /// Placeholder text for the optional pattern publisher field in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add pattern publisher'**
  String get projectComposerPatternPublisherHint;

  /// Label above the text field in the text-only post composer.
  ///
  /// In en, this message translates to:
  /// **'What are you making?'**
  String get postComposeHint;

  /// Hint text inside the main post composer text field.
  ///
  /// In en, this message translates to:
  /// **'Pattern, fabric, what went right, what didn\'t...'**
  String get postComposeBodyHint;

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

  /// Title of the confirm-discard dialog shown when leaving the post composer with unsaved edits.
  ///
  /// In en, this message translates to:
  /// **'Discard draft?'**
  String get postComposeDiscardTitle;

  /// Body of the confirm-discard dialog shown when leaving the post composer with unsaved edits.
  ///
  /// In en, this message translates to:
  /// **'Your draft won\'t be saved.'**
  String get postComposeDiscardMessage;

  /// Confirm action in the post-composer confirm-discard dialog — closes the composer without saving.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get postComposeDiscardConfirm;

  /// Cancel action in the post-composer confirm-discard dialog — returns the user to the composer.
  ///
  /// In en, this message translates to:
  /// **'Keep editing'**
  String get postComposeDiscardCancel;

  /// Snackbar error shown when the post composer image limit is reached.
  ///
  /// In en, this message translates to:
  /// **'You can add up to {maxImages} images'**
  String postComposeImageLimitError(int maxImages);

  /// Snackbar error shown when one or more selected composer images use an unsupported format.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{Unsupported image type} other{{count} unsupported images}}'**
  String postComposeUnsupportedImagesError(int count);

  /// Snackbar error shown when the image picker cannot be opened.
  ///
  /// In en, this message translates to:
  /// **'Could not open image picker'**
  String get postComposeImagePickerError;

  /// Title of the confirm dialog shown before posting images that are missing alt text.
  ///
  /// In en, this message translates to:
  /// **'Some images do not have alt text'**
  String get postComposeMissingAltTitle;

  /// Body of the confirm dialog shown before posting images that are missing alt text.
  ///
  /// In en, this message translates to:
  /// **'Do you wish to post anyway?'**
  String get postComposeMissingAltMessage;

  /// Confirm button label for posting despite missing image alt text.
  ///
  /// In en, this message translates to:
  /// **'Post anyway'**
  String get postComposeMissingAltConfirm;

  /// Cancel button label for returning to the composer to add image alt text.
  ///
  /// In en, this message translates to:
  /// **'Go back'**
  String get postComposeMissingAltCancel;

  /// Heading above the post composer photo attachment controls.
  ///
  /// In en, this message translates to:
  /// **'Photos'**
  String get postComposePhotosTitle;

  /// Alt-text completion status shown before any composer photos are attached.
  ///
  /// In en, this message translates to:
  /// **'0 described'**
  String get postComposeNoImagesDescribed;

  /// Alt-text completion status for attached composer photos.
  ///
  /// In en, this message translates to:
  /// **'{describedCount} / {imageCount} described'**
  String postComposeImagesDescribed(int describedCount, int imageCount);

  /// Helper text under the photo heading before photos are attached.
  ///
  /// In en, this message translates to:
  /// **'Up to {maxImages} photos'**
  String postComposePhotosLimitHelper(int maxImages);

  /// Helper text under the photo heading once photos are attached.
  ///
  /// In en, this message translates to:
  /// **'{imageCount}/{maxImages} · drag to reorder · first is the cover'**
  String postComposePhotosReorderHelper(int imageCount, int maxImages);

  /// Tooltip for moving an attached composer image earlier in the order.
  ///
  /// In en, this message translates to:
  /// **'Move image up'**
  String get postComposeMoveImageUp;

  /// Tooltip for moving an attached composer image later in the order.
  ///
  /// In en, this message translates to:
  /// **'Move image down'**
  String get postComposeMoveImageDown;

  /// Tooltip for removing an attached composer image.
  ///
  /// In en, this message translates to:
  /// **'Remove image'**
  String get postComposeRemoveImage;

  /// Tooltip for the composer image drag handle.
  ///
  /// In en, this message translates to:
  /// **'Drag to reorder'**
  String get postComposeDragToReorder;

  /// Uppercase label above an attached image alt-text field.
  ///
  /// In en, this message translates to:
  /// **'ALT TEXT'**
  String get postComposeAltTextLabel;

  /// Hint text inside an attached image alt-text field.
  ///
  /// In en, this message translates to:
  /// **'Describe the image for someone who cannot see it, including the craft, materials, colors, and important details.'**
  String get postComposeAltTextHint;

  /// Status text shown beside an image alt-text field once alt text is present.
  ///
  /// In en, this message translates to:
  /// **'Described'**
  String get postComposeImageDescribed;

  /// Status text shown beside an image alt-text field when alt text is missing.
  ///
  /// In en, this message translates to:
  /// **'Help screen readers'**
  String get postComposeImageNeedsAltText;

  /// Label on the composer card for adding the first photo.
  ///
  /// In en, this message translates to:
  /// **'Add a photo'**
  String get postComposeAddPhoto;

  /// Label on the composer card for adding another photo.
  ///
  /// In en, this message translates to:
  /// **'Add another photo'**
  String get postComposeAddAnotherPhoto;

  /// Subtitle on the add-photo card showing how many more photos can be attached.
  ///
  /// In en, this message translates to:
  /// **'Up to {remainingCount} more'**
  String postComposePhotosRemaining(int remainingCount);

  /// Status shown while a composer image file is being read.
  ///
  /// In en, this message translates to:
  /// **'Reading image'**
  String get postComposeReadingImage;

  /// Status shown while a composer image is being resized or encoded.
  ///
  /// In en, this message translates to:
  /// **'Preparing image'**
  String get postComposePreparingImage;

  /// Status shown while a composer image is uploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading image'**
  String get postComposeUploadingImage;

  /// Status shown after a composer image upload succeeds.
  ///
  /// In en, this message translates to:
  /// **'Uploaded'**
  String get postComposeUploadedImage;

  /// Status shown after a composer image upload fails.
  ///
  /// In en, this message translates to:
  /// **'Failed'**
  String get postComposeImageFailed;

  /// Overlay label shown while the server is finalizing a composer image upload.
  ///
  /// In en, this message translates to:
  /// **'Processing'**
  String get postComposeProcessingImage;

  /// Overlay label showing composer image upload progress percentage.
  ///
  /// In en, this message translates to:
  /// **'Uploading {percent}%'**
  String postComposeUploadingProgress(int percent);

  /// Tooltip for liking a post.
  ///
  /// In en, this message translates to:
  /// **'Like'**
  String get postLikeAction;

  /// Tooltip for removing a like from a post.
  ///
  /// In en, this message translates to:
  /// **'Unlike'**
  String get postUnlikeAction;

  /// Snackbar shown when liking or unliking a post fails and the previous state is restored.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t update like.'**
  String get postLikeError;

  /// Tooltip and optional label for replying to a post.
  ///
  /// In en, this message translates to:
  /// **'Reply'**
  String get postReplyAction;

  /// Tooltip for reposting a post.
  ///
  /// In en, this message translates to:
  /// **'Repost'**
  String get postRepostAction;

  /// Tooltip for removing a repost.
  ///
  /// In en, this message translates to:
  /// **'Unrepost'**
  String get postUnrepostAction;

  /// Menu label for creating a quote post.
  ///
  /// In en, this message translates to:
  /// **'Quote'**
  String get postQuoteAction;

  /// Tooltip for opening repost and quote actions for a post.
  ///
  /// In en, this message translates to:
  /// **'Share'**
  String get postShareAction;

  /// Timeline attribution shown above a post when a followed account reposted it.
  ///
  /// In en, this message translates to:
  /// **'Reposted by {name}'**
  String postRepostedBy(String name);

  /// Placeholder shown when a quoted post is hidden by moderation or policy.
  ///
  /// In en, this message translates to:
  /// **'Quoted post hidden'**
  String get postQuoteHidden;

  /// Placeholder shown when a quoted post is missing, deleted, or unavailable.
  ///
  /// In en, this message translates to:
  /// **'Quoted post unavailable'**
  String get postQuoteUnavailable;

  /// Tooltip and menu label for deleting a post.
  ///
  /// In en, this message translates to:
  /// **'Delete post'**
  String get postDeleteAction;

  /// Menu label for reporting a post.
  ///
  /// In en, this message translates to:
  /// **'Report post'**
  String get postReportAction;

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
  /// **'This removes the post from CraftSky.'**
  String get postDeleteMessage;

  /// Body text of the delete-comment confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This removes the comment from CraftSky.'**
  String get commentDeleteMessage;

  /// Body text of the delete-reply confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This removes the reply from CraftSky.'**
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

  /// Snackbar shown when a follow or unfollow request fails and the previous state is restored.
  ///
  /// In en, this message translates to:
  /// **'Could not update follow state.'**
  String get profileFollowToggleError;

  /// Snackbar shown when tapping Share while share wiring isn't implemented yet.
  ///
  /// In en, this message translates to:
  /// **'Share coming soon.'**
  String get profileShareComingSoon;

  /// Tooltip/action label for reporting a visitor profile.
  ///
  /// In en, this message translates to:
  /// **'Report profile'**
  String get profileReportAction;

  /// Generic inline warning copy for a warned post.
  ///
  /// In en, this message translates to:
  /// **'This post may not follow CraftSky community guidelines.'**
  String get moderationWarningPost;

  /// Generic inline warning copy for a warned profile.
  ///
  /// In en, this message translates to:
  /// **'This profile may not follow CraftSky community guidelines.'**
  String get moderationWarningProfile;

  /// Generic inline warning copy for posts by a warned author.
  ///
  /// In en, this message translates to:
  /// **'This author may not follow CraftSky community guidelines.'**
  String get moderationWarningAuthor;

  /// Primary action label in the report dialog/sheet.
  ///
  /// In en, this message translates to:
  /// **'Submit'**
  String get reportSubmit;

  /// Primary action label while a report submission is in flight.
  ///
  /// In en, this message translates to:
  /// **'Submitting…'**
  String get reportSubmitting;

  /// Snackbar shown after a report submission succeeds.
  ///
  /// In en, this message translates to:
  /// **'Thanks — your report was submitted.'**
  String get reportSubmitSuccess;

  /// Inline error shown when a report submission fails and can be retried.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t submit report. Please try again.'**
  String get reportSubmitError;

  /// Label for private optional details text field in the report dialog/sheet.
  ///
  /// In en, this message translates to:
  /// **'Details'**
  String get reportDetailsLabel;

  /// Validation error when optional report details exceed the maximum length.
  ///
  /// In en, this message translates to:
  /// **'Details must be 1000 characters or fewer.'**
  String get reportDetailsTooLong;

  /// Section title above the report reason choices.
  ///
  /// In en, this message translates to:
  /// **'Reason'**
  String get reportReasonTitle;

  /// Report reason label for harassment.
  ///
  /// In en, this message translates to:
  /// **'Harassment'**
  String get reportReasonHarassment;

  /// Report reason label for hate.
  ///
  /// In en, this message translates to:
  /// **'Hate'**
  String get reportReasonHate;

  /// Report reason label for spam.
  ///
  /// In en, this message translates to:
  /// **'Spam'**
  String get reportReasonSpam;

  /// Report reason label for misleading content.
  ///
  /// In en, this message translates to:
  /// **'Misleading'**
  String get reportReasonMisleading;

  /// Report reason label for suspected AI-generated content.
  ///
  /// In en, this message translates to:
  /// **'Suspected AI-generated'**
  String get reportReasonSuspectedAiGenerated;

  /// Report reason label for adult or graphic content.
  ///
  /// In en, this message translates to:
  /// **'Adult or graphic'**
  String get reportReasonAdultOrGraphic;

  /// Report reason label for impersonation.
  ///
  /// In en, this message translates to:
  /// **'Impersonation'**
  String get reportReasonImpersonation;

  /// Report reason label for off-topic content.
  ///
  /// In en, this message translates to:
  /// **'Off-topic'**
  String get reportReasonOffTopic;

  /// Report reason label for intellectual-property concerns.
  ///
  /// In en, this message translates to:
  /// **'Intellectual property'**
  String get reportReasonIntellectualProperty;

  /// Report reason label for other concerns.
  ///
  /// In en, this message translates to:
  /// **'Other'**
  String get reportReasonOther;

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

  /// Tooltip/action label for choosing a new profile avatar image.
  ///
  /// In en, this message translates to:
  /// **'Change avatar'**
  String get editProfileChangeAvatar;

  /// Action label for choosing a new profile cover/banner image.
  ///
  /// In en, this message translates to:
  /// **'Change cover'**
  String get editProfileChangeCover;

  /// Snackbar shown when a selected profile avatar or cover image cannot be prepared or uploaded.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t upload that photo.'**
  String get editProfilePhotoUploadError;

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

  /// Title for the search page.
  ///
  /// In en, this message translates to:
  /// **'Search'**
  String get searchTitle;

  /// Placeholder text in the search field.
  ///
  /// In en, this message translates to:
  /// **'Search hashtags, people or projects...'**
  String get searchHint;

  /// Action next to the focused search input that returns to the blank search page.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get searchCancelAction;

  /// Tooltip for clearing the current search text.
  ///
  /// In en, this message translates to:
  /// **'Clear search text'**
  String get searchClearAction;

  /// Heading above the user's recent searches on the blank search page.
  ///
  /// In en, this message translates to:
  /// **'Recent searches'**
  String get searchRecentHeading;

  /// Tooltip for deleting one recent search.
  ///
  /// In en, this message translates to:
  /// **'Delete recent search'**
  String get searchDeleteRecentAction;

  /// Heading above craft-grouped trending hashtags on the blank search page.
  ///
  /// In en, this message translates to:
  /// **'Trending hashtags'**
  String get searchTrendingHashtagsHeading;

  /// Heading for profile suggestions or profile results.
  ///
  /// In en, this message translates to:
  /// **'Profiles'**
  String get searchProfilesHeading;

  /// Heading for hashtag suggestions or hashtag results.
  ///
  /// In en, this message translates to:
  /// **'Hashtags'**
  String get searchHashtagsHeading;

  /// Action that opens the full results tab for a suggestion section.
  ///
  /// In en, this message translates to:
  /// **'View all'**
  String get searchViewAllAction;

  /// Tab label for submitted search post results.
  ///
  /// In en, this message translates to:
  /// **'Posts'**
  String get searchTabPosts;

  /// Tab label for submitted search project results.
  ///
  /// In en, this message translates to:
  /// **'Projects'**
  String get searchTabProjects;

  /// Tab label for submitted search profile results.
  ///
  /// In en, this message translates to:
  /// **'Profiles'**
  String get searchTabProfiles;

  /// Tab label for submitted search hashtag results.
  ///
  /// In en, this message translates to:
  /// **'Tags'**
  String get searchTabTags;

  /// Empty state for submitted search Posts tab.
  ///
  /// In en, this message translates to:
  /// **'No posts found.'**
  String get searchEmptyPosts;

  /// Empty state for submitted search Projects tab.
  ///
  /// In en, this message translates to:
  /// **'No projects found.'**
  String get searchEmptyProjects;

  /// Empty state for submitted search Profiles tab.
  ///
  /// In en, this message translates to:
  /// **'No profiles found.'**
  String get searchEmptyProfiles;

  /// Empty state for submitted search Tags tab.
  ///
  /// In en, this message translates to:
  /// **'No tags found.'**
  String get searchEmptyTags;

  /// Error title shown when search results fail to load.
  ///
  /// In en, this message translates to:
  /// **'Search didn\'t load.'**
  String get searchLoadError;

  /// Snackbar shown when saving a recent search fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save recent search.'**
  String get searchRecentSaveError;

  /// Snackbar shown when deleting a recent search fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t delete recent search.'**
  String get searchRecentDeleteError;

  /// Post count label for a hashtag suggestion or result.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 post} other{{count} posts}}'**
  String searchTagPostCount(int count);

  /// Subtitle for a profile suggestion combining display name and craft labels.
  ///
  /// In en, this message translates to:
  /// **'{name} • {crafts}'**
  String searchProfileCraftSubtitle(String name, String crafts);

  /// Sort label for chronological/newest search and project results.
  ///
  /// In en, this message translates to:
  /// **'Newest'**
  String get searchSortNewest;

  /// Description for the newest sort menu item.
  ///
  /// In en, this message translates to:
  /// **'Show the newest items first.'**
  String get searchSortNewestDescription;

  /// Sort label for popular search and project results.
  ///
  /// In en, this message translates to:
  /// **'Popular'**
  String get searchSortPopular;

  /// Description for the popular sort menu item.
  ///
  /// In en, this message translates to:
  /// **'Show the most popular items first.'**
  String get searchSortPopularDescription;

  /// Title for an exact hashtag search page.
  ///
  /// In en, this message translates to:
  /// **'#{tag}'**
  String tagSearchTitle(String tag);

  /// Empty state for an exact hashtag feed.
  ///
  /// In en, this message translates to:
  /// **'No posts found for this tag.'**
  String get tagSearchEmpty;

  /// Title for the Projects browse page.
  ///
  /// In en, this message translates to:
  /// **'Projects'**
  String get projectsTitle;

  /// Button label opening the project filters sheet.
  ///
  /// In en, this message translates to:
  /// **'Filters'**
  String get projectsFilterAction;

  /// Title for the project filters sheet scoped to the selected craft.
  ///
  /// In en, this message translates to:
  /// **'Filter {craft} projects'**
  String projectsFiltersTitle(String craft);

  /// Read-only craft context label in the project filters sheet.
  ///
  /// In en, this message translates to:
  /// **'Browsing {craft}'**
  String projectsCraftContext(String craft);

  /// Project filter group label for project type.
  ///
  /// In en, this message translates to:
  /// **'Project type'**
  String get projectsFilterProjectType;

  /// Project filter group label for pattern difficulty.
  ///
  /// In en, this message translates to:
  /// **'Pattern difficulty'**
  String get projectsFilterDifficulty;

  /// Project filter group label for colors.
  ///
  /// In en, this message translates to:
  /// **'Color'**
  String get projectsFilterColor;

  /// Project filter group label for design tags.
  ///
  /// In en, this message translates to:
  /// **'Design tag'**
  String get projectsFilterDesignTag;

  /// Project filter group label for material free-text filters.
  ///
  /// In en, this message translates to:
  /// **'Material'**
  String get projectsFilterMaterial;

  /// Project filter group label for project tag free-text filters.
  ///
  /// In en, this message translates to:
  /// **'Project tag'**
  String get projectsFilterProjectTag;

  /// Hint text for adding a free-text project filter chip.
  ///
  /// In en, this message translates to:
  /// **'Add a value'**
  String get projectsFreeTextHint;

  /// Button label for adding a free-text project filter value.
  ///
  /// In en, this message translates to:
  /// **'Add'**
  String get projectsAddFilterValueAction;

  /// Primary action in the project filters sheet.
  ///
  /// In en, this message translates to:
  /// **'Apply filters'**
  String get projectsApplyFiltersAction;

  /// Action clearing all project filters.
  ///
  /// In en, this message translates to:
  /// **'Clear all'**
  String get projectsClearFiltersAction;

  /// Empty state on the Projects browse page.
  ///
  /// In en, this message translates to:
  /// **'No projects found.'**
  String get projectsEmpty;

  /// Error title shown when the Projects browse feed fails to load.
  ///
  /// In en, this message translates to:
  /// **'Projects didn\'t load.'**
  String get projectsLoadError;
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
