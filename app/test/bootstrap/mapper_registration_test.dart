import 'package:craftsky_app/bootstrap.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('initializeMappers registers every application bootstrap mapper', () {
    initializeMappers();

    final registeredTypes = MapperContainer.globals
        .getAll()
        .map((mapper) => mapper.type.toString())
        .toSet();
    const requiredTypes = {
      'AppDependencies',
      'CraftskyDeviceInfo',
      'LoginResponse',
      'WhoAmI',
      'PendingAuth',
      'Post',
      'PostCommentSection',
      'PostPage',
      'ProfilePinState',
      'TimelinePage',
      'CreatePostImage',
      'UploadedImageBlob',
      'UserPostsState',
      'InteractionWriteResponse',
      'ModerationMetadata',
      'ReportResult',
      'ReportSubmission',
      'NotificationCategory',
      'NotificationActor',
      'NotificationReplyRef',
      'NotificationCommon',
      'NotificationPreferenceScope',
      'NotificationPreference',
      'NotificationsState',
      'Project',
      'ProjectBrowseQuery',
      'ProjectBrowseFilters',
      'UserProjectsState',
      'Profile',
      'BusinessProfile',
      'BusinessEvent',
      'BusinessEventPage',
      'RecordMutationResult',
      'ProfileAccountSummary',
      'ProfileAccountPage',
      'ProfileRelationship',
      'ProjectSearchFilters',
      'BlankSearchData',
      'SearchSort',
      'SearchSuggestionQuery',
      'HashtagSearchQuery',
      'HashtagResultSearchQuery',
      'ProfileSearchQuery',
      'PostSearchQuery',
      'ProjectSearchQuery',
      'TopHashtagsQuery',
      'HashtagSearchResult',
      'HashtagSearchPage',
      'SearchSuggestionProfileSection',
      'SearchSuggestionHashtagSection',
      'SearchSuggestions',
      'SearchPostResultsState',
      'ProfileSearchResultsState',
      'HashtagSearchResultsState',
      'RecentSearchType',
      'QueryRecentSearchPayload',
      'HashtagRecentSearchPayload',
      'ProfileRecentSearchPayload',
      'PostRecentSearchPayload',
      'ProjectRecentSearchPayload',
      'SaveRecentSearchRequest',
      'RecentSearchItem',
      'RecentSearchPage',
      'SearchPostPage',
      'ProfileSearchResult',
      'ProfileSearchPage',
      'TopHashtagsResponse',
      'TopHashtagGroup',
      'TopHashtagItem',
      'AccountSuggestion',
      'HashtagSuggestion',
    };

    expect(
      registeredTypes,
      containsAll(requiredTypes),
      reason: 'initializeMappers must register every mapper used at startup',
    );
  });
}
