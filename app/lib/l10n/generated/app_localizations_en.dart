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
  String notificationRepostRow(String actor) {
    return '$actor reposted your post';
  }

  @override
  String notificationReplyRow(String actor) {
    return '$actor replied to your post';
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
  String get welcomeSignInAction => 'Sign in';

  @override
  String get welcomeCreateAccountAction => 'Create account on a PDS';

  @override
  String get signInTitle => 'Sign in';

  @override
  String get signInHandleLabel => 'Handle';

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
  String get authCompleteNoPendingSignInError =>
      'No sign-in is in progress. Please sign in again.';

  @override
  String get authCompleteStorageError =>
      'Couldn\'t save your session securely. Please sign in again.';

  @override
  String get authCompleteGenericError =>
      'Couldn\'t complete sign-in. Please sign in again.';

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
  String get feedEmpty => 'Your feed is quiet.';

  @override
  String get feedLoadError => 'Feed didn\'t load.';

  @override
  String get messengerDismiss => 'Dismiss';

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
  String get profileSettingsAction => 'Settings';

  @override
  String get profileShareAction => 'Share';

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
  String get editProfileDiscardTitle => 'Discard changes?';

  @override
  String get editProfileDiscardMessage => 'Your edits won\'t be saved.';

  @override
  String get editProfileDiscardConfirm => 'Discard';

  @override
  String get editProfileDiscardCancel => 'Keep editing';

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
}
