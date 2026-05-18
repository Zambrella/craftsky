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
  String get messengerDismiss => 'Dismiss';

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
  String get postComposeHint => 'What are you making?';

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
  String get postDeleteAction => 'Delete post';

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
  String get postDeleteMessage => 'This removes the post from Craftsky.';

  @override
  String get commentDeleteMessage => 'This removes the comment from Craftsky.';

  @override
  String get replyDeleteMessage => 'This removes the reply from Craftsky.';

  @override
  String get postDeleteConfirm => 'Delete';

  @override
  String get postDeleteSuccess => 'Post deleted.';

  @override
  String get postDeleteError => 'Couldn\'t delete post.';

  @override
  String get profileFollowComingSoon => 'Follow coming soon.';

  @override
  String get profileShareComingSoon => 'Share coming soon.';

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
  String get editProfilePhotosComingSoon => 'Photo uploads coming soon';

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
}
