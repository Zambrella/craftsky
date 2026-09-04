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

  /// Empty-feed action that opens Instagram migration.
  ///
  /// In en, this message translates to:
  /// **'Connect Instagram'**
  String get feedConnectInstagramAction;

  /// Tooltip and accessible label for the button that opens the app navigation drawer.
  ///
  /// In en, this message translates to:
  /// **'Open navigation menu'**
  String get navigationMenuTooltip;

  /// Label for the signed-in member's Profile navigation destination.
  ///
  /// In en, this message translates to:
  /// **'Profile'**
  String get navigationProfile;

  /// Compact label for the Saved posts navigation destination.
  ///
  /// In en, this message translates to:
  /// **'Saved'**
  String get navigationSaved;

  /// Compact label for the Scheduled posts navigation destination.
  ///
  /// In en, this message translates to:
  /// **'Scheduled'**
  String get navigationScheduled;

  /// Label for the external Terms link in app navigation.
  ///
  /// In en, this message translates to:
  /// **'Terms'**
  String get navigationTerms;

  /// Label for the external Privacy link in app navigation.
  ///
  /// In en, this message translates to:
  /// **'Privacy'**
  String get navigationPrivacy;

  /// Label for the currently inert Feedback control in app navigation.
  ///
  /// In en, this message translates to:
  /// **'Feedback'**
  String get navigationFeedback;

  /// App version and build number shown below Feedback in app navigation.
  ///
  /// In en, this message translates to:
  /// **'{version} ({buildNumber})'**
  String navigationBuildVersion(String version, String buildNumber);

  /// Safe error shown when an external navigation link cannot be opened.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t open that link.'**
  String get navigationLinkOpenError;

  /// Title of the external-link confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'Open link?'**
  String get externalLinkConfirmTitle;

  /// Body of the external-link confirmation dialog.
  ///
  /// In en, this message translates to:
  /// **'This will open outside CraftSky.'**
  String get externalLinkConfirmBody;

  /// Confirmation action for opening an external link.
  ///
  /// In en, this message translates to:
  /// **'Open link'**
  String get externalLinkConfirmAction;

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

  /// Notification shown when an Instagram following import matches a CraftSky account. It does not imply an automatic follow.
  ///
  /// In en, this message translates to:
  /// **'You found {actor} through your Instagram following'**
  String notificationInstagramMatchActorRow(String actor);

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
  /// **'Comments & replies'**
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

  /// No description provided for @notificationCategoryInstagramMatches.
  ///
  /// In en, this message translates to:
  /// **'Instagram matches'**
  String get notificationCategoryInstagramMatches;

  /// No description provided for @notificationInstagramMatchPreferenceDescription.
  ///
  /// In en, this message translates to:
  /// **'Push alerts are based on your private Instagram matches and never name the matched account.'**
  String get notificationInstagramMatchPreferenceDescription;

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

  /// Legacy title for the welcome route.
  ///
  /// In en, this message translates to:
  /// **'Welcome'**
  String get welcomeTitle;

  /// Main heading on the signed-out welcome page.
  ///
  /// In en, this message translates to:
  /// **'Join CraftSky'**
  String get welcomeJoinTitle;

  /// Short introduction beneath the welcome heading.
  ///
  /// In en, this message translates to:
  /// **'Share what you make. Find people who make what you love.'**
  String get welcomeSubtitle;

  /// Primary registration action on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'Register'**
  String get welcomeRegisterAction;

  /// Disabled welcome-page action label while the browser redirect is being prepared.
  ///
  /// In en, this message translates to:
  /// **'Redirecting...'**
  String get welcomeRedirectingAction;

  /// Short note beneath Register explaining the external Bluesky account-creation handoff.
  ///
  /// In en, this message translates to:
  /// **'You\'ll create your account with Bluesky, then return to CraftSky.'**
  String get welcomeRegistrationHandoff;

  /// Divider between registration and sign-in on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'Or'**
  String get welcomeOr;

  /// Primary button label on the welcome page that opens sign-in.
  ///
  /// In en, this message translates to:
  /// **'Sign in'**
  String get welcomeSignInAction;

  /// Action that starts provider-first account registration.
  ///
  /// In en, this message translates to:
  /// **'Create an account'**
  String get welcomeCreateAccountAction;

  /// Provider disclosure shown immediately before each account-registration action.
  ///
  /// In en, this message translates to:
  /// **'Bluesky hosts your portable account, which you can use with Craftsky.'**
  String get registrationProviderDisclosure;

  /// Expandable explainer heading on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'What is an Atmosphere account?'**
  String get welcomeAtmosphereTitle;

  /// Explains account portability and compatible providers on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'CraftSky is built on the AT Protocol, so your account, posts and social graph are portable. You can use an existing Bluesky or compatible Atmosphere account, or register a new one.'**
  String get welcomeAtmosphereBody;

  /// Text before the Terms link on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'By continuing, you agree to our'**
  String get welcomeLegalPrefix;

  /// Text joining the Terms and Privacy links on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'and'**
  String get welcomeLegalAnd;

  /// Privacy policy link label on the welcome page.
  ///
  /// In en, this message translates to:
  /// **'Privacy Policy'**
  String get welcomePrivacyAction;

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
  /// **'Your Atmosphere Handle'**
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

  /// Error shown when the OAuth handoff code or confirmation receipt is invalid or expired.
  ///
  /// In en, this message translates to:
  /// **'That sign-in link expired. Please sign in again.'**
  String get authCompleteTimedOutError;

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

  /// Safe callback message shown when provider-first registration is canceled.
  ///
  /// In en, this message translates to:
  /// **'Account creation was canceled.'**
  String get authRegistrationCanceledError;

  /// Safe callback message shown when the registration provider is temporarily unavailable.
  ///
  /// In en, this message translates to:
  /// **'Bluesky is temporarily unavailable. Please try again.'**
  String get authRegistrationProviderUnavailableError;

  /// Safe callback message shown when provider-first registration could not be verified or completed.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t verify or complete account creation.'**
  String get authRegistrationIncompleteError;

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

  /// Headline shown when account-critical initialization fails.
  ///
  /// In en, this message translates to:
  /// **'We couldn’t load this account'**
  String get activeAccountInitializationFailedTitle;

  /// Recovery guidance shown when account-critical initialization fails.
  ///
  /// In en, this message translates to:
  /// **'Try again, switch accounts, or sign out.'**
  String get activeAccountInitializationFailedBody;

  /// Recovery action for choosing another retained account.
  ///
  /// In en, this message translates to:
  /// **'Switch account'**
  String get activeAccountSwitchAction;

  /// Recovery action for signing out the account that failed to initialize.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get activeAccountSignOutAction;

  /// Non-sensitive feedback when an account recovery action fails.
  ///
  /// In en, this message translates to:
  /// **'That didn’t work. Please try again.'**
  String get activeAccountRecoveryFailed;

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

  /// Label on the secondary action in a profile card, opens the full profile page.
  ///
  /// In en, this message translates to:
  /// **'Visit profile'**
  String get profileVisitAction;

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
  /// **'Comments & replies'**
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

  /// Empty state shown in the profile Comments & replies tab when the user has not responded to a post.
  ///
  /// In en, this message translates to:
  /// **'No comments or replies yet.'**
  String get profileCommentsEmpty;

  /// Error title shown in the profile Comments & replies tab when response fetching fails.
  ///
  /// In en, this message translates to:
  /// **'Comments and replies didn\'t load.'**
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

  /// Empty state shown on a post thread when the root post has no comments.
  ///
  /// In en, this message translates to:
  /// **'No comments yet.'**
  String get postThreadEmptyReplies;

  /// Supporting copy beneath the empty comment-thread title.
  ///
  /// In en, this message translates to:
  /// **'Start the conversation with a comment.'**
  String get postThreadEmptyCommentsSubtitle;

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

  /// Heading above the optional detail actions in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Add more details'**
  String get projectComposerMoreDetailsHeading;

  /// Helper text above the optional detail actions in the project composer.
  ///
  /// In en, this message translates to:
  /// **'Optional details help other crafters discover and understand your project.'**
  String get projectComposerMoreDetailsPrompt;

  /// Title of the optional pattern details action and page.
  ///
  /// In en, this message translates to:
  /// **'Pattern details'**
  String get projectComposerPatternDetailsTitle;

  /// Description shown when no optional pattern details have been added.
  ///
  /// In en, this message translates to:
  /// **'Add designer, publisher, link and difficulty'**
  String get projectComposerPatternDetailsDescription;

  /// Title of the common optional project metadata action and page.
  ///
  /// In en, this message translates to:
  /// **'Materials and style'**
  String get projectComposerCommonDetailsTitle;

  /// Description shown when no materials or style details have been added.
  ///
  /// In en, this message translates to:
  /// **'Help others discover projects like yours'**
  String get projectComposerCommonDetailsDescription;

  /// Title of the selected craft's optional detail action and page.
  ///
  /// In en, this message translates to:
  /// **'{craft} details'**
  String projectComposerCraftDetailsTitle(String craft);

  /// Description shown when no craft-specific details have been added.
  ///
  /// In en, this message translates to:
  /// **'Add project type and craft-specific details'**
  String get projectComposerCraftDetailsDescription;

  /// Summary count for a populated optional project detail group.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 detail added} other{{count} details added}}'**
  String projectComposerDetailsAdded(int count);

  /// Notice shown after changing craft type clears populated craft-specific details.
  ///
  /// In en, this message translates to:
  /// **'Craft details cleared.'**
  String get projectComposerCraftCleared;

  /// Notice shown after removing a pattern name clears populated pattern details.
  ///
  /// In en, this message translates to:
  /// **'Pattern details cleared.'**
  String get projectComposerPatternCleared;

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

  /// Title of the composer when commenting directly on a root post.
  ///
  /// In en, this message translates to:
  /// **'Comment'**
  String get postComposeCommentTitle;

  /// Label above the text field in reply mode.
  ///
  /// In en, this message translates to:
  /// **'Write your reply'**
  String get postComposeReplyHint;

  /// Label above the text field when commenting directly on a root post.
  ///
  /// In en, this message translates to:
  /// **'Write your comment'**
  String get postComposeCommentHint;

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

  /// Submit button label when commenting directly on a root post.
  ///
  /// In en, this message translates to:
  /// **'Comment'**
  String get postComposeCommentSubmit;

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

  /// Subtle provenance label for a post created by the Instagram historical importer. It does not imply account ownership verification.
  ///
  /// In en, this message translates to:
  /// **'Imported from Instagram'**
  String get postImportedFromInstagram;

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

  /// Menu and sheet title for reporting a top-level comment.
  ///
  /// In en, this message translates to:
  /// **'Report comment'**
  String get commentReportAction;

  /// Menu and sheet title for reporting a nested reply.
  ///
  /// In en, this message translates to:
  /// **'Report reply'**
  String get replyReportAction;

  /// Menu label for pinning an eligible post to its owner's profile.
  ///
  /// In en, this message translates to:
  /// **'Pin post'**
  String get postPinAction;

  /// Menu label for removing the current profile pin.
  ///
  /// In en, this message translates to:
  /// **'Unpin post'**
  String get postUnpinAction;

  /// Non-interactive profile-card attribution identifying the visible pinned post.
  ///
  /// In en, this message translates to:
  /// **'Pinned post'**
  String get postPinnedAnnotation;

  /// Confirmation shown after a post is pinned.
  ///
  /// In en, this message translates to:
  /// **'Post pinned'**
  String get postPinSuccess;

  /// Confirmation shown after a post is unpinned.
  ///
  /// In en, this message translates to:
  /// **'Post unpinned'**
  String get postUnpinSuccess;

  /// Retry message shown when pinning a post fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn’t pin post. Try again.'**
  String get postPinError;

  /// Retry message shown when unpinning a post fails.
  ///
  /// In en, this message translates to:
  /// **'Couldn’t unpin post. Try again.'**
  String get postUnpinError;

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

  /// Snackbar shown after successfully deleting a top-level comment.
  ///
  /// In en, this message translates to:
  /// **'Comment deleted.'**
  String get commentDeleteSuccess;

  /// Snackbar shown after successfully deleting a nested reply.
  ///
  /// In en, this message translates to:
  /// **'Reply deleted.'**
  String get replyDeleteSuccess;

  /// Snackbar shown when deleting a comment or reply fails and its exact role is unavailable.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t delete that comment or reply.'**
  String get responseDeleteError;

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

  /// Snackbar identifying the business portion of a profile save as failed.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save your business details.'**
  String get editProfileBusinessSaveError;

  /// Snackbar shown when both records in a combined profile save fail.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save your profile or business details.'**
  String get editProfileBothSaveError;

  /// Snackbar shown when the business declaration CID is stale. Full reload conflict UX is implemented later.
  ///
  /// In en, this message translates to:
  /// **'Your business details changed elsewhere. Reload before saving them again.'**
  String get editProfileBusinessConflictError;

  /// Heading for business-only fields in Edit Profile.
  ///
  /// In en, this message translates to:
  /// **'Business details'**
  String get editProfileBusinessHeading;

  /// Helper text for business-only profile fields.
  ///
  /// In en, this message translates to:
  /// **'These details appear on your public business profile.'**
  String get editProfileBusinessHelper;

  /// Label for the business type selector.
  ///
  /// In en, this message translates to:
  /// **'Business types'**
  String get editProfileBusinessTypesLabel;

  /// Limit guidance for business types.
  ///
  /// In en, this message translates to:
  /// **'Choose up to 5.'**
  String get editProfileBusinessTypesHelper;

  /// Validation error for too many business types.
  ///
  /// In en, this message translates to:
  /// **'Choose no more than 5 business types.'**
  String get editProfileBusinessTypesLimit;

  /// Label for the business offerings selector.
  ///
  /// In en, this message translates to:
  /// **'Offerings'**
  String get editProfileBusinessOfferingsLabel;

  /// Limit guidance for business offerings.
  ///
  /// In en, this message translates to:
  /// **'Choose up to 10.'**
  String get editProfileBusinessOfferingsHelper;

  /// Validation error for too many offerings.
  ///
  /// In en, this message translates to:
  /// **'Choose no more than 10 offerings.'**
  String get editProfileBusinessOfferingsLimit;

  /// Label for the business tagline field.
  ///
  /// In en, this message translates to:
  /// **'Tagline'**
  String get editProfileBusinessTaglineLabel;

  /// Validation error for the business tagline bounds.
  ///
  /// In en, this message translates to:
  /// **'Tagline must be 100 characters or fewer.'**
  String get editProfileBusinessTaglineTooLong;

  /// Label for the free-text business hours note.
  ///
  /// In en, this message translates to:
  /// **'Hours'**
  String get editProfileBusinessHoursLabel;

  /// Validation error for the business hours bounds.
  ///
  /// In en, this message translates to:
  /// **'Hours must be 300 characters or fewer.'**
  String get editProfileBusinessHoursTooLong;

  /// Label for the display-only service area.
  ///
  /// In en, this message translates to:
  /// **'Service area'**
  String get editProfileBusinessServiceAreaLabel;

  /// Validation error for the business service-area bounds.
  ///
  /// In en, this message translates to:
  /// **'Service area must be 200 characters or fewer.'**
  String get editProfileBusinessServiceAreaTooLong;

  /// Label for the ISO alpha-2 country code field.
  ///
  /// In en, this message translates to:
  /// **'Country code'**
  String get editProfileBusinessCountryLabel;

  /// Validation error for an unsupported business country.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid two-letter country code.'**
  String get editProfileBusinessCountryInvalid;

  /// Label for the optional business locality.
  ///
  /// In en, this message translates to:
  /// **'Town or locality'**
  String get editProfileBusinessLocalityLabel;

  /// Validation error for the business locality bounds.
  ///
  /// In en, this message translates to:
  /// **'Town or locality must be 100 characters or fewer.'**
  String get editProfileBusinessLocalityTooLong;

  /// Label for the approved business primary-action selector.
  ///
  /// In en, this message translates to:
  /// **'Primary action'**
  String get editProfileBusinessActionLabel;

  /// Selector option that removes the business primary action.
  ///
  /// In en, this message translates to:
  /// **'No primary action'**
  String get editProfileBusinessActionNone;

  /// Label for the business primary action HTTPS or mailto destination.
  ///
  /// In en, this message translates to:
  /// **'Action destination'**
  String get editProfileBusinessActionDestinationLabel;

  /// Validation error for a malformed business action destination.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid HTTPS or email destination.'**
  String get editProfileBusinessActionDestinationInvalid;

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

  /// Title for the profile customisation settings page and settings tile.
  ///
  /// In en, this message translates to:
  /// **'Customisation'**
  String get profileCustomisationTitle;

  /// Accessible heading for the live profile customisation preview.
  ///
  /// In en, this message translates to:
  /// **'Preview'**
  String get profileCustomisationPreview;

  /// Heading for the fixed profile colour choices.
  ///
  /// In en, this message translates to:
  /// **'Colour'**
  String get profileCustomisationColour;

  /// Heading for profile picture border thickness choices.
  ///
  /// In en, this message translates to:
  /// **'Profile border'**
  String get profileCustomisationBorder;

  /// Heading for profile background texture choices.
  ///
  /// In en, this message translates to:
  /// **'Profile background'**
  String get profileCustomisationBackground;

  /// Action that saves profile customisation choices.
  ///
  /// In en, this message translates to:
  /// **'Save'**
  String get profileCustomisationSave;

  /// Success feedback shown after saving profile customisation.
  ///
  /// In en, this message translates to:
  /// **'Profile customisation saved'**
  String get profileCustomisationSaved;

  /// Retryable failure feedback shown when profile customisation cannot be saved.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t save your profile customisation.'**
  String get profileCustomisationSaveError;

  /// Error shown when the customisation editor cannot load.
  ///
  /// In en, this message translates to:
  /// **'Couldn\'t load your profile customisation.'**
  String get profileCustomisationLoadError;

  /// Retries loading profile customisation.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get profileCustomisationRetry;

  /// Title of the customisation discard confirmation.
  ///
  /// In en, this message translates to:
  /// **'Discard customisation changes?'**
  String get profileCustomisationDiscardTitle;

  /// Body of the customisation discard confirmation.
  ///
  /// In en, this message translates to:
  /// **'Your customisation changes won\'t be saved.'**
  String get profileCustomisationDiscardMessage;

  /// Label for no profile background texture.
  ///
  /// In en, this message translates to:
  /// **'None'**
  String get profileCustomisationNone;

  /// Label for the cobalt profile colour.
  ///
  /// In en, this message translates to:
  /// **'Cobalt'**
  String get profileCustomisationColourCobalt;

  /// Label for the orchid profile colour.
  ///
  /// In en, this message translates to:
  /// **'Orchid'**
  String get profileCustomisationColourOrchid;

  /// Label for the rose profile colour.
  ///
  /// In en, this message translates to:
  /// **'Rose'**
  String get profileCustomisationColourRose;

  /// Label for the amber profile colour.
  ///
  /// In en, this message translates to:
  /// **'Amber'**
  String get profileCustomisationColourAmber;

  /// Label for the green profile colour stored under the stable lime key.
  ///
  /// In en, this message translates to:
  /// **'Green'**
  String get profileCustomisationColourGreen;

  /// Label for the teal profile colour.
  ///
  /// In en, this message translates to:
  /// **'Teal'**
  String get profileCustomisationColourTeal;

  /// Label for the profile colour that uses the CraftSky theme Ink colour.
  ///
  /// In en, this message translates to:
  /// **'Ink'**
  String get profileCustomisationColourInk;

  /// Label for the thin profile picture border.
  ///
  /// In en, this message translates to:
  /// **'Thin'**
  String get profileCustomisationBorderThin;

  /// Label for the medium profile picture border.
  ///
  /// In en, this message translates to:
  /// **'Medium'**
  String get profileCustomisationBorderMedium;

  /// Label for the thick profile picture border.
  ///
  /// In en, this message translates to:
  /// **'Thick'**
  String get profileCustomisationBorderThick;

  /// Label for the bayerdark profile texture.
  ///
  /// In en, this message translates to:
  /// **'Dither'**
  String get profileCustomisationBackgroundDither;

  /// Label for the cubedark profile texture.
  ///
  /// In en, this message translates to:
  /// **'Grid'**
  String get profileCustomisationBackgroundGrid;

  /// Label for the dotcrossdark profile texture.
  ///
  /// In en, this message translates to:
  /// **'Cross stitch'**
  String get profileCustomisationBackgroundCrossStitch;

  /// Label for the scallopdark profile texture.
  ///
  /// In en, this message translates to:
  /// **'Scallops'**
  String get profileCustomisationBackgroundScallops;

  /// Label for the skewdark profile texture.
  ///
  /// In en, this message translates to:
  /// **'Diagonal weave'**
  String get profileCustomisationBackgroundDiagonalWeave;

  /// Label for the x2 profile texture.
  ///
  /// In en, this message translates to:
  /// **'Crosshatch'**
  String get profileCustomisationBackgroundCrosshatch;

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

  /// No description provided for @instagramMigrationTitle.
  ///
  /// In en, this message translates to:
  /// **'Find people from Instagram'**
  String get instagramMigrationTitle;

  /// No description provided for @instagramMigrationSettingsSubtitle.
  ///
  /// In en, this message translates to:
  /// **'Verify your account or privately import handles.'**
  String get instagramMigrationSettingsSubtitle;

  /// No description provided for @instagramMigrationLoadError.
  ///
  /// In en, this message translates to:
  /// **'Instagram migration data didn\'t load.'**
  String get instagramMigrationLoadError;

  /// No description provided for @instagramMigrationNoActiveAccount.
  ///
  /// In en, this message translates to:
  /// **'Sign in to an account to use Instagram migration.'**
  String get instagramMigrationNoActiveAccount;

  /// No description provided for @instagramVerificationTitle.
  ///
  /// In en, this message translates to:
  /// **'Verify your Instagram account'**
  String get instagramVerificationTitle;

  /// No description provided for @instagramVerificationDescription.
  ///
  /// In en, this message translates to:
  /// **'Send a one-time challenge to CraftSky\'s official Instagram account. You will confirm the username here before it is verified.'**
  String get instagramVerificationDescription;

  /// No description provided for @instagramVerificationUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Instagram verification is unavailable right now.'**
  String get instagramVerificationUnavailable;

  /// No description provided for @instagramVerificationUnavailableImports.
  ///
  /// In en, this message translates to:
  /// **'Imports become available after Instagram verification is configured and your account is verified.'**
  String get instagramVerificationUnavailableImports;

  /// No description provided for @instagramVerificationRequiredForImport.
  ///
  /// In en, this message translates to:
  /// **'Complete verification to import the accounts you follow.'**
  String get instagramVerificationRequiredForImport;

  /// No description provided for @instagramVerificationStart.
  ///
  /// In en, this message translates to:
  /// **'Create verification challenge'**
  String get instagramVerificationStart;

  /// No description provided for @instagramVerificationSendChallenge.
  ///
  /// In en, this message translates to:
  /// **'Send this exact one-time challenge in an Instagram DM:'**
  String get instagramVerificationSendChallenge;

  /// No description provided for @instagramVerificationChallengeLabel.
  ///
  /// In en, this message translates to:
  /// **'Instagram verification challenge'**
  String get instagramVerificationChallengeLabel;

  /// No description provided for @instagramVerificationProcessing.
  ///
  /// In en, this message translates to:
  /// **'Checking your message…'**
  String get instagramVerificationProcessing;

  /// No description provided for @instagramCopyChallenge.
  ///
  /// In en, this message translates to:
  /// **'Copy challenge'**
  String get instagramCopyChallenge;

  /// No description provided for @instagramChallengeCopied.
  ///
  /// In en, this message translates to:
  /// **'Challenge copied'**
  String get instagramChallengeCopied;

  /// No description provided for @instagramOpenDm.
  ///
  /// In en, this message translates to:
  /// **'Open Instagram DM'**
  String get instagramOpenDm;

  /// No description provided for @instagramCancelVerification.
  ///
  /// In en, this message translates to:
  /// **'Cancel verification'**
  String get instagramCancelVerification;

  /// No description provided for @instagramVerificationCandidate.
  ///
  /// In en, this message translates to:
  /// **'Account: @{username}'**
  String instagramVerificationCandidate(String username);

  /// No description provided for @instagramUnknownUsername.
  ///
  /// In en, this message translates to:
  /// **'unknown'**
  String get instagramUnknownUsername;

  /// No description provided for @instagramVerificationCandidateWarning.
  ///
  /// In en, this message translates to:
  /// **'Confirm only if this is your Instagram username.'**
  String get instagramVerificationCandidateWarning;

  /// No description provided for @instagramDiscoverableLabel.
  ///
  /// In en, this message translates to:
  /// **'Let others find me by my Instagram username'**
  String get instagramDiscoverableLabel;

  /// No description provided for @instagramDiscoverableDescription.
  ///
  /// In en, this message translates to:
  /// **'When enabled, eligible CraftSky members who imported your Instagram username may see a private suggestion to follow you.'**
  String get instagramDiscoverableDescription;

  /// No description provided for @instagramDiscoverableAllow.
  ///
  /// In en, this message translates to:
  /// **'Allow discovery'**
  String get instagramDiscoverableAllow;

  /// No description provided for @instagramDiscoverablePrivate.
  ///
  /// In en, this message translates to:
  /// **'Keep private'**
  String get instagramDiscoverablePrivate;

  /// No description provided for @instagramDiscoverablePrivateDescription.
  ///
  /// In en, this message translates to:
  /// **'Your Instagram account remains verified, but CraftSky will not match it with people who imported your username.'**
  String get instagramDiscoverablePrivateDescription;

  /// No description provided for @instagramVerificationConfirm.
  ///
  /// In en, this message translates to:
  /// **'Confirm this account'**
  String get instagramVerificationConfirm;

  /// No description provided for @instagramVerificationConfirmed.
  ///
  /// In en, this message translates to:
  /// **'Instagram account confirmed.'**
  String get instagramVerificationConfirmed;

  /// No description provided for @instagramVerificationExpired.
  ///
  /// In en, this message translates to:
  /// **'This verification challenge expired.'**
  String get instagramVerificationExpired;

  /// No description provided for @instagramVerificationCancelled.
  ///
  /// In en, this message translates to:
  /// **'This verification challenge is no longer active.'**
  String get instagramVerificationCancelled;

  /// No description provided for @instagramVerificationRejected.
  ///
  /// In en, this message translates to:
  /// **'Instagram could not verify this message. Create a new challenge to try again.'**
  String get instagramVerificationRejected;

  /// No description provided for @instagramVerificationProfileUnavailable.
  ///
  /// In en, this message translates to:
  /// **'Instagram profile lookup is temporarily unavailable. Create a new challenge to try again.'**
  String get instagramVerificationProfileUnavailable;

  /// No description provided for @instagramVerificationProfileInvalid.
  ///
  /// In en, this message translates to:
  /// **'Instagram returned an invalid profile result. Create a new challenge to try again.'**
  String get instagramVerificationProfileInvalid;

  /// No description provided for @instagramVerificationMembershipInactive.
  ///
  /// In en, this message translates to:
  /// **'Your CraftSky membership is inactive. Restore membership before trying again.'**
  String get instagramVerificationMembershipInactive;

  /// No description provided for @instagramVerificationConflict.
  ///
  /// In en, this message translates to:
  /// **'This Instagram account cannot be verified automatically. Your existing verified account remains unchanged.'**
  String get instagramVerificationConflict;

  /// No description provided for @instagramActionError.
  ///
  /// In en, this message translates to:
  /// **'That Instagram action didn\'t complete. Try again.'**
  String get instagramActionError;

  /// No description provided for @instagramRetry.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get instagramRetry;

  /// No description provided for @instagramLoadMore.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get instagramLoadMore;

  /// No description provided for @instagramAccountTitle.
  ///
  /// In en, this message translates to:
  /// **'Instagram account'**
  String get instagramAccountTitle;

  /// No description provided for @instagramLinkedAs.
  ///
  /// In en, this message translates to:
  /// **'Verified as @{username}'**
  String instagramLinkedAs(String username);

  /// No description provided for @instagramConflictPending.
  ///
  /// In en, this message translates to:
  /// **'There is a private account conflict to resolve. No ownership was transferred.'**
  String get instagramConflictPending;

  /// No description provided for @instagramReactivateAccount.
  ///
  /// In en, this message translates to:
  /// **'Reactivate Instagram account'**
  String get instagramReactivateAccount;

  /// No description provided for @instagramReactivateAccountDisclosure.
  ///
  /// In en, this message translates to:
  /// **'Reactivation keeps discovery off until you choose to turn it on again.'**
  String get instagramReactivateAccountDisclosure;

  /// No description provided for @instagramRevokeAccount.
  ///
  /// In en, this message translates to:
  /// **'Revoke Instagram verification'**
  String get instagramRevokeAccount;

  /// No description provided for @instagramRevokeAccountConfirmTitle.
  ///
  /// In en, this message translates to:
  /// **'Revoke Instagram verification?'**
  String get instagramRevokeAccountConfirmTitle;

  /// No description provided for @instagramRevokeAccountConfirmMessage.
  ///
  /// In en, this message translates to:
  /// **'This removes your Instagram verification and deletes your imported handles. Existing CraftSky follows will not be affected.'**
  String get instagramRevokeAccountConfirmMessage;

  /// No description provided for @instagramImportTitle.
  ///
  /// In en, this message translates to:
  /// **'Import accounts you follow'**
  String get instagramImportTitle;

  /// No description provided for @instagramImportManual.
  ///
  /// In en, this message translates to:
  /// **'Enter handles'**
  String get instagramImportManual;

  /// No description provided for @instagramImportManualDescription.
  ///
  /// In en, this message translates to:
  /// **'Enter the Instagram handles of accounts you follow, one per line.'**
  String get instagramImportManualDescription;

  /// No description provided for @instagramImportJson.
  ///
  /// In en, this message translates to:
  /// **'Instagram export'**
  String get instagramImportJson;

  /// No description provided for @instagramImportJsonDescription.
  ///
  /// In en, this message translates to:
  /// **'Choose an Instagram export containing Accounts you follow. CraftSky processes it on this device and uploads only those usernames. If you select an all-information ZIP, everything else stays on your device.'**
  String get instagramImportJsonDescription;

  /// No description provided for @instagramImportHandles.
  ///
  /// In en, this message translates to:
  /// **'Instagram handles'**
  String get instagramImportHandles;

  /// No description provided for @instagramImportHandlesHint.
  ///
  /// In en, this message translates to:
  /// **'One handle per line'**
  String get instagramImportHandlesHint;

  /// No description provided for @instagramImportManualAction.
  ///
  /// In en, this message translates to:
  /// **'Import handles'**
  String get instagramImportManualAction;

  /// No description provided for @instagramImportSelectJson.
  ///
  /// In en, this message translates to:
  /// **'Select Instagram export'**
  String get instagramImportSelectJson;

  /// No description provided for @instagramImportFilePickerError.
  ///
  /// In en, this message translates to:
  /// **'The Instagram export couldn\'t be opened on this device.'**
  String get instagramImportFilePickerError;

  /// No description provided for @instagramImportInvalidJson.
  ///
  /// In en, this message translates to:
  /// **'This file is not valid JSON.'**
  String get instagramImportInvalidJson;

  /// No description provided for @instagramImportUnsupportedShape.
  ///
  /// In en, this message translates to:
  /// **'This is not a supported Instagram accounts-followed export. Choose an export containing Accounts you follow.'**
  String get instagramImportUnsupportedShape;

  /// No description provided for @instagramImportUnsupportedFormat.
  ///
  /// In en, this message translates to:
  /// **'This Instagram export uses a format CraftSky can\'t read.'**
  String get instagramImportUnsupportedFormat;

  /// No description provided for @instagramImportInvalidArchive.
  ///
  /// In en, this message translates to:
  /// **'This Instagram ZIP is incomplete or damaged. Download a new export and try again.'**
  String get instagramImportInvalidArchive;

  /// No description provided for @instagramImportArchiveTooLarge.
  ///
  /// In en, this message translates to:
  /// **'This Instagram ZIP contains too many files to process safely.'**
  String get instagramImportArchiveTooLarge;

  /// No description provided for @instagramImportFileTooLarge.
  ///
  /// In en, this message translates to:
  /// **'The accounts-followed data is larger than 20 MiB.'**
  String get instagramImportFileTooLarge;

  /// No description provided for @instagramImportTooManyEntries.
  ///
  /// In en, this message translates to:
  /// **'This import contains more than 10,000 unique handles.'**
  String get instagramImportTooManyEntries;

  /// No description provided for @instagramImportFollowingPreviewCount.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 account you follow ready} other{{count} accounts you follow ready}}'**
  String instagramImportFollowingPreviewCount(int count);

  /// No description provided for @instagramImportIgnoredCount.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 unsupported entry ignored} other{{count} unsupported entries ignored}}'**
  String instagramImportIgnoredCount(int count);

  /// No description provided for @instagramImportDuplicateCount.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 duplicate removed} other{{count} duplicates removed}}'**
  String instagramImportDuplicateCount(int count);

  /// No description provided for @instagramImportUploadSuccess.
  ///
  /// In en, this message translates to:
  /// **'Instagram import created'**
  String get instagramImportUploadSuccess;

  /// No description provided for @instagramImportUploadError.
  ///
  /// In en, this message translates to:
  /// **'Instagram import wasn\'t created. Try again.'**
  String get instagramImportUploadError;

  /// No description provided for @instagramImportsTitle.
  ///
  /// In en, this message translates to:
  /// **'Your imports'**
  String get instagramImportsTitle;

  /// No description provided for @instagramImportsLoadError.
  ///
  /// In en, this message translates to:
  /// **'Your Instagram imports didn\'t load.'**
  String get instagramImportsLoadError;

  /// No description provided for @instagramImportsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No Instagram imports yet.'**
  String get instagramImportsEmpty;

  /// No description provided for @instagramImportManualSource.
  ///
  /// In en, this message translates to:
  /// **'Manual handles'**
  String get instagramImportManualSource;

  /// No description provided for @instagramImportJsonSource.
  ///
  /// In en, this message translates to:
  /// **'Instagram export'**
  String get instagramImportJsonSource;

  /// No description provided for @instagramImportUnknownSource.
  ///
  /// In en, this message translates to:
  /// **'Instagram import'**
  String get instagramImportUnknownSource;

  /// No description provided for @instagramImportCounts.
  ///
  /// In en, this message translates to:
  /// **'{followingCount, plural, =1{1 account imported} other{{followingCount} accounts imported}}'**
  String instagramImportCounts(int followingCount);

  /// No description provided for @instagramImportReactivationDisclosure.
  ///
  /// In en, this message translates to:
  /// **'This import paused when your CraftSky membership changed. Reactivate it to resume matching.'**
  String get instagramImportReactivationDisclosure;

  /// No description provided for @instagramImportReactivate.
  ///
  /// In en, this message translates to:
  /// **'Reactivate import'**
  String get instagramImportReactivate;

  /// No description provided for @instagramImportDelete.
  ///
  /// In en, this message translates to:
  /// **'Delete import'**
  String get instagramImportDelete;

  /// No description provided for @instagramImportSuggestionDisclosure.
  ///
  /// In en, this message translates to:
  /// **'Importing creates private suggestions only. You choose whether to follow each account.'**
  String get instagramImportSuggestionDisclosure;

  /// No description provided for @instagramSuggestionsTitle.
  ///
  /// In en, this message translates to:
  /// **'Possible CraftSky accounts'**
  String get instagramSuggestionsTitle;

  /// No description provided for @instagramSuggestionsDescription.
  ///
  /// In en, this message translates to:
  /// **'Your imports find possible CraftSky accounts privately. Nobody is followed until you choose Follow.'**
  String get instagramSuggestionsDescription;

  /// No description provided for @instagramSuggestionsLoadError.
  ///
  /// In en, this message translates to:
  /// **'Possible CraftSky accounts didn\'t load.'**
  String get instagramSuggestionsLoadError;

  /// No description provided for @instagramSuggestionsEmpty.
  ///
  /// In en, this message translates to:
  /// **'No possible CraftSky accounts yet.'**
  String get instagramSuggestionsEmpty;

  /// No description provided for @instagramSuggestionFollow.
  ///
  /// In en, this message translates to:
  /// **'Follow'**
  String get instagramSuggestionFollow;

  /// No description provided for @instagramSuggestionDismiss.
  ///
  /// In en, this message translates to:
  /// **'Dismiss'**
  String get instagramSuggestionDismiss;

  /// No description provided for @instagramSuggestionsActionError.
  ///
  /// In en, this message translates to:
  /// **'That suggestion action didn\'t complete. Try again.'**
  String get instagramSuggestionsActionError;

  /// Tooltip and action label for saving a post.
  ///
  /// In en, this message translates to:
  /// **'Save post'**
  String get savedPostSaveAction;

  /// Tooltip for removing a post from the viewer's private saved collection.
  ///
  /// In en, this message translates to:
  /// **'Remove from saved posts'**
  String get savedPostUnsaveAction;

  /// Safe retryable feedback after removing a saved post fails.
  ///
  /// In en, this message translates to:
  /// **'This post couldn\'t be removed. Try again.'**
  String get savedPostUnsaveError;

  /// Title for the chooser used to move an existing saved post.
  ///
  /// In en, this message translates to:
  /// **'Move saved post'**
  String get savedPostMoveTitle;

  /// Action opening the folder chooser for an existing saved post.
  ///
  /// In en, this message translates to:
  /// **'Move'**
  String get savedPostMoveAction;

  /// Row action removing a post from saved posts.
  ///
  /// In en, this message translates to:
  /// **'Unsave'**
  String get savedPostRowUnsaveAction;

  /// Chooser option that leaves a saved post outside a folder.
  ///
  /// In en, this message translates to:
  /// **'No folder'**
  String get savedPostNoFolder;

  /// Label for the saved-post folder selector.
  ///
  /// In en, this message translates to:
  /// **'Folder'**
  String get savedPostFolderSelectionLabel;

  /// Safe inline error after a save or move request fails.
  ///
  /// In en, this message translates to:
  /// **'That change couldn\'t be saved. Try again.'**
  String get savedPostConfirmError;

  /// Safe inline error when saved-post folders cannot load.
  ///
  /// In en, this message translates to:
  /// **'Folders couldn\'t load.'**
  String get savedPostFoldersLoadError;

  /// Action loading the next page of saved-post folders.
  ///
  /// In en, this message translates to:
  /// **'Load more folders'**
  String get savedPostLoadMoreFolders;

  /// Action opening inline saved-folder creation.
  ///
  /// In en, this message translates to:
  /// **'New folder'**
  String get savedPostNewFolder;

  /// Input label for a saved-post folder name.
  ///
  /// In en, this message translates to:
  /// **'Folder name'**
  String get savedPostFolderNameHint;

  /// Action submitting a new saved-post folder.
  ///
  /// In en, this message translates to:
  /// **'Create folder'**
  String get savedPostCreateFolderAction;

  /// Safe inline error after saved-folder creation fails.
  ///
  /// In en, this message translates to:
  /// **'That folder couldn\'t be created. Try again.'**
  String get savedPostCreateFolderError;

  /// Title for the viewer's private saved-post collection.
  ///
  /// In en, this message translates to:
  /// **'Saved posts'**
  String get savedPostsTitle;

  /// Heading above saved-post folders.
  ///
  /// In en, this message translates to:
  /// **'Folders'**
  String get savedPostsFoldersHeading;

  /// Heading above saved posts that have no folder.
  ///
  /// In en, this message translates to:
  /// **'Unfiled'**
  String get savedPostsUnfiledHeading;

  /// Empty state when the viewer has no saved posts or folders.
  ///
  /// In en, this message translates to:
  /// **'Nothing saved yet'**
  String get savedPostsEmpty;

  /// Sort label showing the oldest saves first.
  ///
  /// In en, this message translates to:
  /// **'Oldest'**
  String get savedPostsSortOldest;

  /// Helper text for newest-first saved-post ordering.
  ///
  /// In en, this message translates to:
  /// **'Most recently saved first'**
  String get savedPostsSortNewestDescription;

  /// Helper text for oldest-first saved-post ordering.
  ///
  /// In en, this message translates to:
  /// **'Earliest saved first'**
  String get savedPostsSortOldestDescription;

  /// Safe initial error for the saved-post collection.
  ///
  /// In en, this message translates to:
  /// **'Saved posts couldn\'t load.'**
  String get savedPostsLoadError;

  /// Action loading the next page of saved posts.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get savedPostsLoadMore;

  /// Tooltip opening saved-folder actions.
  ///
  /// In en, this message translates to:
  /// **'Folder actions'**
  String get savedPostFolderActions;

  /// Tooltip opening move and unsave actions for one saved post.
  ///
  /// In en, this message translates to:
  /// **'Saved post actions'**
  String get savedPostRowActions;

  /// Action and dialog title for renaming a saved folder.
  ///
  /// In en, this message translates to:
  /// **'Rename folder'**
  String get savedPostRenameFolder;

  /// Destructive action and dialog title for deleting a saved folder.
  ///
  /// In en, this message translates to:
  /// **'Delete folder'**
  String get savedPostDeleteFolder;

  /// Prompt explaining the two scopes of saved-folder deletion.
  ///
  /// In en, this message translates to:
  /// **'What should happen to the posts in this folder?'**
  String get savedPostDeleteFolderBody;

  /// Deletes a folder while moving its saves to Unfiled.
  ///
  /// In en, this message translates to:
  /// **'Keep saved posts'**
  String get savedPostKeepSaves;

  /// Deletes a folder and all private saves assigned to it.
  ///
  /// In en, this message translates to:
  /// **'Delete saved posts'**
  String get savedPostDeleteSaves;

  /// Settings destination for App, Primary, and Content language preferences.
  ///
  /// In en, this message translates to:
  /// **'Languages'**
  String get settingsLanguages;

  /// Title of the Languages settings page.
  ///
  /// In en, this message translates to:
  /// **'Languages'**
  String get languagesTitle;

  /// Heading for the language used by the CraftSky interface.
  ///
  /// In en, this message translates to:
  /// **'App language'**
  String get appLanguageTitle;

  /// Explanation of the App language setting.
  ///
  /// In en, this message translates to:
  /// **'Select which language to use for the app\'s user interface.'**
  String get appLanguageDescription;

  /// English option in the App language setting.
  ///
  /// In en, this message translates to:
  /// **'English'**
  String get appLanguageEnglish;

  /// Explains that English is currently the only App language.
  ///
  /// In en, this message translates to:
  /// **'More app languages are coming.'**
  String get appLanguageMoreComing;

  /// Heading for the default language used by new posts.
  ///
  /// In en, this message translates to:
  /// **'Primary language'**
  String get primaryLanguageTitle;

  /// Explanation of the Primary language setting.
  ///
  /// In en, this message translates to:
  /// **'Select the default language used when you create a post.'**
  String get primaryLanguageDescription;

  /// Heading for languages included in browsing and discovery.
  ///
  /// In en, this message translates to:
  /// **'Content languages'**
  String get contentLanguagesTitle;

  /// Explanation of Content languages, including empty selection behavior.
  ///
  /// In en, this message translates to:
  /// **'Select which languages you want posts in your feeds and discovery results to include. If none are selected, all languages will be shown.'**
  String get contentLanguagesDescription;

  /// Hint for finding a language in the full catalogue.
  ///
  /// In en, this message translates to:
  /// **'Search languages'**
  String get languageSearchHint;

  /// Action opening the Content language selector.
  ///
  /// In en, this message translates to:
  /// **'Add more languages…'**
  String get languageAddMore;

  /// Action dismissing a language selector without saving.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get languageCancel;

  /// Action confirming selected Content languages.
  ///
  /// In en, this message translates to:
  /// **'Done'**
  String get languageDone;

  /// Safe feedback after a private language preference replacement fails.
  ///
  /// In en, this message translates to:
  /// **'That change could not be saved. Try again.'**
  String get languageSaveError;

  /// Accessibility label for the post language selector.
  ///
  /// In en, this message translates to:
  /// **'Post languages'**
  String get postLanguagesSemantics;

  /// Action adding another language to a post.
  ///
  /// In en, this message translates to:
  /// **'Add language'**
  String get postLanguageAdd;

  /// Visible and accessible explanation when a post has the maximum three languages.
  ///
  /// In en, this message translates to:
  /// **'Up to three languages'**
  String get postLanguageLimit;

  /// Title of the searchable post-language dialog.
  ///
  /// In en, this message translates to:
  /// **'Add post language'**
  String get postLanguageDialogTitle;

  /// Action retrying a composer's Primary-language preference load.
  ///
  /// In en, this message translates to:
  /// **'Retry loading languages'**
  String get postLanguageRetryLoading;

  /// Title for scheduled-post settings and management.
  ///
  /// In en, this message translates to:
  /// **'Scheduled posts'**
  String get scheduledPostsTitle;

  /// Empty state on the scheduled-post management page.
  ///
  /// In en, this message translates to:
  /// **'No scheduled posts'**
  String get scheduledPostsEmpty;

  /// Title for scheduled-post deletion confirmation.
  ///
  /// In en, this message translates to:
  /// **'Delete scheduled post?'**
  String get scheduledPostsDeleteTitle;

  /// Explanation shown before deleting a scheduled post.
  ///
  /// In en, this message translates to:
  /// **'This removes the unpublished post and its private media.'**
  String get scheduledPostsDeleteMessage;

  /// Confirmation action for deleting a scheduled post.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get scheduledPostsDeleteAction;

  /// Safe failure feedback when an owner cannot delete a scheduled post.
  ///
  /// In en, this message translates to:
  /// **'Could not delete the scheduled post. Try again.'**
  String get scheduledPostDeleteError;

  /// Project-post type label in scheduled-post management.
  ///
  /// In en, this message translates to:
  /// **'Project'**
  String get scheduledPostsKindProject;

  /// Standard-post type label in scheduled-post management.
  ///
  /// In en, this message translates to:
  /// **'Standard'**
  String get scheduledPostsKindStandard;

  /// Scheduled-post type and local publication date and time.
  ///
  /// In en, this message translates to:
  /// **'{kind} · {date}, {time}'**
  String scheduledPostsRowDateTime(String kind, String date, String time);

  /// Status label for a post waiting for publication.
  ///
  /// In en, this message translates to:
  /// **'Scheduled'**
  String get scheduledPostsStatusScheduled;

  /// Status label for a post currently publishing.
  ///
  /// In en, this message translates to:
  /// **'Publishing'**
  String get scheduledPostsStatusPublishing;

  /// Status label for a scheduled post that will retry.
  ///
  /// In en, this message translates to:
  /// **'Retrying'**
  String get scheduledPostsStatusRetrying;

  /// Status label for a scheduled post requiring user action.
  ///
  /// In en, this message translates to:
  /// **'Needs attention'**
  String get scheduledPostsStatusNeedsAttention;

  /// Explanation shown while a scheduled post is locked for publication.
  ///
  /// In en, this message translates to:
  /// **'Editing is unavailable while publishing'**
  String get scheduledPostsPublishingLocked;

  /// Accessibility label for the publishing lock icon.
  ///
  /// In en, this message translates to:
  /// **'Publishing lock'**
  String get scheduledPostsPublishingLockSemantics;

  /// Tooltip for editing a scheduled post.
  ///
  /// In en, this message translates to:
  /// **'Edit scheduled post'**
  String get scheduledPostsEditTooltip;

  /// Tooltip for deleting a scheduled post.
  ///
  /// In en, this message translates to:
  /// **'Delete scheduled post'**
  String get scheduledPostsDeleteTooltip;

  /// Accessibility label for an authenticated scheduled-post thumbnail.
  ///
  /// In en, this message translates to:
  /// **'Scheduled post image'**
  String get scheduledPostsThumbnailSemantics;

  /// Deletion deadline shown for a scheduled post that needs attention.
  ///
  /// In en, this message translates to:
  /// **'Deleted on {date}'**
  String scheduledPostsDeletedOn(String date);

  /// Safe error shown when the scheduled-post list cannot load.
  ///
  /// In en, this message translates to:
  /// **'Could not load scheduled posts'**
  String get scheduledPostsLoadError;

  /// Action retrying a scheduled-post list load.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get scheduledPostsRetryAction;

  /// Label for selecting when to publish.
  ///
  /// In en, this message translates to:
  /// **'When'**
  String get scheduledPostWhenTitle;

  /// Default option to publish immediately.
  ///
  /// In en, this message translates to:
  /// **'Now'**
  String get scheduledPostNow;

  /// Option to choose a future publication time.
  ///
  /// In en, this message translates to:
  /// **'Schedule for later'**
  String get scheduledPostLater;

  /// Error shown when a selected schedule time falls outside the supported window.
  ///
  /// In en, this message translates to:
  /// **'Choose a whole-minute time from 5 minutes through 28 days from now'**
  String get scheduledPostTimeRangeError;

  /// Composer action that saves a post for future publication.
  ///
  /// In en, this message translates to:
  /// **'Schedule'**
  String get scheduledPostAction;

  /// Visible and announced private-media staging progress.
  ///
  /// In en, this message translates to:
  /// **'Preparing image {current} of {total}'**
  String scheduledPostStagingProgress(int current, int total);

  /// Visible and announced progress while a schedule is saved.
  ///
  /// In en, this message translates to:
  /// **'Saving scheduled post'**
  String get scheduledPostCreating;

  /// Action opening scheduled-post management.
  ///
  /// In en, this message translates to:
  /// **'Manage scheduled posts'**
  String get scheduledPostManageAction;

  /// Warning shown when all three scheduled-post slots are occupied.
  ///
  /// In en, this message translates to:
  /// **'You can\'t schedule another post because you already have 3 scheduled posts.'**
  String get scheduledPostCapacityWarning;

  /// Selected publication time in the device's local timezone.
  ///
  /// In en, this message translates to:
  /// **'{date} at {time} ({zone}, UTC{offset})'**
  String scheduledPostLocalTime(
    String date,
    String time,
    String zone,
    String offset,
  );

  /// Original missed publication time shown while Post now is selected for a Needs attention item.
  ///
  /// In en, this message translates to:
  /// **'Missed schedule: {time}'**
  String scheduledPostMissedTime(String time);

  /// Success feedback after scheduling a standard post.
  ///
  /// In en, this message translates to:
  /// **'Post scheduled'**
  String get scheduledPostSaved;

  /// Safe failure feedback when a standard post cannot be scheduled.
  ///
  /// In en, this message translates to:
  /// **'Could not schedule post. Your draft is still here.'**
  String get scheduledPostSaveError;

  /// Safe failure feedback when publishing an edited scheduled standard post now fails.
  ///
  /// In en, this message translates to:
  /// **'Could not post now. Your draft is still here.'**
  String get scheduledPostNowError;

  /// Success feedback after scheduling a project post.
  ///
  /// In en, this message translates to:
  /// **'Project scheduled'**
  String get scheduledProjectSaved;

  /// Safe failure feedback when a project post cannot be scheduled.
  ///
  /// In en, this message translates to:
  /// **'Could not schedule project. Your draft is still here.'**
  String get scheduledProjectSaveError;

  /// Safe failure feedback when publishing an edited scheduled project now fails.
  ///
  /// In en, this message translates to:
  /// **'Could not post now. Your project is still here.'**
  String get scheduledProjectNowError;

  /// Title for local draft settings and management.
  ///
  /// In en, this message translates to:
  /// **'Drafts'**
  String get draftsTitle;

  /// Empty state on the local draft management page.
  ///
  /// In en, this message translates to:
  /// **'No drafts'**
  String get draftsEmpty;

  /// Title for local draft deletion confirmation.
  ///
  /// In en, this message translates to:
  /// **'Delete draft?'**
  String get draftsDeleteTitle;

  /// Explanation shown before deleting a local draft.
  ///
  /// In en, this message translates to:
  /// **'This removes the draft and its saved images from this device.'**
  String get draftsDeleteMessage;

  /// Confirmation action for deleting a local draft.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get draftsDeleteAction;

  /// Project-post type label in local draft management.
  ///
  /// In en, this message translates to:
  /// **'Project'**
  String get draftsKindProject;

  /// Standard-post type label in local draft management.
  ///
  /// In en, this message translates to:
  /// **'Standard'**
  String get draftsKindStandard;

  /// Kind and last-saved local date/time shown for a draft.
  ///
  /// In en, this message translates to:
  /// **'{kind} · {date}, {time}'**
  String draftsRowDateTime(String kind, String date, String time);

  /// Safe row label for a damaged or unsupported local draft.
  ///
  /// In en, this message translates to:
  /// **'Draft unavailable'**
  String get draftsUnavailable;

  /// Preview label when a valid draft has no text or title.
  ///
  /// In en, this message translates to:
  /// **'Untitled draft'**
  String get draftsUntitled;

  /// Placeholder label for missing or damaged draft media.
  ///
  /// In en, this message translates to:
  /// **'Image unavailable'**
  String get draftsImageUnavailable;

  /// Safe error shown when local drafts cannot load.
  ///
  /// In en, this message translates to:
  /// **'Could not load drafts'**
  String get draftsLoadError;

  /// Action retrying a local draft list load.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get draftsRetryAction;

  /// Tooltip for editing a local draft.
  ///
  /// In en, this message translates to:
  /// **'Edit draft'**
  String get draftsEditTooltip;

  /// Tooltip for deleting a local draft.
  ///
  /// In en, this message translates to:
  /// **'Delete draft'**
  String get draftsDeleteTooltip;

  /// Accessibility label for a local draft thumbnail.
  ///
  /// In en, this message translates to:
  /// **'Draft image'**
  String get draftsThumbnailSemantics;

  /// Blocking immediate-submission status.
  ///
  /// In en, this message translates to:
  /// **'Publishing your post…'**
  String get submissionPublishingPost;

  /// Blocking scheduled-submission status.
  ///
  /// In en, this message translates to:
  /// **'Scheduling your post…'**
  String get submissionSchedulingPost;

  /// Action saving a new local post draft.
  ///
  /// In en, this message translates to:
  /// **'Save draft'**
  String get draftSaveAction;

  /// Action overwriting an existing local post draft.
  ///
  /// In en, this message translates to:
  /// **'Save changes'**
  String get draftSaveChangesAction;

  /// Success feedback after saving a local draft.
  ///
  /// In en, this message translates to:
  /// **'Draft saved'**
  String get draftSavedMessage;

  /// Safe retryable local draft save error.
  ///
  /// In en, this message translates to:
  /// **'Could not save draft'**
  String get draftSaveError;

  /// Title shown when closing a dirty draft-eligible composer.
  ///
  /// In en, this message translates to:
  /// **'Save your draft?'**
  String get draftCloseTitle;

  /// Explanation shown when closing a draft-eligible composer.
  ///
  /// In en, this message translates to:
  /// **'You can save this work on this device before closing.'**
  String get draftCloseMessage;

  /// Action returning to the composer.
  ///
  /// In en, this message translates to:
  /// **'Keep editing'**
  String get draftKeepEditingAction;

  /// Action discarding a never-saved composer.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get draftDiscardAction;

  /// Action closing without overwriting an existing draft.
  ///
  /// In en, this message translates to:
  /// **'Discard changes'**
  String get draftDiscardChangesAction;

  /// Safe warning after remote success and local draft cleanup failure.
  ///
  /// In en, this message translates to:
  /// **'Your post was submitted, but the local draft could not be removed. You can delete it from Drafts.'**
  String get draftCleanupError;

  /// Action replacing unavailable media in an editable local draft.
  ///
  /// In en, this message translates to:
  /// **'Replace image'**
  String get draftsReplaceImageAction;

  /// Settings page title.
  ///
  /// In en, this message translates to:
  /// **'Settings'**
  String get settingsTitle;

  /// Settings row and page title for choosing the app theme.
  ///
  /// In en, this message translates to:
  /// **'Appearance'**
  String get appearanceTitle;

  /// Theme choice that follows the device light or dark setting.
  ///
  /// In en, this message translates to:
  /// **'Use device setting'**
  String get appearanceUseDeviceSetting;

  /// Theme choice that always uses light mode.
  ///
  /// In en, this message translates to:
  /// **'Light'**
  String get appearanceLight;

  /// Theme choice that always uses dark mode.
  ///
  /// In en, this message translates to:
  /// **'Dark'**
  String get appearanceDark;

  /// Action opening the retained-account switcher.
  ///
  /// In en, this message translates to:
  /// **'Switch account'**
  String get settingsSwitchAccount;

  /// Settings section containing app preferences.
  ///
  /// In en, this message translates to:
  /// **'Preferences'**
  String get settingsSectionPreferences;

  /// Settings section containing relationship lists.
  ///
  /// In en, this message translates to:
  /// **'Connections'**
  String get settingsSectionConnections;

  /// Settings section containing account discovery.
  ///
  /// In en, this message translates to:
  /// **'Discovery'**
  String get settingsSectionDiscovery;

  /// Settings section containing Account and About.
  ///
  /// In en, this message translates to:
  /// **'General'**
  String get settingsSectionGeneral;

  /// Settings section containing business owner tools.
  ///
  /// In en, this message translates to:
  /// **'Business'**
  String get settingsSectionBusiness;

  /// Settings row opening business event management.
  ///
  /// In en, this message translates to:
  /// **'Events'**
  String get settingsBusinessEvents;

  /// Settings row opening featured product management.
  ///
  /// In en, this message translates to:
  /// **'Products'**
  String get settingsBusinessProducts;

  /// Settings row opening notification settings.
  ///
  /// In en, this message translates to:
  /// **'Notifications'**
  String get settingsNotifications;

  /// Settings row and page title for follower growth.
  ///
  /// In en, this message translates to:
  /// **'Growth'**
  String get settingsGrowth;

  /// Follower growth metric heading.
  ///
  /// In en, this message translates to:
  /// **'Followers'**
  String get growthMetricLabel;

  /// Follower growth chart section heading.
  ///
  /// In en, this message translates to:
  /// **'Trend'**
  String get growthTrendLabel;

  /// Explains the follower metric scope.
  ///
  /// In en, this message translates to:
  /// **'Craftsky followers.'**
  String get growthScopeCopy;

  /// Explains follower metric freshness and date timezone.
  ///
  /// In en, this message translates to:
  /// **'Updated daily. Dates are UTC.'**
  String get growthFreshnessCopy;

  /// Seven-day follower growth period.
  ///
  /// In en, this message translates to:
  /// **'7 days'**
  String get growthPeriodSevenDays;

  /// Thirty-day follower growth period.
  ///
  /// In en, this message translates to:
  /// **'30 days'**
  String get growthPeriodThirtyDays;

  /// One-year follower growth period.
  ///
  /// In en, this message translates to:
  /// **'1 year'**
  String get growthPeriodOneYear;

  /// Latest persisted follower count.
  ///
  /// In en, this message translates to:
  /// **'{count} followers'**
  String growthLatestCount(String count);

  /// Positive follower change.
  ///
  /// In en, this message translates to:
  /// **'Up {count}'**
  String growthChangeUp(String count);

  /// Negative follower change.
  ///
  /// In en, this message translates to:
  /// **'Down {count}'**
  String growthChangeDown(String count);

  /// Flat follower change.
  ///
  /// In en, this message translates to:
  /// **'No change'**
  String get growthNoChange;

  /// Follower change cannot yet be calculated.
  ///
  /// In en, this message translates to:
  /// **'Not enough history'**
  String get growthInsufficientHistory;

  /// Date of the latest persisted follower snapshot.
  ///
  /// In en, this message translates to:
  /// **'Latest snapshot: {date}'**
  String growthLatestSnapshot(String date);

  /// No follower snapshots exist for the owner.
  ///
  /// In en, this message translates to:
  /// **'No follower history yet'**
  String get growthNoHistory;

  /// Older follower history exists but the selected period is empty.
  ///
  /// In en, this message translates to:
  /// **'No observations in this period'**
  String get growthNoObservationsInPeriod;

  /// Explains when trustworthy follower history begins.
  ///
  /// In en, this message translates to:
  /// **'History available since {date}'**
  String growthHistoryAvailableSince(String date);

  /// Date range included in the follower growth chart summary.
  ///
  /// In en, this message translates to:
  /// **'From {start} to {end}'**
  String growthChartRange(String start, String end);

  /// Explains dates with no persisted follower observation.
  ///
  /// In en, this message translates to:
  /// **'{count, plural, =1{1 day has no observation} other{{count} days have no observation}}'**
  String growthMissingDays(int count);

  /// Safe follower growth loading error.
  ///
  /// In en, this message translates to:
  /// **'Could not load follower growth.'**
  String get growthLoadError;

  /// Settings row opening followers.
  ///
  /// In en, this message translates to:
  /// **'Followers'**
  String get settingsFollowers;

  /// Settings row opening followed accounts.
  ///
  /// In en, this message translates to:
  /// **'Following'**
  String get settingsFollowing;

  /// Title shown when the viewer's follower collection is empty.
  ///
  /// In en, this message translates to:
  /// **'No one follows you yet'**
  String get followersEmptyTitle;

  /// Explanation shown when the viewer's follower collection is empty.
  ///
  /// In en, this message translates to:
  /// **'When someone follows you, they will appear here.'**
  String get followersEmptySubtitle;

  /// Title shown when the viewer's following collection is empty.
  ///
  /// In en, this message translates to:
  /// **'You are not following anyone'**
  String get followingEmptyTitle;

  /// Explanation shown when the viewer's following collection is empty.
  ///
  /// In en, this message translates to:
  /// **'Accounts you follow will appear here.'**
  String get followingEmptySubtitle;

  /// Title shown when a mutual-follower sheet opens after its summary became stale.
  ///
  /// In en, this message translates to:
  /// **'No mutual followers'**
  String get mutualFollowersEmptyTitle;

  /// Explanation for a stale-empty mutual-follower list.
  ///
  /// In en, this message translates to:
  /// **'This list may have changed since the profile loaded.'**
  String get mutualFollowersEmptySubtitle;

  /// Settings row opening account settings.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get settingsAccount;

  /// Settings row opening About.
  ///
  /// In en, this message translates to:
  /// **'About'**
  String get settingsAbout;

  /// External Terms link on About.
  ///
  /// In en, this message translates to:
  /// **'Terms'**
  String get settingsTerms;

  /// External Privacy policy link on About.
  ///
  /// In en, this message translates to:
  /// **'Privacy policy'**
  String get settingsPrivacyPolicy;

  /// Immediate action clearing downloaded image caches.
  ///
  /// In en, this message translates to:
  /// **'Clear image cache'**
  String get settingsClearImageCache;

  /// Success feedback after clearing image caches.
  ///
  /// In en, this message translates to:
  /// **'Image cache cleared'**
  String get settingsImageCacheCleared;

  /// Read-only app version label on About.
  ///
  /// In en, this message translates to:
  /// **'Version'**
  String get settingsVersion;

  /// Immediate action signing out the active account.
  ///
  /// In en, this message translates to:
  /// **'Sign out'**
  String get settingsSignOut;

  /// Account settings page title.
  ///
  /// In en, this message translates to:
  /// **'Account'**
  String get accountTitle;

  /// Heading for the regular or business account type selector.
  ///
  /// In en, this message translates to:
  /// **'Account type'**
  String get accountTypeTitle;

  /// Regular account type selector option.
  ///
  /// In en, this message translates to:
  /// **'Regular'**
  String get accountTypeRegular;

  /// Business account type selector option.
  ///
  /// In en, this message translates to:
  /// **'Business'**
  String get accountTypeBusiness;

  /// First permanent CraftSky deletion dialog title.
  ///
  /// In en, this message translates to:
  /// **'Delete CraftSky account?'**
  String get deleteAccountTitle;

  /// Destructive account deletion action.
  ///
  /// In en, this message translates to:
  /// **'Delete account'**
  String get deleteAccountAction;

  /// Action continuing to fresh reauthentication.
  ///
  /// In en, this message translates to:
  /// **'Continue'**
  String get deleteAccountContinue;

  /// Typed-handle confirmation dialog title.
  ///
  /// In en, this message translates to:
  /// **'Confirm account deletion'**
  String get deleteAccountConfirmTitle;

  /// Label for exact-handle deletion confirmation input.
  ///
  /// In en, this message translates to:
  /// **'Type your handle'**
  String get deleteAccountTypeHandleLabel;

  /// Prompt for exact-handle deletion confirmation.
  ///
  /// In en, this message translates to:
  /// **'Type {handle} exactly to permanently delete this CraftSky account.'**
  String deleteAccountConfirmationPrompt(String handle);

  /// Coarse sign-in outcome while durable account deletion is active.
  ///
  /// In en, this message translates to:
  /// **'Your CraftSky account deletion is already in progress. You cannot sign in again until it has finished.'**
  String get accountDeletionAlreadyInProgress;

  /// Permanent CraftSky deletion boundary shown before typed-handle confirmation.
  ///
  /// In en, this message translates to:
  /// **'Deleting {handle} permanently removes all your CraftSky data from your PDS and all private data held by CraftSky. It won’t delete your PDS, DID, or wider AT Protocol account.\n\nTo continue, you’ll need to authenticate with your PDS again.'**
  String deleteAccountBoundary(String handle);

  /// Accessible label while the composer fetches a link preview.
  ///
  /// In en, this message translates to:
  /// **'Loading link preview'**
  String get linkPreviewLoading;

  /// Tooltip and semantics label for the previous preview action.
  ///
  /// In en, this message translates to:
  /// **'Previous link preview'**
  String get linkPreviewPrevious;

  /// Tooltip and semantics label for the next preview action.
  ///
  /// In en, this message translates to:
  /// **'Next link preview'**
  String get linkPreviewNext;

  /// Tooltip and semantics label for dismissing composer previews.
  ///
  /// In en, this message translates to:
  /// **'Dismiss link previews'**
  String get linkPreviewDismiss;

  /// Snackbar text after previews are dismissed for the composer session.
  ///
  /// In en, this message translates to:
  /// **'Link previews hidden'**
  String get linkPreviewHidden;

  /// Snackbar action restoring dismissed previews.
  ///
  /// In en, this message translates to:
  /// **'Undo'**
  String get linkPreviewUndo;

  /// Position label for the composer preview carousel.
  ///
  /// In en, this message translates to:
  /// **'Link preview {current} of {total}'**
  String linkPreviewPosition(int current, int total);

  /// Accessible action label for an external link card.
  ///
  /// In en, this message translates to:
  /// **'Open link to {host}'**
  String externalCardOpen(String host);

  /// Accessible label for an external card thumbnail.
  ///
  /// In en, this message translates to:
  /// **'Preview image for {title}'**
  String externalCardThumbnail(String title);

  /// Accessible action label for a playable YouTube external card.
  ///
  /// In en, this message translates to:
  /// **'Play YouTube video: {title}'**
  String youtubePlayVideo(String title);

  /// Accessible label for an active YouTube video player.
  ///
  /// In en, this message translates to:
  /// **'YouTube video player: {title}'**
  String youtubeVideoPlayer(String title);

  /// Title of the privacy disclosure shown before loading YouTube.
  ///
  /// In en, this message translates to:
  /// **'Play video from YouTube?'**
  String get youtubeConsentTitle;

  /// Privacy disclosure shown before loading the third-party YouTube player.
  ///
  /// In en, this message translates to:
  /// **'Playing this video connects to YouTube. YouTube may receive your IP address and device information.'**
  String get youtubeConsentMessage;

  /// Button that permits loading one YouTube player without remembering consent.
  ///
  /// In en, this message translates to:
  /// **'Allow once'**
  String get youtubeAllowOnce;

  /// Button that permits YouTube players and remembers the preference on this device.
  ///
  /// In en, this message translates to:
  /// **'Always allow YouTube'**
  String get youtubeAlwaysAllow;

  /// Button that opens the original YouTube URL outside CraftSky.
  ///
  /// In en, this message translates to:
  /// **'Open in YouTube'**
  String get youtubeOpenExternally;

  /// Tooltip for opening an inline YouTube video in a full-screen route.
  ///
  /// In en, this message translates to:
  /// **'Enter full screen'**
  String get youtubeEnterFullscreen;

  /// Stable fallback shown when the YouTube iframe reports a playback error.
  ///
  /// In en, this message translates to:
  /// **'This video can’t be played here. It may be private, unavailable, or restricted from embedded playback.'**
  String get youtubePlaybackUnavailable;

  /// No description provided for @onboardingTitle.
  ///
  /// In en, this message translates to:
  /// **'Set up your CraftSky profile'**
  String get onboardingTitle;

  /// No description provided for @onboardingSkip.
  ///
  /// In en, this message translates to:
  /// **'Skip'**
  String get onboardingSkip;

  /// No description provided for @onboardingNext.
  ///
  /// In en, this message translates to:
  /// **'Next'**
  String get onboardingNext;

  /// No description provided for @onboardingSaveNext.
  ///
  /// In en, this message translates to:
  /// **'Save & next'**
  String get onboardingSaveNext;

  /// No description provided for @onboardingFinish.
  ///
  /// In en, this message translates to:
  /// **'Finish'**
  String get onboardingFinish;

  /// No description provided for @onboardingStepProgress.
  ///
  /// In en, this message translates to:
  /// **'Step {current} of {total}'**
  String onboardingStepProgress(int current, int total);

  /// No description provided for @onboardingProfileTitle.
  ///
  /// In en, this message translates to:
  /// **'Make it yours'**
  String get onboardingProfileTitle;

  /// No description provided for @onboardingProfileDescription.
  ///
  /// In en, this message translates to:
  /// **'Add a name, bio, and photo so other crafters can recognize you.'**
  String get onboardingProfileDescription;

  /// No description provided for @onboardingAvatarUploading.
  ///
  /// In en, this message translates to:
  /// **'Uploading photo...'**
  String get onboardingAvatarUploading;

  /// No description provided for @onboardingAvatarUploadFailed.
  ///
  /// In en, this message translates to:
  /// **'Photo upload failed. Try again.'**
  String get onboardingAvatarUploadFailed;

  /// No description provided for @onboardingHandleLabel.
  ///
  /// In en, this message translates to:
  /// **'Signed in as @{handle}'**
  String onboardingHandleLabel(String handle);

  /// No description provided for @onboardingCraftsTitle.
  ///
  /// In en, this message translates to:
  /// **'What do you make?'**
  String get onboardingCraftsTitle;

  /// No description provided for @onboardingCraftsDescription.
  ///
  /// In en, this message translates to:
  /// **'Choose as many crafts as you like. You can change these later.'**
  String get onboardingCraftsDescription;

  /// No description provided for @onboardingInstagramTitle.
  ///
  /// In en, this message translates to:
  /// **'Find your crafting community'**
  String get onboardingInstagramTitle;

  /// No description provided for @onboardingInstagramDescription.
  ///
  /// In en, this message translates to:
  /// **'Connecting Instagram is optional. CraftSky uses your choices only to help match accounts and import who you follow.'**
  String get onboardingInstagramDescription;

  /// No description provided for @onboardingSaveError.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t save your profile. Your changes are still here; try again.'**
  String get onboardingSaveError;

  /// No description provided for @onboardingLoadError.
  ///
  /// In en, this message translates to:
  /// **'We couldn\'t load your profile.'**
  String get onboardingLoadError;

  /// No description provided for @onboardingRetry.
  ///
  /// In en, this message translates to:
  /// **'Try again'**
  String get onboardingRetry;

  /// No description provided for @onboardingProgressSemantics.
  ///
  /// In en, this message translates to:
  /// **'Onboarding step {current} of {total}'**
  String onboardingProgressSemantics(int current, int total);

  /// Plain self-declared business label on a full profile.
  ///
  /// In en, this message translates to:
  /// **'Business'**
  String get businessProfileLabel;

  /// Tab label for featured business products.
  ///
  /// In en, this message translates to:
  /// **'Products'**
  String get profileTabProducts;

  /// Tab label for upcoming business events.
  ///
  /// In en, this message translates to:
  /// **'Upcoming Events'**
  String get profileTabUpcomingEvents;

  /// About section heading for business types.
  ///
  /// In en, this message translates to:
  /// **'Business types'**
  String get businessTypesHeading;

  /// About section heading for business offerings.
  ///
  /// In en, this message translates to:
  /// **'Offerings'**
  String get businessOfferingsHeading;

  /// About section heading for hydrated business locality and country.
  ///
  /// In en, this message translates to:
  /// **'Location'**
  String get businessLocationHeading;

  /// About section heading for authored service area text.
  ///
  /// In en, this message translates to:
  /// **'Service area'**
  String get businessServiceAreaHeading;

  /// About section heading for authored hours text.
  ///
  /// In en, this message translates to:
  /// **'Hours'**
  String get businessHoursHeading;

  /// Bounded readable fallback for an unknown safe federated business catalog value.
  ///
  /// In en, this message translates to:
  /// **'Other: {value}'**
  String businessUnknownValue(String value);

  /// Hydrated business locality followed by localized country.
  ///
  /// In en, this message translates to:
  /// **'{locality}, {country}'**
  String businessLocationValue(String locality, String country);

  /// No description provided for @businessTypeDyer.
  ///
  /// In en, this message translates to:
  /// **'Dyer'**
  String get businessTypeDyer;

  /// No description provided for @businessTypeFiberProducer.
  ///
  /// In en, this message translates to:
  /// **'Fiber producer'**
  String get businessTypeFiberProducer;

  /// No description provided for @businessTypeFiberProcessor.
  ///
  /// In en, this message translates to:
  /// **'Fiber processor'**
  String get businessTypeFiberProcessor;

  /// No description provided for @businessTypeYarnShop.
  ///
  /// In en, this message translates to:
  /// **'Yarn shop'**
  String get businessTypeYarnShop;

  /// No description provided for @businessTypeFabricShop.
  ///
  /// In en, this message translates to:
  /// **'Fabric shop'**
  String get businessTypeFabricShop;

  /// No description provided for @businessTypeCraftSupplyShop.
  ///
  /// In en, this message translates to:
  /// **'Craft supply shop'**
  String get businessTypeCraftSupplyShop;

  /// No description provided for @businessTypePatternDesigner.
  ///
  /// In en, this message translates to:
  /// **'Pattern designer'**
  String get businessTypePatternDesigner;

  /// No description provided for @businessTypeFinishedGoodsMaker.
  ///
  /// In en, this message translates to:
  /// **'Finished goods maker'**
  String get businessTypeFinishedGoodsMaker;

  /// No description provided for @businessTypeToolMaker.
  ///
  /// In en, this message translates to:
  /// **'Tool maker'**
  String get businessTypeToolMaker;

  /// No description provided for @businessTypeTeacher.
  ///
  /// In en, this message translates to:
  /// **'Teacher'**
  String get businessTypeTeacher;

  /// No description provided for @businessTypeCraftStudio.
  ///
  /// In en, this message translates to:
  /// **'Craft studio'**
  String get businessTypeCraftStudio;

  /// No description provided for @businessTypeRepairService.
  ///
  /// In en, this message translates to:
  /// **'Repair service'**
  String get businessTypeRepairService;

  /// No description provided for @businessTypeTechnicalEditor.
  ///
  /// In en, this message translates to:
  /// **'Technical editor'**
  String get businessTypeTechnicalEditor;

  /// No description provided for @businessTypePhotographer.
  ///
  /// In en, this message translates to:
  /// **'Photographer'**
  String get businessTypePhotographer;

  /// No description provided for @businessTypePublisher.
  ///
  /// In en, this message translates to:
  /// **'Publisher'**
  String get businessTypePublisher;

  /// No description provided for @businessTypeOtherCraftBusiness.
  ///
  /// In en, this message translates to:
  /// **'Other craft business'**
  String get businessTypeOtherCraftBusiness;

  /// No description provided for @businessOfferingYarn.
  ///
  /// In en, this message translates to:
  /// **'Yarn'**
  String get businessOfferingYarn;

  /// No description provided for @businessOfferingFiber.
  ///
  /// In en, this message translates to:
  /// **'Fiber'**
  String get businessOfferingFiber;

  /// No description provided for @businessOfferingFabric.
  ///
  /// In en, this message translates to:
  /// **'Fabric'**
  String get businessOfferingFabric;

  /// No description provided for @businessOfferingPatterns.
  ///
  /// In en, this message translates to:
  /// **'Patterns'**
  String get businessOfferingPatterns;

  /// No description provided for @businessOfferingKits.
  ///
  /// In en, this message translates to:
  /// **'Kits'**
  String get businessOfferingKits;

  /// No description provided for @businessOfferingNotions.
  ///
  /// In en, this message translates to:
  /// **'Notions'**
  String get businessOfferingNotions;

  /// No description provided for @businessOfferingTools.
  ///
  /// In en, this message translates to:
  /// **'Tools'**
  String get businessOfferingTools;

  /// No description provided for @businessOfferingFinishedGoods.
  ///
  /// In en, this message translates to:
  /// **'Finished goods'**
  String get businessOfferingFinishedGoods;

  /// No description provided for @businessOfferingCustomWork.
  ///
  /// In en, this message translates to:
  /// **'Custom work'**
  String get businessOfferingCustomWork;

  /// No description provided for @businessOfferingRepairs.
  ///
  /// In en, this message translates to:
  /// **'Repairs'**
  String get businessOfferingRepairs;

  /// No description provided for @businessOfferingClasses.
  ///
  /// In en, this message translates to:
  /// **'Classes'**
  String get businessOfferingClasses;

  /// No description provided for @businessOfferingStudioHire.
  ///
  /// In en, this message translates to:
  /// **'Studio hire'**
  String get businessOfferingStudioHire;

  /// No description provided for @businessOfferingWholesale.
  ///
  /// In en, this message translates to:
  /// **'Wholesale'**
  String get businessOfferingWholesale;

  /// No description provided for @businessOfferingDigitalProducts.
  ///
  /// In en, this message translates to:
  /// **'Digital products'**
  String get businessOfferingDigitalProducts;

  /// No description provided for @businessOfferingTechnicalEditing.
  ///
  /// In en, this message translates to:
  /// **'Technical editing'**
  String get businessOfferingTechnicalEditing;

  /// No description provided for @businessOfferingPhotographyServices.
  ///
  /// In en, this message translates to:
  /// **'Photography services'**
  String get businessOfferingPhotographyServices;

  /// No description provided for @businessOfferingFiberProcessing.
  ///
  /// In en, this message translates to:
  /// **'Fiber processing'**
  String get businessOfferingFiberProcessing;

  /// No description provided for @businessActionShop.
  ///
  /// In en, this message translates to:
  /// **'Shop'**
  String get businessActionShop;

  /// No description provided for @businessActionBrowsePatterns.
  ///
  /// In en, this message translates to:
  /// **'Browse patterns'**
  String get businessActionBrowsePatterns;

  /// No description provided for @businessActionRequestCustomOrder.
  ///
  /// In en, this message translates to:
  /// **'Request custom order'**
  String get businessActionRequestCustomOrder;

  /// No description provided for @businessActionBookClass.
  ///
  /// In en, this message translates to:
  /// **'Book class'**
  String get businessActionBookClass;

  /// No description provided for @businessActionBookAppointment.
  ///
  /// In en, this message translates to:
  /// **'Book appointment'**
  String get businessActionBookAppointment;

  /// No description provided for @businessActionViewEventCalendar.
  ///
  /// In en, this message translates to:
  /// **'View event calendar'**
  String get businessActionViewEventCalendar;

  /// No description provided for @businessActionEmail.
  ///
  /// In en, this message translates to:
  /// **'Email'**
  String get businessActionEmail;

  /// No description provided for @businessActionVisitWebsite.
  ///
  /// In en, this message translates to:
  /// **'Visit website'**
  String get businessActionVisitWebsite;

  /// No description provided for @businessActionWholesaleEnquiries.
  ///
  /// In en, this message translates to:
  /// **'Wholesale enquiries'**
  String get businessActionWholesaleEnquiries;

  /// Owner empty state in the Products profile tab.
  ///
  /// In en, this message translates to:
  /// **'Add featured products to help visitors find your work.'**
  String get businessProductsOwnerEmpty;

  /// Visitor empty state in the Products profile tab.
  ///
  /// In en, this message translates to:
  /// **'No featured products yet.'**
  String get businessProductsVisitorEmpty;

  /// Owner setup action in the Products profile tab.
  ///
  /// In en, this message translates to:
  /// **'Manage products'**
  String get businessProductsManageAction;

  /// Accessible label for a featured product external-link card.
  ///
  /// In en, this message translates to:
  /// **'Open {title} outside CraftSky'**
  String businessProductOpen(String title);

  /// Owner placeholder empty state in Upcoming Events.
  ///
  /// In en, this message translates to:
  /// **'Add an event appearance to share what’s coming up.'**
  String get businessEventsOwnerEmpty;

  /// Visitor placeholder empty state in Upcoming Events.
  ///
  /// In en, this message translates to:
  /// **'No upcoming events yet.'**
  String get businessEventsVisitorEmpty;

  /// Owner placeholder setup action in Upcoming Events.
  ///
  /// In en, this message translates to:
  /// **'Manage events'**
  String get businessEventsManageAction;

  /// Initial error in the Upcoming Events profile tab.
  ///
  /// In en, this message translates to:
  /// **'Upcoming events could not be loaded.'**
  String get businessEventsLoadError;

  /// Incremental pagination error in the Upcoming Events profile tab.
  ///
  /// In en, this message translates to:
  /// **'Couldn’t load more events.'**
  String get businessEventsLoadMoreError;

  /// Refresh error shown while confirmed upcoming events remain visible.
  ///
  /// In en, this message translates to:
  /// **'Couldn’t refresh upcoming events.'**
  String get businessEventsRefreshError;

  /// Action retrying an event list request.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get businessEventsRetryAction;

  /// Action requesting the next public event page.
  ///
  /// In en, this message translates to:
  /// **'Load more'**
  String get businessEventsLoadMoreAction;

  /// Action refreshing public upcoming events from the first page.
  ///
  /// In en, this message translates to:
  /// **'Refresh'**
  String get businessEventsRefreshAction;

  /// Public upcoming event end-of-list state.
  ///
  /// In en, this message translates to:
  /// **'You’ve reached the end.'**
  String get businessEventsEnd;

  /// Compact display-only time label for an all-day event.
  ///
  /// In en, this message translates to:
  /// **'All day'**
  String get businessEventAllDayDisplay;

  /// Localized time range for an event.
  ///
  /// In en, this message translates to:
  /// **'{start}–{end}'**
  String businessEventTimeRange(String start, String end);

  /// Localized inclusive date range for an all-day event whose wire end is exclusive.
  ///
  /// In en, this message translates to:
  /// **'{start}–{end}, {year}'**
  String businessEventDateRange(String start, String end, int year);

  /// No description provided for @businessEventRoleOrganizer.
  ///
  /// In en, this message translates to:
  /// **'Organizer'**
  String get businessEventRoleOrganizer;

  /// No description provided for @businessEventRoleInstructor.
  ///
  /// In en, this message translates to:
  /// **'Instructor'**
  String get businessEventRoleInstructor;

  /// No description provided for @businessEventRoleVendor.
  ///
  /// In en, this message translates to:
  /// **'Vendor'**
  String get businessEventRoleVendor;

  /// No description provided for @businessEventRoleExhibitor.
  ///
  /// In en, this message translates to:
  /// **'Exhibitor'**
  String get businessEventRoleExhibitor;

  /// No description provided for @businessEventRoleSpeaker.
  ///
  /// In en, this message translates to:
  /// **'Speaker'**
  String get businessEventRoleSpeaker;

  /// No description provided for @businessEventRoleDemonstrator.
  ///
  /// In en, this message translates to:
  /// **'Demonstrator'**
  String get businessEventRoleDemonstrator;

  /// No description provided for @businessEventModeInPerson.
  ///
  /// In en, this message translates to:
  /// **'In person'**
  String get businessEventModeInPerson;

  /// No description provided for @businessEventModeOnline.
  ///
  /// In en, this message translates to:
  /// **'Online'**
  String get businessEventModeOnline;

  /// No description provided for @businessEventModeHybrid.
  ///
  /// In en, this message translates to:
  /// **'Hybrid'**
  String get businessEventModeHybrid;

  /// App bar title on event detail.
  ///
  /// In en, this message translates to:
  /// **'Event'**
  String get businessEventDetailTitle;

  /// Safe title when an event is absent, blocked, or moderated.
  ///
  /// In en, this message translates to:
  /// **'Event unavailable'**
  String get businessEventUnavailableTitle;

  /// Safe body when event detail is unavailable.
  ///
  /// In en, this message translates to:
  /// **'This event can’t be viewed.'**
  String get businessEventUnavailableBody;

  /// Title for reporting a visible business event.
  ///
  /// In en, this message translates to:
  /// **'Report event'**
  String get businessEventReportAction;

  /// Action label for reporting a visible business event.
  ///
  /// In en, this message translates to:
  /// **'Report'**
  String get businessEventReportActionShort;

  /// Generic retryable event detail error.
  ///
  /// In en, this message translates to:
  /// **'Event details could not be loaded.'**
  String get businessEventDetailLoadError;

  /// No description provided for @businessEventStatusScheduled.
  ///
  /// In en, this message translates to:
  /// **'Scheduled'**
  String get businessEventStatusScheduled;

  /// No description provided for @businessEventStatusCancelled.
  ///
  /// In en, this message translates to:
  /// **'Cancelled'**
  String get businessEventStatusCancelled;

  /// No description provided for @businessEventStatusPostponed.
  ///
  /// In en, this message translates to:
  /// **'Postponed'**
  String get businessEventStatusPostponed;

  /// No description provided for @businessEventLifecycleUpcoming.
  ///
  /// In en, this message translates to:
  /// **'Upcoming'**
  String get businessEventLifecycleUpcoming;

  /// No description provided for @businessEventLifecyclePast.
  ///
  /// In en, this message translates to:
  /// **'Past'**
  String get businessEventLifecyclePast;

  /// No description provided for @businessEventDateLabel.
  ///
  /// In en, this message translates to:
  /// **'Date'**
  String get businessEventDateLabel;

  /// No description provided for @businessEventTimeLabel.
  ///
  /// In en, this message translates to:
  /// **'Time'**
  String get businessEventTimeLabel;

  /// Business event role selector heading.
  ///
  /// In en, this message translates to:
  /// **'Your role'**
  String get businessEventRolesLabel;

  /// Event attendance mode selector label.
  ///
  /// In en, this message translates to:
  /// **'Attendance mode'**
  String get businessEventModeLabel;

  /// Event lifecycle status selector label.
  ///
  /// In en, this message translates to:
  /// **'Status'**
  String get businessEventStatusLabel;

  /// No description provided for @businessEventLifecycleLabel.
  ///
  /// In en, this message translates to:
  /// **'Lifecycle'**
  String get businessEventLifecycleLabel;

  /// Event timezone selector label.
  ///
  /// In en, this message translates to:
  /// **'Timezone'**
  String get businessEventTimeZoneLabel;

  /// Optional event venue field label.
  ///
  /// In en, this message translates to:
  /// **'Venue (optional)'**
  String get businessEventVenueLabel;

  /// No description provided for @businessEventPublishedLabel.
  ///
  /// In en, this message translates to:
  /// **'Published'**
  String get businessEventPublishedLabel;

  /// Friendly event publication date copy.
  ///
  /// In en, this message translates to:
  /// **'Published {date}'**
  String businessEventPublishedOn(String date);

  /// External action opening hydrated event information.
  ///
  /// In en, this message translates to:
  /// **'Event website'**
  String get businessEventWebsiteAction;

  /// External action opening hydrated event registration.
  ///
  /// In en, this message translates to:
  /// **'Register'**
  String get businessEventRegistrationAction;

  /// Title of the business product management page.
  ///
  /// In en, this message translates to:
  /// **'Products'**
  String get businessProductsSettingsTitle;

  /// Action for adding a featured product.
  ///
  /// In en, this message translates to:
  /// **'Add product'**
  String get businessProductsAdd;

  /// Empty state on the product management page.
  ///
  /// In en, this message translates to:
  /// **'No featured products yet.'**
  String get businessProductsEmpty;

  /// Owner guard shown when a regular account reaches product management.
  ///
  /// In en, this message translates to:
  /// **'Product management is available to business accounts.'**
  String get businessProductsUnavailable;

  /// Error shown when product management cannot load.
  ///
  /// In en, this message translates to:
  /// **'Products could not be loaded.'**
  String get businessProductsLoadError;

  /// Action for retrying product management loading.
  ///
  /// In en, this message translates to:
  /// **'Retry'**
  String get businessProductsRetry;

  /// Conflict warning for a stale complete business declaration.
  ///
  /// In en, this message translates to:
  /// **'These business details changed elsewhere. Reload the complete current profile before trying again.'**
  String get businessProductsConflict;

  /// Action that replaces a stale product draft with the complete current profile.
  ///
  /// In en, this message translates to:
  /// **'Reload current profile'**
  String get businessProductsReload;

  /// Bounded error shown after product validation or save failure.
  ///
  /// In en, this message translates to:
  /// **'Products could not be saved. Check the fields and try again.'**
  String get businessProductsSaveError;

  /// Recoverable product image upload failure.
  ///
  /// In en, this message translates to:
  /// **'The image could not be uploaded. Try again.'**
  String get businessProductsUploadError;

  /// Accessible busy label while a business surface loads.
  ///
  /// In en, this message translates to:
  /// **'Loading business information'**
  String get businessLoading;

  /// Accessible busy label while business changes are saved.
  ///
  /// In en, this message translates to:
  /// **'Saving business information'**
  String get businessSaving;

  /// Accessible busy label while a business image uploads.
  ///
  /// In en, this message translates to:
  /// **'Uploading image'**
  String get businessImageUploading;

  /// Accessible authored product count and limit.
  ///
  /// In en, this message translates to:
  /// **'{count} of {limit} products'**
  String businessProductsCount(int count, int limit);

  /// Accessible action for editing a product.
  ///
  /// In en, this message translates to:
  /// **'Edit {title}'**
  String businessProductEdit(String title);

  /// Accessible action for removing a product.
  ///
  /// In en, this message translates to:
  /// **'Remove {title}'**
  String businessProductRemove(String title);

  /// Title for confirming immediate product removal.
  ///
  /// In en, this message translates to:
  /// **'Remove product?'**
  String get businessProductRemoveConfirmTitle;

  /// Message for confirming immediate product removal.
  ///
  /// In en, this message translates to:
  /// **'Remove {title} from your featured products? This change is saved immediately.'**
  String businessProductRemoveConfirmMessage(String title);

  /// Destructive confirmation action for removing a product.
  ///
  /// In en, this message translates to:
  /// **'Remove'**
  String get businessProductRemoveConfirm;

  /// Cancellation action for retaining a product.
  ///
  /// In en, this message translates to:
  /// **'Keep product'**
  String get businessProductRemoveCancel;

  /// Accessible action for moving a product earlier.
  ///
  /// In en, this message translates to:
  /// **'Move {title} up'**
  String businessProductMoveUp(String title);

  /// Accessible action for moving a product later.
  ///
  /// In en, this message translates to:
  /// **'Move {title} down'**
  String businessProductMoveDown(String title);

  /// Heading for the new product editor.
  ///
  /// In en, this message translates to:
  /// **'Add product'**
  String get businessProductEditorAddTitle;

  /// Heading for the existing product editor.
  ///
  /// In en, this message translates to:
  /// **'Edit product'**
  String get businessProductEditorEditTitle;

  /// Product title field label.
  ///
  /// In en, this message translates to:
  /// **'Title'**
  String get businessProductTitleLabel;

  /// Product HTTPS destination field label.
  ///
  /// In en, this message translates to:
  /// **'Destination'**
  String get businessProductDestinationLabel;

  /// Example HTTPS destination in the product editor.
  ///
  /// In en, this message translates to:
  /// **'https://example.com/product'**
  String get businessProductDestinationHint;

  /// Optional canonical product price amount field label.
  ///
  /// In en, this message translates to:
  /// **'Amount'**
  String get businessProductAmountLabel;

  /// Optional ISO currency code field label.
  ///
  /// In en, this message translates to:
  /// **'Currency'**
  String get businessProductCurrencyLabel;

  /// Optional product image alt text field label.
  ///
  /// In en, this message translates to:
  /// **'Image description'**
  String get businessProductAltLabel;

  /// Action for selecting a required product image.
  ///
  /// In en, this message translates to:
  /// **'Add image'**
  String get businessProductAddImage;

  /// Action for replacing a saved product image.
  ///
  /// In en, this message translates to:
  /// **'Replace image'**
  String get businessProductReplaceImage;

  /// Action for explicitly removing a product image draft.
  ///
  /// In en, this message translates to:
  /// **'Remove image'**
  String get businessProductRemoveImage;

  /// Action for applying one product editor draft to the manager.
  ///
  /// In en, this message translates to:
  /// **'Save product'**
  String get businessProductSave;

  /// Action for closing the product editor without applying changes.
  ///
  /// In en, this message translates to:
  /// **'Cancel'**
  String get businessProductCancel;

  /// Required product title validation.
  ///
  /// In en, this message translates to:
  /// **'Add a title.'**
  String get businessProductTitleRequired;

  /// Product title limit validation.
  ///
  /// In en, this message translates to:
  /// **'Use 150 characters or fewer.'**
  String get businessProductTitleTooLong;

  /// Product destination validation.
  ///
  /// In en, this message translates to:
  /// **'Enter a credential-free HTTPS link.'**
  String get businessProductDestinationInvalid;

  /// Validation shown when another featured product already uses this destination.
  ///
  /// In en, this message translates to:
  /// **'Use a different destination. Each product must link to a unique page.'**
  String get businessProductDestinationDuplicate;

  /// Required product image validation.
  ///
  /// In en, this message translates to:
  /// **'Add an image.'**
  String get businessProductImageRequired;

  /// Canonical optional product price validation.
  ///
  /// In en, this message translates to:
  /// **'Enter a canonical amount and uppercase ISO currency code, or clear both.'**
  String get businessProductPriceInvalid;

  /// Title and submit action for a new business event.
  ///
  /// In en, this message translates to:
  /// **'Create event'**
  String get businessEventCreateTitle;

  /// Title for editing a business event.
  ///
  /// In en, this message translates to:
  /// **'Edit event'**
  String get businessEventEditTitle;

  /// Submit action for an existing business event.
  ///
  /// In en, this message translates to:
  /// **'Save event'**
  String get businessEventSave;

  /// Event name field label.
  ///
  /// In en, this message translates to:
  /// **'Event name'**
  String get businessEventNameLabel;

  /// Required event name validation.
  ///
  /// In en, this message translates to:
  /// **'Add an event name.'**
  String get businessEventNameRequired;

  /// Event start date and time field label.
  ///
  /// In en, this message translates to:
  /// **'Start'**
  String get businessEventStartLabel;

  /// Event end date and time field label.
  ///
  /// In en, this message translates to:
  /// **'End'**
  String get businessEventEndLabel;

  /// Placeholder for an event date and time picker.
  ///
  /// In en, this message translates to:
  /// **'Select date and time'**
  String get businessEventDateTimeHint;

  /// Hint for local event boundary input.
  ///
  /// In en, this message translates to:
  /// **'YYYY-MM-DD HH:MM'**
  String get businessEventTimeHint;

  /// Invalid event boundary validation.
  ///
  /// In en, this message translates to:
  /// **'Enter a valid start and end.'**
  String get businessEventTimeInvalid;

  /// Validation shown when an event end is not after its start.
  ///
  /// In en, this message translates to:
  /// **'End must be after start.'**
  String get businessEventEndAfterStart;

  /// All-day event toggle label.
  ///
  /// In en, this message translates to:
  /// **'All-day event'**
  String get businessEventAllDay;

  /// Required event role validation.
  ///
  /// In en, this message translates to:
  /// **'Choose at least one role.'**
  String get businessEventRolesRequired;

  /// Optional event summary field label.
  ///
  /// In en, this message translates to:
  /// **'Summary (optional)'**
  String get businessEventSummaryLabel;

  /// Optional event information link field label.
  ///
  /// In en, this message translates to:
  /// **'Event link (optional)'**
  String get businessEventUriLabel;

  /// Optional event registration link field label.
  ///
  /// In en, this message translates to:
  /// **'Registration link (optional)'**
  String get businessEventRegistrationUriLabel;

  /// Optional event image alt text label.
  ///
  /// In en, this message translates to:
  /// **'Image description'**
  String get businessEventImageDescriptionLabel;

  /// Action for adding an optional event image.
  ///
  /// In en, this message translates to:
  /// **'Add image'**
  String get businessEventAddImage;

  /// Action for replacing an event image.
  ///
  /// In en, this message translates to:
  /// **'Replace image'**
  String get businessEventReplaceImage;

  /// Action for removing an event image.
  ///
  /// In en, this message translates to:
  /// **'Remove image'**
  String get businessEventRemoveImage;

  /// Recoverable event image upload failure.
  ///
  /// In en, this message translates to:
  /// **'The image could not be uploaded. Try again.'**
  String get businessEventUploadError;

  /// Bounded event form validation message.
  ///
  /// In en, this message translates to:
  /// **'Check the event details and try again.'**
  String get businessEventValidationError;

  /// Event mutation failure message.
  ///
  /// In en, this message translates to:
  /// **'The event could not be saved. Try again.'**
  String get businessEventSaveError;

  /// Stale event conflict warning.
  ///
  /// In en, this message translates to:
  /// **'This event changed elsewhere. Reload the current event before trying again.'**
  String get businessEventConflict;

  /// Action for reloading an event after conflict.
  ///
  /// In en, this message translates to:
  /// **'Reload current event'**
  String get businessEventReload;

  /// Unsaved event confirmation title.
  ///
  /// In en, this message translates to:
  /// **'Discard event changes?'**
  String get businessEventDiscardTitle;

  /// Unsaved event confirmation body.
  ///
  /// In en, this message translates to:
  /// **'Your unsaved event changes will be lost.'**
  String get businessEventDiscardMessage;

  /// Confirm discarding event edits.
  ///
  /// In en, this message translates to:
  /// **'Discard'**
  String get businessEventDiscard;

  /// Cancel discarding event edits.
  ///
  /// In en, this message translates to:
  /// **'Keep editing'**
  String get businessEventKeepEditing;

  /// Owner explanation for an event suppressed because the account is not a business.
  ///
  /// In en, this message translates to:
  /// **'Your account is not currently presented as a business.'**
  String get businessEventDiagnosticOwnerNotBusiness;

  /// Owner explanation for an invalid event time range.
  ///
  /// In en, this message translates to:
  /// **'The event’s time range is invalid.'**
  String get businessEventDiagnosticInvalidTimeRange;

  /// Owner explanation for an event exceeding the duration limit.
  ///
  /// In en, this message translates to:
  /// **'The event is longer than the supported limit.'**
  String get businessEventDiagnosticDurationExceedsLimit;

  /// Owner explanation for a moderated event record.
  ///
  /// In en, this message translates to:
  /// **'This event is hidden by moderation.'**
  String get businessEventDiagnosticRecordModerated;

  /// Owner explanation for an event excluded because it ended.
  ///
  /// In en, this message translates to:
  /// **'This event has ended.'**
  String get businessEventDiagnosticEnded;

  /// Owner explanation for a cancelled event.
  ///
  /// In en, this message translates to:
  /// **'This event is cancelled.'**
  String get businessEventDiagnosticCancelled;

  /// Owner explanation for a postponed event.
  ///
  /// In en, this message translates to:
  /// **'This event is postponed.'**
  String get businessEventDiagnosticPostponed;

  /// Title of the owner event management page.
  ///
  /// In en, this message translates to:
  /// **'Events'**
  String get businessEventsSettingsTitle;

  /// Owner event management upcoming tab.
  ///
  /// In en, this message translates to:
  /// **'Upcoming'**
  String get businessEventsUpcomingTab;

  /// Owner event management history tab.
  ///
  /// In en, this message translates to:
  /// **'History'**
  String get businessEventsHistoryTab;

  /// Owner guard shown when a regular account reaches event management.
  ///
  /// In en, this message translates to:
  /// **'Event management is available to business accounts.'**
  String get businessEventsUnavailable;

  /// Initial owner event management loading error.
  ///
  /// In en, this message translates to:
  /// **'Events could not be loaded.'**
  String get businessEventsOwnerLoadError;

  /// Owner event refresh error retaining confirmed rows.
  ///
  /// In en, this message translates to:
  /// **'Events could not be refreshed.'**
  String get businessEventsOwnerRefreshError;

  /// Empty state for owner upcoming events.
  ///
  /// In en, this message translates to:
  /// **'No upcoming events yet.'**
  String get businessEventsUpcomingEmpty;

  /// Empty state for owner event history.
  ///
  /// In en, this message translates to:
  /// **'No event history yet.'**
  String get businessEventsHistoryEmpty;

  /// Accessible menu label for managing one owner event.
  ///
  /// In en, this message translates to:
  /// **'Manage {name}'**
  String businessEventManage(String name);

  /// Owner action opening event editing.
  ///
  /// In en, this message translates to:
  /// **'Edit event'**
  String get businessEventEditAction;

  /// Owner lifecycle action setting an event to cancelled.
  ///
  /// In en, this message translates to:
  /// **'Cancel event'**
  String get businessEventCancelAction;

  /// Owner lifecycle action setting an event to postponed.
  ///
  /// In en, this message translates to:
  /// **'Postpone event'**
  String get businessEventPostponeAction;

  /// Owner action starting destructive event deletion.
  ///
  /// In en, this message translates to:
  /// **'Delete event'**
  String get businessEventDeleteAction;

  /// Destructive event deletion confirmation title.
  ///
  /// In en, this message translates to:
  /// **'Delete this event?'**
  String get businessEventDeleteConfirmTitle;

  /// Destructive event deletion confirmation body.
  ///
  /// In en, this message translates to:
  /// **'This permanently deletes the event record from your account.'**
  String get businessEventDeleteConfirmMessage;

  /// Action confirming permanent event deletion.
  ///
  /// In en, this message translates to:
  /// **'Delete'**
  String get businessEventDeleteConfirmAction;
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
