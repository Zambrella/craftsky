// ignore: unused_import
import 'package:intl/intl.dart' as intl;
import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'CraftSky';

  @override
  String get homeSubtitle => 'Scaffold ready';

  @override
  String homeVersionLabel(String version) {
    return 'v$version';
  }

  @override
  String get feedTitle => 'Feed';

  @override
  String get navigationMenuTooltip => 'Open navigation menu';

  @override
  String get navigationProfile => 'Profile';

  @override
  String get navigationSaved => 'Saved';

  @override
  String get navigationScheduled => 'Scheduled';

  @override
  String get navigationTerms => 'Terms';

  @override
  String get navigationPrivacy => 'Privacy';

  @override
  String get navigationFeedback => 'Feedback';

  @override
  String navigationBuildVersion(String version, String buildNumber) {
    return '$version ($buildNumber)';
  }

  @override
  String get navigationLinkOpenError => 'Couldn\'t open that link.';

  @override
  String get externalLinkConfirmTitle => 'Open link?';

  @override
  String get externalLinkConfirmBody => 'This will open outside CraftSky.';

  @override
  String get externalLinkConfirmAction => 'Open link';

  @override
  String get notificationsTitle => 'Notifications';

  @override
  String get notificationsEmpty => 'No notifications yet.';

  @override
  String get notificationsLoadError => 'Notifications didn\'t load.';

  @override
  String get notificationsLoadMore => 'Load more';

  @override
  String notificationFollowRow(String actor) {
    return '$actor followed you';
  }

  @override
  String notificationLikeRow(String actor) {
    return '$actor liked your post';
  }

  @override
  String notificationLikeCommentRow(String actor) {
    return '$actor liked your comment';
  }

  @override
  String notificationLikeReplyRow(String actor) {
    return '$actor liked your reply';
  }

  @override
  String notificationRepostRow(String actor) {
    return '$actor reposted your post';
  }

  @override
  String notificationRepostCommentRow(String actor) {
    return '$actor reposted your comment';
  }

  @override
  String notificationRepostReplyRow(String actor) {
    return '$actor reposted your reply';
  }

  @override
  String notificationReplyRow(String actor) {
    return '$actor commented on your post';
  }

  @override
  String notificationReplyToCommentRow(String actor) {
    return '$actor replied to your comment';
  }

  @override
  String notificationReplyToReplyRow(String actor) {
    return '$actor replied to your reply';
  }

  @override
  String notificationMentionRow(String actor) {
    return '$actor mentioned you';
  }

  @override
  String notificationQuoteRow(String actor) {
    return '$actor quoted your post';
  }

  @override
  String get notificationGenericRow => 'New activity';

  @override
  String get notificationUnavailableRow => 'Activity unavailable';

  @override
  String notificationInstagramMatchActorRow(String actor) {
    return 'You found $actor through your Instagram following';
  }

  @override
  String get notificationSettingsAction => 'Notification settings';

  @override
  String get notificationSettingsIntro =>
      'Category preferences apply to all devices signed in to this account.';

  @override
  String get notificationDeviceDisabled =>
      'Notifications are disabled on this device';

  @override
  String get notificationDeviceDisabledDescription =>
      'Account preferences still apply. Enable alerts in system settings.';

  @override
  String get notificationOpenSettings => 'Open settings';

  @override
  String get notificationCategoryLikes => 'Likes';

  @override
  String get notificationCategoryFollows => 'Follows';

  @override
  String get notificationCategoryReplies => 'Replies';

  @override
  String get notificationCategoryMentions => 'Mentions';

  @override
  String get notificationCategoryQuotes => 'Quotes';

  @override
  String get notificationCategoryReposts => 'Reposts';

  @override
  String get notificationCategoryInstagramMatches => 'Instagram matches';

  @override
  String get notificationInstagramMatchPreferenceDescription =>
      'Push alerts are based on your private Instagram matches and never name the matched account.';

  @override
  String get notificationCategoryEverythingElse => 'Everything else';

  @override
  String get notificationPreferenceFrom => 'From';

  @override
  String get notificationScopeEveryone => 'Everyone';

  @override
  String get notificationScopePeopleIFollow => 'People I follow';

  @override
  String get notificationPushEnabled => 'Push notifications';

  @override
  String get notificationPreferenceSaveError =>
      'Could not save notification preference';

  @override
  String get notificationBannerOpen => 'Open';

  @override
  String notificationNewActivityCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count new activities',
      one: '1 new activity',
    );
    return '$_temp0';
  }

  @override
  String get welcomeTitle => 'Welcome';

  @override
  String get welcomeJoinTitle => 'Join CraftSky';

  @override
  String get welcomeSubtitle =>
      'Share what you make. Find people who make what you love.';

  @override
  String get welcomeRegisterAction => 'Register';

  @override
  String get welcomeRegistrationHandoff =>
      'You\'ll create your account with Bluesky, then return to CraftSky.';

  @override
  String get welcomeOr => 'Or';

  @override
  String get welcomeSignInAction => 'Sign in';

  @override
  String get welcomeCreateAccountAction => 'Create an account';

  @override
  String get registrationProviderDisclosure =>
      'Bluesky hosts your portable account, which you can use with Craftsky.';

  @override
  String get welcomeAtmosphereTitle => 'What is an Atmosphere account?';

  @override
  String get welcomeAtmosphereBody =>
      'CraftSky is built on the AT Protocol, so your account, posts and social graph are portable. You can use an existing Bluesky or compatible Atmosphere account, or register a new one.';

  @override
  String get welcomeLegalPrefix => 'By continuing, you agree to our';

  @override
  String get welcomeLegalAnd => 'and';

  @override
  String get welcomePrivacyAction => 'Privacy Policy';

  @override
  String get signInTitle => 'Sign in';

  @override
  String get addAccountTitle => 'Add account';

  @override
  String get addAccountDescription =>
      'Sign in to another account. Your current account stays signed in.';

  @override
  String get accountSwitcherAdd => 'Add account';

  @override
  String get accountSwitcherMaximum => 'Maximum of 5 accounts';

  @override
  String get accountSwitcherTooltip => 'Switch account';

  @override
  String get accountSwitcherLongPressHint => 'Long press to switch account';

  @override
  String get accountSwitchingLabel => 'Switching account';

  @override
  String get accountIdentityFallback => 'Account';

  @override
  String get signInHandleLabel => 'Your Atmosphere Handle';

  @override
  String get signInContinueAction => 'Continue';

  @override
  String get signInHandleRequiredError => 'Please enter a handle.';

  @override
  String get signInInvalidHandleError => 'We couldn\'t recognise that handle.';

  @override
  String get signInServerUnavailableError =>
      'Couldn\'t reach the server. Please try again.';

  @override
  String get signInBrowserLaunchError =>
      'Couldn\'t open the browser. Check that you have one installed.';

  @override
  String get signInGenericError => 'Something went wrong. Please try again.';

  @override
  String get authCompleteSigningIn => 'Signing in…';

  @override
  String get authCompleteTimedOutError =>
      'That sign-in link expired. Please sign in again.';

  @override
  String get authCompleteStorageError =>
      'Couldn\'t save your session securely. Please sign in again.';

  @override
  String get authCompleteGenericError =>
      'Couldn\'t complete sign-in. Please sign in again.';

  @override
  String get authRegistrationCanceledError => 'Account creation was canceled.';

  @override
  String get authRegistrationProviderUnavailableError =>
      'Bluesky is temporarily unavailable. Please try again.';

  @override
  String get authRegistrationIncompleteError =>
      'We couldn\'t verify or complete account creation.';

  @override
  String get dialogConfirmDefault => 'Confirm';

  @override
  String get dialogCancelDefault => 'Cancel';

  @override
  String get dialogOkDefault => 'OK';

  @override
  String get loading => 'Loading';

  @override
  String get initializationFailedTitle => 'Initialization Failed';

  @override
  String get retryButton => 'Retry';

  @override
  String get activeAccountInitializationFailedTitle =>
      'We couldn’t load this account';

  @override
  String get activeAccountInitializationFailedBody =>
      'Try again, switch accounts, or sign out.';

  @override
  String get activeAccountSwitchAction => 'Switch account';

  @override
  String get activeAccountSignOutAction => 'Sign out';

  @override
  String get activeAccountRecoveryFailed =>
      'That didn’t work. Please try again.';

  @override
  String get backButton => 'Back';

  @override
  String get notificationDestinationUnavailableTitle =>
      'This is no longer available';

  @override
  String get notificationDestinationUnavailableBody =>
      'This post or profile may have been deleted or hidden.';

  @override
  String get notificationDestinationViewNotifications => 'View notifications';

  @override
  String get notificationDestinationRetryTitle => 'That didn\'t load';

  @override
  String get notificationDestinationRetryBody =>
      'Check your connection and try again.';

  @override
  String get feedEmpty => 'Your feed is quiet.';

  @override
  String get feedLoadError => 'Feed didn\'t load.';

  @override
  String get messengerDismiss => 'Dismiss';

  @override
  String get signOutSuccess => 'Signed out successfully.';

  @override
  String signOutSuccessWithAccount(String handle) {
    return 'Signed out successfully. Now signed in as @$handle.';
  }

  @override
  String get routingErrorTitle => 'Something went wrong';

  @override
  String get goHomeButton => 'Go home';

  @override
  String get errorNetworkUnavailable =>
      'You\'re offline. Check your connection and try again.';

  @override
  String get errorServiceUnavailable =>
      'CraftSky is having trouble right now. Please try again.';

  @override
  String get errorSessionExpired => 'Please sign in again.';

  @override
  String get errorPermissionDenied => 'You don\'t have permission to do that.';

  @override
  String get errorContentUnavailable => 'That content is no longer available.';

  @override
  String get errorStorageUnavailable =>
      'CraftSky couldn\'t access secure storage. Please try again.';

  @override
  String get errorInitializationFailed =>
      'CraftSky couldn\'t finish starting. Please try again.';

  @override
  String get errorNavigationFailed => 'That page couldn\'t be opened.';

  @override
  String get errorActionFailed => 'That didn\'t work. Please try again.';

  @override
  String get errorBackgroundLoadFailed =>
      'This didn\'t load. Please try again.';

  @override
  String get errorUnexpected => 'Something went wrong. Please try again.';

  @override
  String get errorActionSignIn => 'Sign in';

  @override
  String get profileEditAction => 'Edit profile';

  @override
  String get profileVisitAction => 'Visit profile';

  @override
  String get profileSettingsAction => 'Settings';

  @override
  String get profileShareAction => 'Share';

  @override
  String get profileMoreActions => 'More profile actions';

  @override
  String get profileMuteAction => 'Mute account';

  @override
  String get profileUnmuteAction => 'Unmute account';

  @override
  String get profileBlockAction => 'Block account';

  @override
  String get profileUnblockAction => 'Unblock account';

  @override
  String get profileMuteAnnotation => 'Muted account';

  @override
  String get profileBlockingAnnotation => 'Blocked by you';

  @override
  String get profileBlockedByAnnotation => 'This account has blocked you';

  @override
  String get profileMutualBlockAnnotation => 'You have blocked each other';

  @override
  String get profileRelationshipError =>
      'Could not update account relationship.';

  @override
  String get profileMuteSuccess => 'Account muted.';

  @override
  String get profileUnmuteSuccess => 'Account unmuted.';

  @override
  String get profileBlockSuccess => 'Account blocked.';

  @override
  String get profileUnblockSuccess => 'Account unblocked.';

  @override
  String get profileBlockConfirmTitle => 'Block this account?';

  @override
  String get profileBlockConfirmBody =>
      'Blocking is public on the AT Protocol. You will no longer see or interact with each other\'s content.';

  @override
  String get profileUnblockConfirmTitle => 'Unblock this account?';

  @override
  String get profileUnblockConfirmBody =>
      'You may see and interact with each other\'s content again.';

  @override
  String get actionCancel => 'Cancel';

  @override
  String get actionConfirm => 'Confirm';

  @override
  String get destructiveActionHint => 'Destructive action';

  @override
  String get settingsMutedAccounts => 'Muted accounts';

  @override
  String get settingsBlockedAccounts => 'Blocked accounts';

  @override
  String get settingsMutedAccountsEmpty => 'You have not muted any accounts.';

  @override
  String get settingsBlockedAccountsEmpty =>
      'You have not blocked any accounts.';

  @override
  String get settingsMutedAccountsError => 'Could not load muted accounts.';

  @override
  String get settingsBlockedAccountsError => 'Could not load blocked accounts.';

  @override
  String get relationshipListRetry => 'Try again';

  @override
  String get relationshipListLoadMore => 'Load more';

  @override
  String get relationshipListUnmute => 'Unmute';

  @override
  String get relationshipListUnblock => 'Unblock';

  @override
  String get relationshipListMutationError => 'Could not update this account.';

  @override
  String get postMutedPlaceholder => 'Post from a muted account';

  @override
  String get postUnavailablePlaceholder => 'Post unavailable';

  @override
  String get postRevealAction => 'Show post';

  @override
  String get profileFollowAction => 'Follow';

  @override
  String get profileFollowingAction => 'Unfollow';

  @override
  String get profileNonCraftskyMarker => 'Non CraftSky profile';

  @override
  String get profileTabPosts => 'Posts';

  @override
  String get profileTabComments => 'Comments';

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
  String get profilePostsEmpty => 'No posts yet.';

  @override
  String get profilePostsLoadError => 'Posts didn\'t load.';

  @override
  String get profilePostsLoadMore => 'Load more posts';

  @override
  String get profileCommentsEmpty => 'No comments yet.';

  @override
  String get profileCommentsLoadError => 'Comments didn\'t load.';

  @override
  String get profileCommentsLoadMore => 'Load more comments';

  @override
  String get postThreadTitle => 'Post';

  @override
  String get postThreadEmptyReplies => 'No replies yet.';

  @override
  String get postThreadReadMoreReplies => 'Read more replies';

  @override
  String get postThreadShowMoreReplies => 'Show more replies';

  @override
  String get postThreadContinueThread => 'Continue thread';

  @override
  String get postThreadReplyAction => 'Reply';

  @override
  String get postCommentAction => 'Comment';

  @override
  String postThreadReplyToAuthor(String author) {
    return 'Reply to $author';
  }

  @override
  String postCommentOnAuthor(String author) {
    return 'Comment on $author';
  }

  @override
  String postThreadShowMoreRepliesForAuthor(String author) {
    return 'Show more replies to $author';
  }

  @override
  String postThreadContinueThreadFromAuthor(String author) {
    return 'Continue thread from $author';
  }

  @override
  String get postCommentsSortOldest => 'Oldest';

  @override
  String get postCommentsSortOldestDescription => 'Conversation order';

  @override
  String get postCommentsSortNewest => 'Newest';

  @override
  String get postCommentsSortNewestDescription => 'Most recent on top';

  @override
  String get postCommentsSortFollows => 'Follows';

  @override
  String get postCommentsSortFollowsDescription => 'People you follow first';

  @override
  String get postCommentsViewReplies => 'View replies';

  @override
  String postCommentsViewReplyCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count replies',
      one: '1 reply',
    );
    return 'Show $_temp0';
  }

  @override
  String get postCommentsLoadMoreReplies => 'Load more replies';

  @override
  String get postCommentsHideReplies => 'Hide replies';

  @override
  String get postCommentsFocusNotFound => 'That reply isn\'t available yet.';

  @override
  String get postCommentsFocusMismatchedRoot =>
      'That reply belongs to a different post.';

  @override
  String get postComposeAction => 'New post';

  @override
  String get postComposeTitle => 'New post';

  @override
  String get postTypeRegularLabel => 'Regular post';

  @override
  String get postTypeRegularDescription =>
      'Share a quick update, thought or question.';

  @override
  String get postTypeProjectLabel => 'Project post';

  @override
  String get postTypeProjectDescription =>
      'Add photos and structured project details.';

  @override
  String get projectComposerTitle => 'Project post';

  @override
  String get projectComposerNextAction => 'Next';

  @override
  String get projectComposerRequiredLabel => 'required';

  @override
  String get projectComposerDetailsPrompt =>
      'Fill in the details about your project';

  @override
  String get projectComposerOptionalDetailsPrompt =>
      'This information is optional but will help others find your project';

  @override
  String get projectComposerProjectTitleLabel => 'Project title';

  @override
  String get projectComposerProjectTitleHint => 'Add a short project title';

  @override
  String get projectComposerDescriptionLabel => 'Project description';

  @override
  String get projectComposerDescriptionHint =>
      'Tell everyone about your project';

  @override
  String get projectComposerCraftTypeLabel => 'Craft type';

  @override
  String get projectComposerStatusLabel => 'Status';

  @override
  String get projectComposerMaterialsLabel => 'Materials';

  @override
  String get projectComposerMaterialsAddHint => 'Add material';

  @override
  String get projectComposerMaterialsAddAction => 'Add';

  @override
  String projectComposerMaterialsMaxLengthError(int max) {
    return 'Keep each material to $max characters or fewer.';
  }

  @override
  String get projectComposerFieldDisabledLabel => 'Disabled';

  @override
  String projectComposerMultiSelectMaxSelectedError(int maxSelected) {
    return 'You can choose up to $maxSelected.';
  }

  @override
  String get projectComposerColoursLabel => 'Colours';

  @override
  String get projectComposerColoursSearchHint => 'Search colours';

  @override
  String get projectComposerDesignTagsLabel => 'Design tags';

  @override
  String get projectComposerDesignTagsSearchHint => 'Search design tags';

  @override
  String get projectComposerAddPatternAction => 'Add pattern';

  @override
  String get projectComposerPatternSectionLabel => 'Pattern';

  @override
  String get projectComposerPatternInfoSectionLabel => 'Pattern info';

  @override
  String get projectComposerMoreDetailsLabel => 'More project details';

  @override
  String get projectComposerSelectCraftTypeEmptyState => 'Select Craft Type';

  @override
  String get projectComposerSewingProjectTypeLabel => 'Project type';

  @override
  String get projectComposerProjectSubtypeLabel => 'Project subtype';

  @override
  String get projectComposerSizeMadeLabel => 'Size made';

  @override
  String get projectComposerSizeMadeHint =>
      'e.g. Medium or custom measurements';

  @override
  String get projectComposerFitNotesLabel => 'Fit notes';

  @override
  String get projectComposerFitNotesHint => 'Add fit notes';

  @override
  String get projectComposerKnittingProjectTypeLabel => 'Project type';

  @override
  String get projectComposerCrochetProjectTypeLabel => 'Project type';

  @override
  String get projectComposerQuiltingProjectTypeLabel => 'Project type';

  @override
  String get projectComposerYarnWeightLabel => 'Yarn weight';

  @override
  String get projectComposerNeedleSizeLabel => 'Needle size';

  @override
  String get projectComposerHookSizeLabel => 'Hook size';

  @override
  String get projectComposerGaugeStitchesLabel => 'Gauge stitches';

  @override
  String get projectComposerGaugeStitchesHint => 'Stitches';

  @override
  String get projectComposerGaugeRowsLabel => 'Gauge rows';

  @override
  String get projectComposerGaugeRowsHint => 'Rows';

  @override
  String get projectComposerGaugeMeasurementLabel => 'Gauge measurement';

  @override
  String get projectComposerGaugeMeasurementHint => 'Measurement';

  @override
  String get projectComposerGaugeUnitLabel => 'Gauge unit';

  @override
  String get projectComposerFinishedSizeLabel => 'Finished size';

  @override
  String get projectComposerFinishedSizeHint => 'Add finished size';

  @override
  String get projectComposerSizeLabel => 'Size';

  @override
  String get projectComposerPiecingTechniqueLabel => 'Piecing technique';

  @override
  String get projectComposerQuiltingMethodLabel => 'Quilting method';

  @override
  String get projectComposerBodyRequiredError => 'Add body text.';

  @override
  String get projectComposerCraftRequiredError => 'Choose a craft type.';

  @override
  String get projectComposerPhotoRequiredError => 'Add at least one photo.';

  @override
  String get projectComposerGaugeInvalidError =>
      'Complete the gauge or clear it.';

  @override
  String get projectComposerPatternNameLabel => 'Pattern tag or name';

  @override
  String get projectComposerPatternNameHint => 'Add pattern name';

  @override
  String get projectComposerPatternUrlLabel => 'Link';

  @override
  String get projectComposerPatternUrlHint => 'https://example.com/pattern';

  @override
  String get projectComposerPatternDifficultyLabel => 'Difficulty';

  @override
  String get projectComposerPatternDesignerLabel => 'Designer';

  @override
  String get projectComposerPatternDesignerHint => 'Add pattern designer';

  @override
  String get projectComposerPatternPublisherLabel => 'Publisher';

  @override
  String get projectComposerPatternPublisherHint => 'Add pattern publisher';

  @override
  String get postComposeHint => 'What are you making?';

  @override
  String get postComposeBodyHint =>
      'Pattern, fabric, what went right, what didn\'t...';

  @override
  String get postComposeReplyTitle => 'Reply';

  @override
  String get postComposeReplyHint => 'Write your reply';

  @override
  String get postComposeSubmit => 'Post';

  @override
  String get postComposeReplySubmit => 'Reply';

  @override
  String get postComposeTooLong => 'Posts must be 2000 characters or fewer';

  @override
  String get postCreateSuccess => 'Posted.';

  @override
  String get postCreateError => 'Couldn\'t post.';

  @override
  String get postComposeDiscardTitle => 'Discard draft?';

  @override
  String get postComposeDiscardMessage => 'Your draft won\'t be saved.';

  @override
  String get postComposeDiscardConfirm => 'Discard';

  @override
  String get postComposeDiscardCancel => 'Keep editing';

  @override
  String postComposeImageLimitError(int maxImages) {
    return 'You can add up to $maxImages images';
  }

  @override
  String postComposeUnsupportedImagesError(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count unsupported images',
      one: 'Unsupported image type',
    );
    return '$_temp0';
  }

  @override
  String get postComposeImagePickerError => 'Could not open image picker';

  @override
  String get postComposeMissingAltTitle => 'Some images do not have alt text';

  @override
  String get postComposeMissingAltMessage => 'Do you wish to post anyway?';

  @override
  String get postComposeMissingAltConfirm => 'Post anyway';

  @override
  String get postComposeMissingAltCancel => 'Go back';

  @override
  String get postComposePhotosTitle => 'Photos';

  @override
  String get postComposeNoImagesDescribed => '0 described';

  @override
  String postComposeImagesDescribed(int describedCount, int imageCount) {
    return '$describedCount / $imageCount described';
  }

  @override
  String postComposePhotosLimitHelper(int maxImages) {
    return 'Up to $maxImages photos';
  }

  @override
  String postComposePhotosReorderHelper(int imageCount, int maxImages) {
    return '$imageCount/$maxImages · drag to reorder · first is the cover';
  }

  @override
  String get postComposeMoveImageUp => 'Move image up';

  @override
  String get postComposeMoveImageDown => 'Move image down';

  @override
  String get postComposeRemoveImage => 'Remove image';

  @override
  String get postComposeDragToReorder => 'Drag to reorder';

  @override
  String get postComposeAltTextLabel => 'ALT TEXT';

  @override
  String get postComposeAltTextHint =>
      'Describe the image for someone who cannot see it, including the craft, materials, colors, and important details.';

  @override
  String get postComposeImageDescribed => 'Described';

  @override
  String get postComposeImageNeedsAltText => 'Help screen readers';

  @override
  String get postComposeAddPhoto => 'Add a photo';

  @override
  String get postComposeAddAnotherPhoto => 'Add another photo';

  @override
  String postComposePhotosRemaining(int remainingCount) {
    return 'Up to $remainingCount more';
  }

  @override
  String get postComposeReadingImage => 'Reading image';

  @override
  String get postComposePreparingImage => 'Preparing image';

  @override
  String get postComposeUploadingImage => 'Uploading image';

  @override
  String get postComposeUploadedImage => 'Uploaded';

  @override
  String get postComposeImageFailed => 'Failed';

  @override
  String get postComposeProcessingImage => 'Processing';

  @override
  String postComposeUploadingProgress(int percent) {
    return 'Uploading $percent%';
  }

  @override
  String get postLikeAction => 'Like';

  @override
  String get postUnlikeAction => 'Unlike';

  @override
  String get postLikeError => 'Couldn\'t update like.';

  @override
  String get postReplyAction => 'Reply';

  @override
  String get postRepostAction => 'Repost';

  @override
  String get postUnrepostAction => 'Unrepost';

  @override
  String get postQuoteAction => 'Quote';

  @override
  String get postShareAction => 'Share';

  @override
  String get postImportedFromInstagram => 'Imported from Instagram';

  @override
  String postRepostedBy(String name) {
    return 'Reposted by $name';
  }

  @override
  String get postQuoteHidden => 'Quoted post hidden';

  @override
  String get postQuoteUnavailable => 'Quoted post unavailable';

  @override
  String get postDeleteAction => 'Delete post';

  @override
  String get postReportAction => 'Report post';

  @override
  String get postPinAction => 'Pin post';

  @override
  String get postUnpinAction => 'Unpin post';

  @override
  String get postPinnedAnnotation => 'Pinned post';

  @override
  String get postPinSuccess => 'Post pinned';

  @override
  String get postUnpinSuccess => 'Post unpinned';

  @override
  String get postPinError => 'Couldn’t pin post. Try again.';

  @override
  String get postUnpinError => 'Couldn’t unpin post. Try again.';

  @override
  String get postMoreActions => 'More actions';

  @override
  String get commentDeleteAction => 'Delete comment';

  @override
  String get replyDeleteAction => 'Delete reply';

  @override
  String get postDeleteTitle => 'Delete post?';

  @override
  String get commentDeleteTitle => 'Delete comment?';

  @override
  String get replyDeleteTitle => 'Delete reply?';

  @override
  String get postDeleteMessage => 'This removes the post from CraftSky.';

  @override
  String get commentDeleteMessage => 'This removes the comment from CraftSky.';

  @override
  String get replyDeleteMessage => 'This removes the reply from CraftSky.';

  @override
  String get postDeleteConfirm => 'Delete';

  @override
  String get postDeleteSuccess => 'Post deleted.';

  @override
  String get postDeleteError => 'Couldn\'t delete post.';

  @override
  String get profileFollowComingSoon => 'Follow coming soon.';

  @override
  String get profileFollowToggleError => 'Could not update follow state.';

  @override
  String get profileShareComingSoon => 'Share coming soon.';

  @override
  String get profileReportAction => 'Report profile';

  @override
  String get moderationWarningPost =>
      'This post may not follow CraftSky community guidelines.';

  @override
  String get moderationWarningProfile =>
      'This profile may not follow CraftSky community guidelines.';

  @override
  String get moderationWarningAuthor =>
      'This author may not follow CraftSky community guidelines.';

  @override
  String get reportSubmit => 'Submit';

  @override
  String get reportSubmitting => 'Submitting…';

  @override
  String get reportSubmitSuccess => 'Thanks — your report was submitted.';

  @override
  String get reportSubmitError => 'Couldn\'t submit report. Please try again.';

  @override
  String get reportDetailsLabel => 'Details';

  @override
  String get reportDetailsTooLong =>
      'Details must be 1000 characters or fewer.';

  @override
  String get reportReasonTitle => 'Reason';

  @override
  String get reportReasonHarassment => 'Harassment';

  @override
  String get reportReasonHate => 'Hate';

  @override
  String get reportReasonSpam => 'Spam';

  @override
  String get reportReasonMisleading => 'Misleading';

  @override
  String get reportReasonSuspectedAiGenerated => 'Suspected AI-generated';

  @override
  String get reportReasonAdultOrGraphic => 'Adult or graphic';

  @override
  String get reportReasonImpersonation => 'Impersonation';

  @override
  String get reportReasonOffTopic => 'Off-topic';

  @override
  String get reportReasonIntellectualProperty => 'Intellectual property';

  @override
  String get reportReasonOther => 'Other';

  @override
  String get editProfileTitle => 'Edit profile';

  @override
  String get editProfileSave => 'Save';

  @override
  String get editProfileCancel => 'Cancel';

  @override
  String get editProfileDisplayNameLabel => 'Display name';

  @override
  String get editProfileDisplayNameHint =>
      'How your name appears on your profile';

  @override
  String get editProfileBioLabel => 'Bio';

  @override
  String get editProfileBioHint => 'Tell people about yourself';

  @override
  String get editProfileDisplayNameTooLong =>
      'Display name must be 64 characters or fewer';

  @override
  String get editProfileBioTooLong => 'Bio must be 256 characters or fewer';

  @override
  String get editProfileCraftsLabel => 'Crafts';

  @override
  String get editProfileCraftsHelper => 'Pick the crafts you make';

  @override
  String get editProfileChangeAvatar => 'Change avatar';

  @override
  String get editProfileChangeCover => 'Change cover';

  @override
  String get editProfilePhotoUploadError => 'Couldn\'t upload that photo.';

  @override
  String get editProfileSaveError => 'Couldn\'t save your profile.';

  @override
  String get editProfileBusinessSaveError =>
      'Couldn\'t save your business details.';

  @override
  String get editProfileBothSaveError =>
      'Couldn\'t save your profile or business details.';

  @override
  String get editProfileBusinessConflictError =>
      'Your business details changed elsewhere. Reload before saving them again.';

  @override
  String get editProfileBusinessHeading => 'Business details';

  @override
  String get editProfileBusinessHelper =>
      'These details appear on your public business profile.';

  @override
  String get editProfileBusinessTypesLabel => 'Business types';

  @override
  String get editProfileBusinessTypesHelper => 'Choose up to 5.';

  @override
  String get editProfileBusinessTypesLimit =>
      'Choose no more than 5 business types.';

  @override
  String get editProfileBusinessOfferingsLabel => 'Offerings';

  @override
  String get editProfileBusinessOfferingsHelper => 'Choose up to 10.';

  @override
  String get editProfileBusinessOfferingsLimit =>
      'Choose no more than 10 offerings.';

  @override
  String get editProfileBusinessTaglineLabel => 'Tagline';

  @override
  String get editProfileBusinessTaglineTooLong =>
      'Tagline must be 100 characters or fewer.';

  @override
  String get editProfileBusinessHoursLabel => 'Hours';

  @override
  String get editProfileBusinessHoursTooLong =>
      'Hours must be 300 characters or fewer.';

  @override
  String get editProfileBusinessServiceAreaLabel => 'Service area';

  @override
  String get editProfileBusinessServiceAreaTooLong =>
      'Service area must be 200 characters or fewer.';

  @override
  String get editProfileBusinessCountryLabel => 'Country code';

  @override
  String get editProfileBusinessCountryInvalid =>
      'Enter a valid two-letter country code.';

  @override
  String get editProfileBusinessLocalityLabel => 'Town or locality';

  @override
  String get editProfileBusinessLocalityTooLong =>
      'Town or locality must be 100 characters or fewer.';

  @override
  String get editProfileBusinessActionLabel => 'Primary action';

  @override
  String get editProfileBusinessActionNone => 'No primary action';

  @override
  String get editProfileBusinessActionDestinationLabel => 'Action destination';

  @override
  String get editProfileBusinessActionDestinationInvalid =>
      'Enter a valid HTTPS or email destination.';

  @override
  String get editProfileDiscardTitle => 'Discard changes?';

  @override
  String get editProfileDiscardMessage => 'Your edits won\'t be saved.';

  @override
  String get editProfileDiscardConfirm => 'Discard';

  @override
  String get editProfileDiscardCancel => 'Keep editing';

  @override
  String get profileCustomisationTitle => 'Customisation';

  @override
  String get profileCustomisationPreview => 'Preview';

  @override
  String get profileCustomisationColour => 'Colour';

  @override
  String get profileCustomisationBorder => 'Profile border';

  @override
  String get profileCustomisationBackground => 'Profile background';

  @override
  String get profileCustomisationSave => 'Save';

  @override
  String get profileCustomisationSaved => 'Profile customisation saved';

  @override
  String get profileCustomisationSaveError =>
      'Couldn\'t save your profile customisation.';

  @override
  String get profileCustomisationLoadError =>
      'Couldn\'t load your profile customisation.';

  @override
  String get profileCustomisationRetry => 'Retry';

  @override
  String get profileCustomisationDiscardTitle =>
      'Discard customisation changes?';

  @override
  String get profileCustomisationDiscardMessage =>
      'Your customisation changes won\'t be saved.';

  @override
  String get profileCustomisationNone => 'None';

  @override
  String get profileCustomisationColourCobalt => 'Cobalt';

  @override
  String get profileCustomisationColourOrchid => 'Orchid';

  @override
  String get profileCustomisationColourRose => 'Rose';

  @override
  String get profileCustomisationColourAmber => 'Amber';

  @override
  String get profileCustomisationColourGreen => 'Green';

  @override
  String get profileCustomisationColourTeal => 'Teal';

  @override
  String get profileCustomisationColourInk => 'Ink';

  @override
  String get profileCustomisationBorderThin => 'Thin';

  @override
  String get profileCustomisationBorderMedium => 'Medium';

  @override
  String get profileCustomisationBorderThick => 'Thick';

  @override
  String get profileCustomisationBackgroundDither => 'Dither';

  @override
  String get profileCustomisationBackgroundGrid => 'Grid';

  @override
  String get profileCustomisationBackgroundCrossStitch => 'Cross stitch';

  @override
  String get profileCustomisationBackgroundScallops => 'Scallops';

  @override
  String get profileCustomisationBackgroundDiagonalWeave => 'Diagonal weave';

  @override
  String get profileCustomisationBackgroundCrosshatch => 'Crosshatch';

  @override
  String get craftSewing => 'Sewing';

  @override
  String get craftQuilting => 'Quilting';

  @override
  String get craftKnitting => 'Knitting';

  @override
  String get craftCrochet => 'Crochet';

  @override
  String get craftEmbroidery => 'Embroidery';

  @override
  String get craftCrossStitch => 'Cross-stitch';

  @override
  String get craftWeaving => 'Weaving';

  @override
  String get craftSpinning => 'Spinning';

  @override
  String get craftFelting => 'Felting';

  @override
  String get craftMacrame => 'Macramé';

  @override
  String get craftPottery => 'Pottery';

  @override
  String get craftWoodworking => 'Woodworking';

  @override
  String get craftLeatherwork => 'Leatherwork';

  @override
  String get craftJewellery => 'Jewellery';

  @override
  String get craftBookbinding => 'Bookbinding';

  @override
  String get craftCalligraphy => 'Calligraphy';

  @override
  String get craftPrintmaking => 'Printmaking';

  @override
  String get craftPapercraft => 'Paper craft';

  @override
  String get craftPainting => 'Painting';

  @override
  String get craftDrawing => 'Drawing';

  @override
  String get craftCandleMaking => 'Candle making';

  @override
  String get craftSoapMaking => 'Soap making';

  @override
  String get searchTitle => 'Search';

  @override
  String get searchHint => 'Search hashtags, people or projects...';

  @override
  String get searchCancelAction => 'Cancel';

  @override
  String get searchClearAction => 'Clear search text';

  @override
  String get searchRecentHeading => 'Recent searches';

  @override
  String get searchDeleteRecentAction => 'Delete recent search';

  @override
  String get searchTrendingHashtagsHeading => 'Trending hashtags';

  @override
  String get searchProfilesHeading => 'Profiles';

  @override
  String get searchHashtagsHeading => 'Hashtags';

  @override
  String get searchViewAllAction => 'View all';

  @override
  String get searchTabPosts => 'Posts';

  @override
  String get searchTabProjects => 'Projects';

  @override
  String get searchTabProfiles => 'Profiles';

  @override
  String get searchTabTags => 'Tags';

  @override
  String get searchEmptyPosts => 'No posts found.';

  @override
  String get searchEmptyProjects => 'No projects found.';

  @override
  String get searchEmptyProfiles => 'No profiles found.';

  @override
  String get searchEmptyTags => 'No tags found.';

  @override
  String get searchLoadError => 'Search didn\'t load.';

  @override
  String get searchRecentSaveError => 'Couldn\'t save recent search.';

  @override
  String get searchRecentDeleteError => 'Couldn\'t delete recent search.';

  @override
  String searchTagPostCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count posts',
      one: '1 post',
    );
    return '$_temp0';
  }

  @override
  String searchProfileCraftSubtitle(String name, String crafts) {
    return '$name • $crafts';
  }

  @override
  String get searchSortNewest => 'Newest';

  @override
  String get searchSortNewestDescription => 'Show the newest items first.';

  @override
  String get searchSortPopular => 'Popular';

  @override
  String get searchSortPopularDescription =>
      'Show the most popular items first.';

  @override
  String tagSearchTitle(String tag) {
    return '#$tag';
  }

  @override
  String get tagSearchEmpty => 'No posts found for this tag.';

  @override
  String get projectsTitle => 'Projects';

  @override
  String get projectsFilterAction => 'Filters';

  @override
  String projectsFiltersTitle(String craft) {
    return 'Filter $craft projects';
  }

  @override
  String projectsCraftContext(String craft) {
    return 'Browsing $craft';
  }

  @override
  String get projectsFilterProjectType => 'Project type';

  @override
  String get projectsFilterDifficulty => 'Pattern difficulty';

  @override
  String get projectsFilterColor => 'Color';

  @override
  String get projectsFilterDesignTag => 'Design tag';

  @override
  String get projectsFilterMaterial => 'Material';

  @override
  String get projectsFilterProjectTag => 'Project tag';

  @override
  String get projectsFreeTextHint => 'Add a value';

  @override
  String get projectsAddFilterValueAction => 'Add';

  @override
  String get projectsApplyFiltersAction => 'Apply filters';

  @override
  String get projectsClearFiltersAction => 'Clear all';

  @override
  String get projectsEmpty => 'No projects found.';

  @override
  String get projectsLoadError => 'Projects didn\'t load.';

  @override
  String get instagramMigrationTitle => 'Find people from Instagram';

  @override
  String get instagramMigrationSettingsSubtitle =>
      'Verify your account or privately import handles.';

  @override
  String get instagramMigrationLoadError =>
      'Instagram migration data didn\'t load.';

  @override
  String get instagramMigrationNoActiveAccount =>
      'Sign in to an account to use Instagram migration.';

  @override
  String get instagramVerificationTitle => 'Verify your Instagram account';

  @override
  String get instagramVerificationDescription =>
      'Send a one-time challenge to CraftSky\'s official Instagram account. You will confirm the username here before it is verified.';

  @override
  String get instagramVerificationUnavailable =>
      'Instagram verification is unavailable right now.';

  @override
  String get instagramVerificationUnavailableImports =>
      'Imports become available after Instagram verification is configured and your account is verified.';

  @override
  String get instagramVerificationRequiredForImport =>
      'Complete verification to import the accounts you follow.';

  @override
  String get instagramVerificationStart => 'Create verification challenge';

  @override
  String get instagramVerificationSendChallenge =>
      'Send this exact one-time challenge in an Instagram DM:';

  @override
  String get instagramVerificationChallengeLabel =>
      'Instagram verification challenge';

  @override
  String get instagramVerificationProcessing => 'Checking your message…';

  @override
  String get instagramCopyChallenge => 'Copy challenge';

  @override
  String get instagramChallengeCopied => 'Challenge copied';

  @override
  String get instagramOpenDm => 'Open Instagram DM';

  @override
  String get instagramCancelVerification => 'Cancel verification';

  @override
  String instagramVerificationCandidate(String username) {
    return 'Account: @$username';
  }

  @override
  String get instagramUnknownUsername => 'unknown';

  @override
  String get instagramVerificationCandidateWarning =>
      'Confirm only if this is your Instagram username.';

  @override
  String get instagramDiscoverableLabel =>
      'Let others find me by my Instagram username';

  @override
  String get instagramDiscoverableDescription =>
      'When enabled, eligible CraftSky members who imported your Instagram username may see a private suggestion to follow you.';

  @override
  String get instagramDiscoverableAllow => 'Allow discovery';

  @override
  String get instagramDiscoverablePrivate => 'Keep private';

  @override
  String get instagramDiscoverablePrivateDescription =>
      'Your Instagram account remains verified, but CraftSky will not match it with people who imported your username.';

  @override
  String get instagramVerificationConfirm => 'Confirm this account';

  @override
  String get instagramVerificationConfirmed => 'Instagram account confirmed.';

  @override
  String get instagramVerificationExpired =>
      'This verification challenge expired.';

  @override
  String get instagramVerificationCancelled =>
      'This verification challenge is no longer active.';

  @override
  String get instagramVerificationRejected =>
      'Instagram could not verify this message. Create a new challenge to try again.';

  @override
  String get instagramVerificationProfileUnavailable =>
      'Instagram profile lookup is temporarily unavailable. Create a new challenge to try again.';

  @override
  String get instagramVerificationProfileInvalid =>
      'Instagram returned an invalid profile result. Create a new challenge to try again.';

  @override
  String get instagramVerificationMembershipInactive =>
      'Your CraftSky membership is inactive. Restore membership before trying again.';

  @override
  String get instagramVerificationConflict =>
      'This Instagram account cannot be verified automatically. Your existing verified account remains unchanged.';

  @override
  String get instagramActionError =>
      'That Instagram action didn\'t complete. Try again.';

  @override
  String get instagramRetry => 'Try again';

  @override
  String get instagramLoadMore => 'Load more';

  @override
  String get instagramAccountTitle => 'Instagram account';

  @override
  String instagramLinkedAs(String username) {
    return 'Verified as @$username';
  }

  @override
  String get instagramConflictPending =>
      'There is a private account conflict to resolve. No ownership was transferred.';

  @override
  String get instagramReactivateAccount => 'Reactivate Instagram account';

  @override
  String get instagramReactivateAccountDisclosure =>
      'Reactivation keeps discovery off until you choose to turn it on again.';

  @override
  String get instagramRevokeAccount => 'Revoke Instagram verification';

  @override
  String get instagramRevokeAccountConfirmTitle =>
      'Revoke Instagram verification?';

  @override
  String get instagramRevokeAccountConfirmMessage =>
      'This removes your Instagram verification and deletes your imported handles. Existing CraftSky follows will not be affected.';

  @override
  String get instagramImportTitle => 'Import accounts you follow';

  @override
  String get instagramImportManual => 'Enter handles';

  @override
  String get instagramImportManualDescription =>
      'Enter the Instagram handles of accounts you follow, one per line.';

  @override
  String get instagramImportJson => 'Instagram export';

  @override
  String get instagramImportJsonDescription =>
      'Choose an Instagram export containing Accounts you follow. CraftSky processes it on this device and uploads only those usernames. If you select an all-information ZIP, everything else stays on your device.';

  @override
  String get instagramImportHandles => 'Instagram handles';

  @override
  String get instagramImportHandlesHint => 'One handle per line';

  @override
  String get instagramImportManualAction => 'Import handles';

  @override
  String get instagramImportSelectJson => 'Select Instagram export';

  @override
  String get instagramImportFilePickerError =>
      'The Instagram export couldn\'t be opened on this device.';

  @override
  String get instagramImportInvalidJson => 'This file is not valid JSON.';

  @override
  String get instagramImportUnsupportedShape =>
      'This is not a supported Instagram accounts-followed export. Choose an export containing Accounts you follow.';

  @override
  String get instagramImportUnsupportedFormat =>
      'This Instagram export uses a format CraftSky can\'t read.';

  @override
  String get instagramImportInvalidArchive =>
      'This Instagram ZIP is incomplete or damaged. Download a new export and try again.';

  @override
  String get instagramImportArchiveTooLarge =>
      'This Instagram ZIP contains too many files to process safely.';

  @override
  String get instagramImportFileTooLarge =>
      'The accounts-followed data is larger than 20 MiB.';

  @override
  String get instagramImportTooManyEntries =>
      'This import contains more than 10,000 unique handles.';

  @override
  String instagramImportFollowingPreviewCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count accounts you follow ready',
      one: '1 account you follow ready',
    );
    return '$_temp0';
  }

  @override
  String instagramImportIgnoredCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count unsupported entries ignored',
      one: '1 unsupported entry ignored',
    );
    return '$_temp0';
  }

  @override
  String instagramImportDuplicateCount(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count duplicates removed',
      one: '1 duplicate removed',
    );
    return '$_temp0';
  }

  @override
  String get instagramImportUploadSuccess => 'Instagram import created';

  @override
  String get instagramImportUploadError =>
      'Instagram import wasn\'t created. Try again.';

  @override
  String get instagramImportsTitle => 'Your imports';

  @override
  String get instagramImportsLoadError =>
      'Your Instagram imports didn\'t load.';

  @override
  String get instagramImportsEmpty => 'No Instagram imports yet.';

  @override
  String get instagramImportManualSource => 'Manual handles';

  @override
  String get instagramImportJsonSource => 'Instagram export';

  @override
  String get instagramImportUnknownSource => 'Instagram import';

  @override
  String instagramImportCounts(int followingCount) {
    String _temp0 = intl.Intl.pluralLogic(
      followingCount,
      locale: localeName,
      other: '$followingCount accounts imported',
      one: '1 account imported',
    );
    return '$_temp0';
  }

  @override
  String get instagramImportReactivationDisclosure =>
      'This import paused when your CraftSky membership changed. Reactivate it to resume matching.';

  @override
  String get instagramImportReactivate => 'Reactivate import';

  @override
  String get instagramImportDelete => 'Delete import';

  @override
  String get instagramImportSuggestionDisclosure =>
      'Importing creates private suggestions only. You choose whether to follow each account.';

  @override
  String get instagramSuggestionsTitle => 'Possible CraftSky accounts';

  @override
  String get instagramSuggestionsDescription =>
      'Your imports find possible CraftSky accounts privately. Nobody is followed until you choose Follow.';

  @override
  String get instagramSuggestionsLoadError =>
      'Possible CraftSky accounts didn\'t load.';

  @override
  String get instagramSuggestionsEmpty => 'No possible CraftSky accounts yet.';

  @override
  String get instagramSuggestionFollow => 'Follow';

  @override
  String get instagramSuggestionDismiss => 'Dismiss';

  @override
  String get instagramSuggestionsActionError =>
      'That suggestion action didn\'t complete. Try again.';

  @override
  String get savedPostSaveAction => 'Save post';

  @override
  String get savedPostUnsaveAction => 'Remove from saved posts';

  @override
  String get savedPostUnsaveError =>
      'This post couldn\'t be removed. Try again.';

  @override
  String get savedPostMoveTitle => 'Move saved post';

  @override
  String get savedPostMoveAction => 'Move';

  @override
  String get savedPostRowUnsaveAction => 'Unsave';

  @override
  String get savedPostNoFolder => 'No folder';

  @override
  String get savedPostFolderSelectionLabel => 'Folder';

  @override
  String get savedPostConfirmError =>
      'That change couldn\'t be saved. Try again.';

  @override
  String get savedPostFoldersLoadError => 'Folders couldn\'t load.';

  @override
  String get savedPostLoadMoreFolders => 'Load more folders';

  @override
  String get savedPostNewFolder => 'New folder';

  @override
  String get savedPostFolderNameHint => 'Folder name';

  @override
  String get savedPostCreateFolderAction => 'Create folder';

  @override
  String get savedPostCreateFolderError =>
      'That folder couldn\'t be created. Try again.';

  @override
  String get savedPostsTitle => 'Saved posts';

  @override
  String get savedPostsFoldersHeading => 'Folders';

  @override
  String get savedPostsUnfiledHeading => 'Unfiled';

  @override
  String get savedPostsEmpty => 'Nothing saved yet';

  @override
  String get savedPostsSortOldest => 'Oldest';

  @override
  String get savedPostsSortNewestDescription => 'Most recently saved first';

  @override
  String get savedPostsSortOldestDescription => 'Earliest saved first';

  @override
  String get savedPostsLoadError => 'Saved posts couldn\'t load.';

  @override
  String get savedPostsLoadMore => 'Load more';

  @override
  String get savedPostFolderActions => 'Folder actions';

  @override
  String get savedPostRowActions => 'Saved post actions';

  @override
  String get savedPostRenameFolder => 'Rename folder';

  @override
  String get savedPostDeleteFolder => 'Delete folder';

  @override
  String get savedPostDeleteFolderBody =>
      'What should happen to the posts in this folder?';

  @override
  String get savedPostKeepSaves => 'Keep saved posts';

  @override
  String get savedPostDeleteSaves => 'Delete saved posts';

  @override
  String get settingsLanguages => 'Languages';

  @override
  String get languagesTitle => 'Languages';

  @override
  String get appLanguageTitle => 'App language';

  @override
  String get appLanguageDescription =>
      'Select which language to use for the app\'s user interface.';

  @override
  String get appLanguageEnglish => 'English';

  @override
  String get appLanguageMoreComing => 'More app languages are coming.';

  @override
  String get primaryLanguageTitle => 'Primary language';

  @override
  String get primaryLanguageDescription =>
      'Select the default language used when you create a post.';

  @override
  String get contentLanguagesTitle => 'Content languages';

  @override
  String get contentLanguagesDescription =>
      'Select which languages you want posts in your feeds and discovery results to include. If none are selected, all languages will be shown.';

  @override
  String get languageSearchHint => 'Search languages';

  @override
  String get languageAddMore => 'Add more languages…';

  @override
  String get languageCancel => 'Cancel';

  @override
  String get languageDone => 'Done';

  @override
  String get languageSaveError => 'That change could not be saved. Try again.';

  @override
  String get postLanguagesSemantics => 'Post languages';

  @override
  String get postLanguageAdd => 'Add language';

  @override
  String get postLanguageLimit => 'Up to three languages';

  @override
  String get postLanguageDialogTitle => 'Add post language';

  @override
  String get postLanguageRetryLoading => 'Retry loading languages';

  @override
  String get scheduledPostsTitle => 'Scheduled posts';

  @override
  String get scheduledPostsEmpty => 'No scheduled posts';

  @override
  String get scheduledPostsDeleteTitle => 'Delete scheduled post?';

  @override
  String get scheduledPostsDeleteMessage =>
      'This removes the unpublished post and its private media.';

  @override
  String get scheduledPostsDeleteAction => 'Delete';

  @override
  String get scheduledPostDeleteError =>
      'Could not delete the scheduled post. Try again.';

  @override
  String get scheduledPostsKindProject => 'Project';

  @override
  String get scheduledPostsKindStandard => 'Standard';

  @override
  String scheduledPostsRowDateTime(String kind, String date, String time) {
    return '$kind · $date, $time';
  }

  @override
  String get scheduledPostsStatusScheduled => 'Scheduled';

  @override
  String get scheduledPostsStatusPublishing => 'Publishing';

  @override
  String get scheduledPostsStatusRetrying => 'Retrying';

  @override
  String get scheduledPostsStatusNeedsAttention => 'Needs attention';

  @override
  String get scheduledPostsPublishingLocked =>
      'Editing is unavailable while publishing';

  @override
  String get scheduledPostsPublishingLockSemantics => 'Publishing lock';

  @override
  String get scheduledPostsEditTooltip => 'Edit scheduled post';

  @override
  String get scheduledPostsDeleteTooltip => 'Delete scheduled post';

  @override
  String get scheduledPostsThumbnailSemantics => 'Scheduled post image';

  @override
  String scheduledPostsDeletedOn(String date) {
    return 'Deleted on $date';
  }

  @override
  String get scheduledPostsLoadError => 'Could not load scheduled posts';

  @override
  String get scheduledPostsRetryAction => 'Try again';

  @override
  String get scheduledPostWhenTitle => 'When';

  @override
  String get scheduledPostNow => 'Now';

  @override
  String get scheduledPostLater => 'Schedule for later';

  @override
  String get scheduledPostTimeRangeError =>
      'Choose a whole-minute time from 5 minutes through 28 days from now';

  @override
  String get scheduledPostAction => 'Schedule';

  @override
  String scheduledPostStagingProgress(int current, int total) {
    return 'Preparing image $current of $total';
  }

  @override
  String get scheduledPostCreating => 'Saving scheduled post';

  @override
  String get scheduledPostManageAction => 'Manage scheduled posts';

  @override
  String get scheduledPostCapacityWarning =>
      'You can\'t schedule another post because you already have 3 scheduled posts.';

  @override
  String scheduledPostLocalTime(
    String date,
    String time,
    String zone,
    String offset,
  ) {
    return '$date at $time ($zone, UTC$offset)';
  }

  @override
  String scheduledPostMissedTime(String time) {
    return 'Missed schedule: $time';
  }

  @override
  String get scheduledPostSaved => 'Post scheduled';

  @override
  String get scheduledPostSaveError =>
      'Could not schedule post. Your draft is still here.';

  @override
  String get scheduledPostNowError =>
      'Could not post now. Your draft is still here.';

  @override
  String get scheduledProjectSaved => 'Project scheduled';

  @override
  String get scheduledProjectSaveError =>
      'Could not schedule project. Your draft is still here.';

  @override
  String get scheduledProjectNowError =>
      'Could not post now. Your project is still here.';

  @override
  String get draftsTitle => 'Drafts';

  @override
  String get draftsEmpty => 'No drafts';

  @override
  String get draftsDeleteTitle => 'Delete draft?';

  @override
  String get draftsDeleteMessage =>
      'This removes the draft and its saved images from this device.';

  @override
  String get draftsDeleteAction => 'Delete';

  @override
  String get draftsKindProject => 'Project';

  @override
  String get draftsKindStandard => 'Standard';

  @override
  String draftsRowDateTime(String kind, String date, String time) {
    return '$kind · $date, $time';
  }

  @override
  String get draftsUnavailable => 'Draft unavailable';

  @override
  String get draftsUntitled => 'Untitled draft';

  @override
  String get draftsImageUnavailable => 'Image unavailable';

  @override
  String get draftsLoadError => 'Could not load drafts';

  @override
  String get draftsRetryAction => 'Try again';

  @override
  String get draftsEditTooltip => 'Edit draft';

  @override
  String get draftsDeleteTooltip => 'Delete draft';

  @override
  String get draftsThumbnailSemantics => 'Draft image';

  @override
  String get submissionPublishingPost => 'Publishing your post…';

  @override
  String get submissionSchedulingPost => 'Scheduling your post…';

  @override
  String get draftSaveAction => 'Save draft';

  @override
  String get draftSaveChangesAction => 'Save changes';

  @override
  String get draftSavedMessage => 'Draft saved';

  @override
  String get draftSaveError => 'Could not save draft';

  @override
  String get draftCloseTitle => 'Save your draft?';

  @override
  String get draftCloseMessage =>
      'You can save this work on this device before closing.';

  @override
  String get draftKeepEditingAction => 'Keep editing';

  @override
  String get draftDiscardAction => 'Discard';

  @override
  String get draftDiscardChangesAction => 'Discard changes';

  @override
  String get draftCleanupError =>
      'Your post was submitted, but the local draft could not be removed. You can delete it from Drafts.';

  @override
  String get draftsReplaceImageAction => 'Replace image';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get appearanceTitle => 'Appearance';

  @override
  String get appearanceUseDeviceSetting => 'Use device setting';

  @override
  String get appearanceLight => 'Light';

  @override
  String get appearanceDark => 'Dark';

  @override
  String get settingsSwitchAccount => 'Switch account';

  @override
  String get settingsSectionPreferences => 'Preferences';

  @override
  String get settingsSectionConnections => 'Connections';

  @override
  String get settingsSectionDiscovery => 'Discovery';

  @override
  String get settingsSectionGeneral => 'General';

  @override
  String get settingsSectionBusiness => 'Business';

  @override
  String get settingsBusinessEvents => 'Events';

  @override
  String get settingsBusinessProducts => 'Products';

  @override
  String get settingsNotifications => 'Notifications';

  @override
  String get settingsGrowth => 'Growth';

  @override
  String get growthMetricLabel => 'Followers';

  @override
  String get growthTrendLabel => 'Trend';

  @override
  String get growthScopeCopy => 'Craftsky followers.';

  @override
  String get growthFreshnessCopy => 'Updated daily. Dates are UTC.';

  @override
  String get growthPeriodSevenDays => '7 days';

  @override
  String get growthPeriodThirtyDays => '30 days';

  @override
  String get growthPeriodOneYear => '1 year';

  @override
  String growthLatestCount(String count) {
    return '$count followers';
  }

  @override
  String growthChangeUp(String count) {
    return 'Up $count';
  }

  @override
  String growthChangeDown(String count) {
    return 'Down $count';
  }

  @override
  String get growthNoChange => 'No change';

  @override
  String get growthInsufficientHistory => 'Not enough history';

  @override
  String growthLatestSnapshot(String date) {
    return 'Latest snapshot: $date';
  }

  @override
  String get growthNoHistory => 'No follower history yet';

  @override
  String get growthNoObservationsInPeriod => 'No observations in this period';

  @override
  String growthHistoryAvailableSince(String date) {
    return 'History available since $date';
  }

  @override
  String growthChartRange(String start, String end) {
    return 'From $start to $end';
  }

  @override
  String growthMissingDays(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count days have no observation',
      one: '1 day has no observation',
    );
    return '$_temp0';
  }

  @override
  String get growthLoadError => 'Could not load follower growth.';

  @override
  String get settingsFollowers => 'Followers';

  @override
  String get settingsFollowing => 'Following';

  @override
  String get settingsAccount => 'Account';

  @override
  String get settingsAbout => 'About';

  @override
  String get settingsTerms => 'Terms';

  @override
  String get settingsPrivacyPolicy => 'Privacy policy';

  @override
  String get settingsClearImageCache => 'Clear image cache';

  @override
  String get settingsImageCacheCleared => 'Image cache cleared';

  @override
  String get settingsVersion => 'Version';

  @override
  String get settingsSignOut => 'Sign out';

  @override
  String get accountTitle => 'Account';

  @override
  String get accountTypeTitle => 'Account type';

  @override
  String get accountTypeRegular => 'Regular';

  @override
  String get accountTypeBusiness => 'Business';

  @override
  String get deleteAccountTitle => 'Delete CraftSky account?';

  @override
  String get deleteAccountAction => 'Delete account';

  @override
  String get deleteAccountContinue => 'Continue';

  @override
  String get deleteAccountConfirmTitle => 'Confirm account deletion';

  @override
  String get deleteAccountTypeHandleLabel => 'Type your handle';

  @override
  String deleteAccountConfirmationPrompt(String handle) {
    return 'Type $handle exactly to permanently delete this CraftSky account.';
  }

  @override
  String get accountDeletionAlreadyInProgress =>
      'Your CraftSky account deletion is already in progress. You cannot sign in again until it has finished.';

  @override
  String deleteAccountBoundary(String handle) {
    return 'Deleting $handle permanently removes all your CraftSky data from your PDS and all private data held by CraftSky. It won’t delete your PDS, DID, or wider AT Protocol account.\n\nTo continue, you’ll need to authenticate with your PDS again.';
  }

  @override
  String get linkPreviewLoading => 'Loading link preview';

  @override
  String get linkPreviewPrevious => 'Previous link preview';

  @override
  String get linkPreviewNext => 'Next link preview';

  @override
  String get linkPreviewDismiss => 'Dismiss link previews';

  @override
  String get linkPreviewHidden => 'Link previews hidden';

  @override
  String get linkPreviewUndo => 'Undo';

  @override
  String linkPreviewPosition(int current, int total) {
    return 'Link preview $current of $total';
  }

  @override
  String externalCardOpen(String host) {
    return 'Open link to $host';
  }

  @override
  String externalCardThumbnail(String title) {
    return 'Preview image for $title';
  }

  @override
  String youtubePlayVideo(String title) {
    return 'Play YouTube video: $title';
  }

  @override
  String youtubeVideoPlayer(String title) {
    return 'YouTube video player: $title';
  }

  @override
  String get youtubeConsentTitle => 'Play video from YouTube?';

  @override
  String get youtubeConsentMessage =>
      'Playing this video connects to YouTube. YouTube may receive your IP address and device information.';

  @override
  String get youtubeAllowOnce => 'Allow once';

  @override
  String get youtubeAlwaysAllow => 'Always allow YouTube';

  @override
  String get youtubeOpenExternally => 'Open in YouTube';

  @override
  String get youtubeEnterFullscreen => 'Enter full screen';

  @override
  String get youtubePlaybackUnavailable =>
      'This video can’t be played here. It may be private, unavailable, or restricted from embedded playback.';

  @override
  String get businessProfileLabel => 'Business';

  @override
  String get profileTabProducts => 'Products';

  @override
  String get profileTabUpcomingEvents => 'Upcoming Events';

  @override
  String get businessTypesHeading => 'Business types';

  @override
  String get businessOfferingsHeading => 'Offerings';

  @override
  String get businessLocationHeading => 'Location';

  @override
  String get businessServiceAreaHeading => 'Service area';

  @override
  String get businessHoursHeading => 'Hours';

  @override
  String businessUnknownValue(String value) {
    return 'Other: $value';
  }

  @override
  String businessLocationValue(String locality, String country) {
    return '$locality, $country';
  }

  @override
  String get businessTypeDyer => 'Dyer';

  @override
  String get businessTypeFiberProducer => 'Fiber producer';

  @override
  String get businessTypeFiberProcessor => 'Fiber processor';

  @override
  String get businessTypeYarnShop => 'Yarn shop';

  @override
  String get businessTypeFabricShop => 'Fabric shop';

  @override
  String get businessTypeCraftSupplyShop => 'Craft supply shop';

  @override
  String get businessTypePatternDesigner => 'Pattern designer';

  @override
  String get businessTypeFinishedGoodsMaker => 'Finished goods maker';

  @override
  String get businessTypeToolMaker => 'Tool maker';

  @override
  String get businessTypeTeacher => 'Teacher';

  @override
  String get businessTypeCraftStudio => 'Craft studio';

  @override
  String get businessTypeRepairService => 'Repair service';

  @override
  String get businessTypeTechnicalEditor => 'Technical editor';

  @override
  String get businessTypePhotographer => 'Photographer';

  @override
  String get businessTypePublisher => 'Publisher';

  @override
  String get businessTypeOtherCraftBusiness => 'Other craft business';

  @override
  String get businessOfferingYarn => 'Yarn';

  @override
  String get businessOfferingFiber => 'Fiber';

  @override
  String get businessOfferingFabric => 'Fabric';

  @override
  String get businessOfferingPatterns => 'Patterns';

  @override
  String get businessOfferingKits => 'Kits';

  @override
  String get businessOfferingNotions => 'Notions';

  @override
  String get businessOfferingTools => 'Tools';

  @override
  String get businessOfferingFinishedGoods => 'Finished goods';

  @override
  String get businessOfferingCustomWork => 'Custom work';

  @override
  String get businessOfferingRepairs => 'Repairs';

  @override
  String get businessOfferingClasses => 'Classes';

  @override
  String get businessOfferingStudioHire => 'Studio hire';

  @override
  String get businessOfferingWholesale => 'Wholesale';

  @override
  String get businessOfferingDigitalProducts => 'Digital products';

  @override
  String get businessOfferingTechnicalEditing => 'Technical editing';

  @override
  String get businessOfferingPhotographyServices => 'Photography services';

  @override
  String get businessOfferingFiberProcessing => 'Fiber processing';

  @override
  String get businessActionShop => 'Shop';

  @override
  String get businessActionBrowsePatterns => 'Browse patterns';

  @override
  String get businessActionRequestCustomOrder => 'Request custom order';

  @override
  String get businessActionBookClass => 'Book class';

  @override
  String get businessActionBookAppointment => 'Book appointment';

  @override
  String get businessActionViewEventCalendar => 'View event calendar';

  @override
  String get businessActionEmail => 'Email';

  @override
  String get businessActionVisitWebsite => 'Visit website';

  @override
  String get businessActionWholesaleEnquiries => 'Wholesale enquiries';

  @override
  String get businessProductsOwnerEmpty =>
      'Add featured products to help visitors find your work.';

  @override
  String get businessProductsVisitorEmpty => 'No featured products yet.';

  @override
  String get businessProductsManageAction => 'Manage products';

  @override
  String businessProductOpen(String title) {
    return 'Open $title outside CraftSky';
  }

  @override
  String get businessEventsOwnerEmpty =>
      'Add an event appearance to share what’s coming up.';

  @override
  String get businessEventsVisitorEmpty => 'No upcoming events yet.';

  @override
  String get businessEventsManageAction => 'Manage events';

  @override
  String get businessEventsLoadError => 'Upcoming events could not be loaded.';

  @override
  String get businessEventsLoadMoreError => 'Couldn’t load more events.';

  @override
  String get businessEventsRefreshError => 'Couldn’t refresh upcoming events.';

  @override
  String get businessEventsRetryAction => 'Retry';

  @override
  String get businessEventsLoadMoreAction => 'Load more';

  @override
  String get businessEventsRefreshAction => 'Refresh';

  @override
  String get businessEventsEnd => 'You’ve reached the end.';

  @override
  String get businessEventAllDayDisplay => 'All day';

  @override
  String businessEventTimeRange(String start, String end) {
    return '$start–$end';
  }

  @override
  String businessEventDateRange(String start, String end, int year) {
    return '$start–$end, $year';
  }

  @override
  String get businessEventRoleOrganizer => 'Organizer';

  @override
  String get businessEventRoleInstructor => 'Instructor';

  @override
  String get businessEventRoleVendor => 'Vendor';

  @override
  String get businessEventRoleExhibitor => 'Exhibitor';

  @override
  String get businessEventRoleSpeaker => 'Speaker';

  @override
  String get businessEventRoleDemonstrator => 'Demonstrator';

  @override
  String get businessEventModeInPerson => 'In person';

  @override
  String get businessEventModeOnline => 'Online';

  @override
  String get businessEventModeHybrid => 'Hybrid';

  @override
  String get businessEventDetailTitle => 'Event';

  @override
  String get businessEventUnavailableTitle => 'Event unavailable';

  @override
  String get businessEventUnavailableBody => 'This event can’t be viewed.';

  @override
  String get businessEventReportAction => 'Report event';

  @override
  String get businessEventReportActionShort => 'Report';

  @override
  String get businessEventDetailLoadError =>
      'Event details could not be loaded.';

  @override
  String get businessEventStatusScheduled => 'Scheduled';

  @override
  String get businessEventStatusCancelled => 'Cancelled';

  @override
  String get businessEventStatusPostponed => 'Postponed';

  @override
  String get businessEventLifecycleUpcoming => 'Upcoming';

  @override
  String get businessEventLifecyclePast => 'Past';

  @override
  String get businessEventDateLabel => 'Date';

  @override
  String get businessEventTimeLabel => 'Time';

  @override
  String get businessEventRolesLabel => 'Your role';

  @override
  String get businessEventModeLabel => 'Attendance mode';

  @override
  String get businessEventStatusLabel => 'Status';

  @override
  String get businessEventLifecycleLabel => 'Lifecycle';

  @override
  String get businessEventTimeZoneLabel => 'Timezone';

  @override
  String get businessEventVenueLabel => 'Venue (optional)';

  @override
  String get businessEventPublishedLabel => 'Published';

  @override
  String businessEventPublishedOn(String date) {
    return 'Published $date';
  }

  @override
  String get businessEventWebsiteAction => 'Event website';

  @override
  String get businessEventRegistrationAction => 'Register';

  @override
  String get businessProductsSettingsTitle => 'Products';

  @override
  String get businessProductsAdd => 'Add product';

  @override
  String get businessProductsEmpty => 'No featured products yet.';

  @override
  String get businessProductsUnavailable =>
      'Product management is available to business accounts.';

  @override
  String get businessProductsLoadError => 'Products could not be loaded.';

  @override
  String get businessProductsRetry => 'Retry';

  @override
  String get businessProductsConflict =>
      'These business details changed elsewhere. Reload the complete current profile before trying again.';

  @override
  String get businessProductsReload => 'Reload current profile';

  @override
  String get businessProductsSaveError =>
      'Products could not be saved. Check the fields and try again.';

  @override
  String get businessProductsUploadError =>
      'The image could not be uploaded. Try again.';

  @override
  String get businessLoading => 'Loading business information';

  @override
  String get businessSaving => 'Saving business information';

  @override
  String get businessImageUploading => 'Uploading image';

  @override
  String businessProductsCount(int count, int limit) {
    return '$count of $limit products';
  }

  @override
  String businessProductEdit(String title) {
    return 'Edit $title';
  }

  @override
  String businessProductRemove(String title) {
    return 'Remove $title';
  }

  @override
  String get businessProductRemoveConfirmTitle => 'Remove product?';

  @override
  String businessProductRemoveConfirmMessage(String title) {
    return 'Remove $title from your featured products? This change is saved immediately.';
  }

  @override
  String get businessProductRemoveConfirm => 'Remove';

  @override
  String get businessProductRemoveCancel => 'Keep product';

  @override
  String businessProductMoveUp(String title) {
    return 'Move $title up';
  }

  @override
  String businessProductMoveDown(String title) {
    return 'Move $title down';
  }

  @override
  String get businessProductEditorAddTitle => 'Add product';

  @override
  String get businessProductEditorEditTitle => 'Edit product';

  @override
  String get businessProductTitleLabel => 'Title';

  @override
  String get businessProductDestinationLabel => 'Destination';

  @override
  String get businessProductDestinationHint => 'https://example.com/product';

  @override
  String get businessProductAmountLabel => 'Amount';

  @override
  String get businessProductCurrencyLabel => 'Currency';

  @override
  String get businessProductAltLabel => 'Image description';

  @override
  String get businessProductAddImage => 'Add image';

  @override
  String get businessProductReplaceImage => 'Replace image';

  @override
  String get businessProductRemoveImage => 'Remove image';

  @override
  String get businessProductSave => 'Save product';

  @override
  String get businessProductCancel => 'Cancel';

  @override
  String get businessProductTitleRequired => 'Add a title.';

  @override
  String get businessProductTitleTooLong => 'Use 150 characters or fewer.';

  @override
  String get businessProductDestinationInvalid =>
      'Enter a credential-free HTTPS link.';

  @override
  String get businessProductDestinationDuplicate =>
      'Use a different destination. Each product must link to a unique page.';

  @override
  String get businessProductImageRequired => 'Add an image.';

  @override
  String get businessProductPriceInvalid =>
      'Enter a canonical amount and uppercase ISO currency code, or clear both.';

  @override
  String get businessEventCreateTitle => 'Create event';

  @override
  String get businessEventEditTitle => 'Edit event';

  @override
  String get businessEventSave => 'Save event';

  @override
  String get businessEventNameLabel => 'Event name';

  @override
  String get businessEventNameRequired => 'Add an event name.';

  @override
  String get businessEventStartLabel => 'Start';

  @override
  String get businessEventEndLabel => 'End';

  @override
  String get businessEventDateTimeHint => 'Select date and time';

  @override
  String get businessEventTimeHint => 'YYYY-MM-DD HH:MM';

  @override
  String get businessEventTimeInvalid => 'Enter a valid start and end.';

  @override
  String get businessEventEndAfterStart => 'End must be after start.';

  @override
  String get businessEventAllDay => 'All-day event';

  @override
  String get businessEventRolesRequired => 'Choose at least one role.';

  @override
  String get businessEventSummaryLabel => 'Summary (optional)';

  @override
  String get businessEventUriLabel => 'Event link (optional)';

  @override
  String get businessEventRegistrationUriLabel =>
      'Registration link (optional)';

  @override
  String get businessEventImageDescriptionLabel => 'Image description';

  @override
  String get businessEventAddImage => 'Add image';

  @override
  String get businessEventReplaceImage => 'Replace image';

  @override
  String get businessEventRemoveImage => 'Remove image';

  @override
  String get businessEventUploadError =>
      'The image could not be uploaded. Try again.';

  @override
  String get businessEventValidationError =>
      'Check the event details and try again.';

  @override
  String get businessEventSaveError =>
      'The event could not be saved. Try again.';

  @override
  String get businessEventConflict =>
      'This event changed elsewhere. Reload the current event before trying again.';

  @override
  String get businessEventReload => 'Reload current event';

  @override
  String get businessEventDiscardTitle => 'Discard event changes?';

  @override
  String get businessEventDiscardMessage =>
      'Your unsaved event changes will be lost.';

  @override
  String get businessEventDiscard => 'Discard';

  @override
  String get businessEventKeepEditing => 'Keep editing';

  @override
  String get businessEventDiagnosticOwnerNotBusiness =>
      'Your account is not currently presented as a business.';

  @override
  String get businessEventDiagnosticInvalidTimeRange =>
      'The event’s time range is invalid.';

  @override
  String get businessEventDiagnosticDurationExceedsLimit =>
      'The event is longer than the supported limit.';

  @override
  String get businessEventDiagnosticRecordModerated =>
      'This event is hidden by moderation.';

  @override
  String get businessEventDiagnosticEnded => 'This event has ended.';

  @override
  String get businessEventDiagnosticCancelled => 'This event is cancelled.';

  @override
  String get businessEventDiagnosticPostponed => 'This event is postponed.';

  @override
  String get businessEventsSettingsTitle => 'Events';

  @override
  String get businessEventsUpcomingTab => 'Upcoming';

  @override
  String get businessEventsHistoryTab => 'History';

  @override
  String get businessEventsUnavailable =>
      'Event management is available to business accounts.';

  @override
  String get businessEventsOwnerLoadError => 'Events could not be loaded.';

  @override
  String get businessEventsOwnerRefreshError =>
      'Events could not be refreshed.';

  @override
  String get businessEventsUpcomingEmpty => 'No upcoming events yet.';

  @override
  String get businessEventsHistoryEmpty => 'No event history yet.';

  @override
  String businessEventManage(String name) {
    return 'Manage $name';
  }

  @override
  String get businessEventEditAction => 'Edit event';

  @override
  String get businessEventCancelAction => 'Cancel event';

  @override
  String get businessEventPostponeAction => 'Postpone event';

  @override
  String get businessEventDeleteAction => 'Delete event';

  @override
  String get businessEventDeleteConfirmTitle => 'Delete this event?';

  @override
  String get businessEventDeleteConfirmMessage =>
      'This permanently deletes the event record from your account.';

  @override
  String get businessEventDeleteConfirmAction => 'Delete';
}
