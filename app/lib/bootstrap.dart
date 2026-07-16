import 'dart:async';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/pending_auth.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/interaction_write_response.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart';
import 'package:craftsky_app/feed/models/post_image_blob.dart';
import 'package:craftsky_app/feed/models/post_page.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/models/user_posts_state.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart'
    as post_comment_section_provider;
import 'package:craftsky_app/feed/providers/user_comments_provider.dart'
    as user_comments_provider;
import 'package:craftsky_app/feed/providers/user_posts_provider.dart'
    as user_posts_provider;
import 'package:craftsky_app/moderation/models/moderation_metadata.dart';
import 'package:craftsky_app/moderation/models/report_result.dart';
import 'package:craftsky_app/moderation/models/report_submission.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/models/notifications_state.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/firebase_notification_bootstrap.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/models/project_browse_filters.dart';
import 'package:craftsky_app/projects/models/user_projects_state.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart'
    as user_projects_provider;
import 'package:craftsky_app/search/models/blank_search_data.dart';
import 'package:craftsky_app/search/models/hashtag_search_page.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/search/models/project_search_filters.dart';
import 'package:craftsky_app/search/models/recent_search.dart';
import 'package:craftsky_app/search/models/search_post_page.dart';
import 'package:craftsky_app/search/models/search_queries.dart';
import 'package:craftsky_app/search/models/search_result_state.dart';
import 'package:craftsky_app/search/models/search_sort.dart';
import 'package:craftsky_app/search/models/search_suggestions.dart';
import 'package:craftsky_app/search/models/top_hashtags.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:craftsky_app/shared/api/models/whoami.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
// flutter_web_plugins ships with Flutter but is not declared in pubspec;
// it's the only place usePathUrlStrategy() lives.
// ignore: depend_on_referenced_packages
import 'package:flutter_web_plugins/url_strategy.dart';
import 'package:intl/intl.dart';
import 'package:logging/logging.dart';
import 'package:shared_preferences/shared_preferences.dart';

final _log = Logger('bootstrap');

Duration? appProviderRetry(int retryCount, Object error) => null;

final class ProviderLogger extends ProviderObserver {
  const ProviderLogger({this.reporter = const NoopErrorReporter()});

  final ErrorReporter reporter;

  static final _log = Logger('ProviderLogger');

  @override
  void didUpdateProvider(
    ProviderObserverContext context,
    Object? previousValue,
    Object? newValue,
  ) {
    _log.fine(
      'provider updated: '
      'provider=${context.provider}, '
      'previousValue=${_formatProviderValue(context.provider, previousValue)}, '
      'newValue=${_formatProviderValue(context.provider, newValue)}, '
      'mutation=${context.mutation}',
    );
  }

  @override
  void providerDidFail(
    ProviderObserverContext context,
    Object error,
    StackTrace stackTrace,
  ) {
    _log.warning(
      'provider failed: '
      'provider=${context.provider}, '
      'mutation=${context.mutation}',
      error,
      stackTrace,
    );

    final appError = AppErrorMapper.map(
      error,
      fallbackKind: AppErrorKind.backgroundLoadFailed,
      source: 'provider',
      fallbackClassification: 'provider.failed',
    );
    if (!appError.reportable) return;

    final providerName = _providerFeature(context.provider.name);
    unawaited(
      reporter.captureException(
        error,
        stackTrace: stackTrace,
        context: ReportContext(
          feature: providerName,
          operation: 'provider',
          classification: appError.sentryClassification,
          severity: appError.metadata.severity.name,
          safeDiagnostics: {
            ...appError.safeDiagnostics,
            'appErrorKind': appError.kind.name,
            'feature': providerName,
            'classification': appError.sentryClassification,
          },
        ),
      ),
    );
  }
}

String _providerFeature(String? name) {
  if (name == null || name.isEmpty) return 'riverpod.provider';
  return name;
}

String _formatProviderValue(Object provider, Object? value) {
  if (value case final AsyncValue<Object?> asyncValue) {
    return _formatAsyncValue(provider, asyncValue);
  }
  return _formatDataValue(provider, value);
}

String _formatAsyncValue(Object provider, AsyncValue<Object?> value) {
  return switch (value) {
    AsyncLoading(:final value?) =>
      'AsyncLoading(previous: ${_formatDataValue(provider, value)})',
    AsyncLoading() => 'AsyncLoading()',
    AsyncError(:final error, :final value?) =>
      'AsyncError('
          'error: $error, '
          'previous: ${_formatDataValue(provider, value)}'
          ')',
    AsyncError(:final error) => 'AsyncError(error: $error)',
    AsyncData(:final value) =>
      'AsyncData(${_formatDataValue(provider, value)})',
  };
}

String _formatDataValue(Object provider, Object? value) {
  return switch (provider) {
    user_posts_provider.UserPostsProvider() =>
      user_posts_provider.UserPosts.formatLogValue(value),
    user_comments_provider.UserCommentsProvider() =>
      user_comments_provider.UserComments.formatLogValue(value),
    user_projects_provider.UserProjectsProvider() =>
      user_projects_provider.UserProjects.formatLogValue(value),
    post_comment_section_provider.PostCommentSectionProvider() =>
      post_comment_section_provider.PostCommentSection.formatLogValue(value),
    _ when value is Iterable<Object?> =>
      '${value.runtimeType}(length: ${value.length})',
    _ when value is Map<Object?, Object?> =>
      '${value.runtimeType}(length: ${value.length})',
    _ => value.toString(),
  };
}

/// Runs platform / Flutter init before `runApp`.
///
/// IMPORTANT: must never throw in production. Anything that *can* fail
/// belongs in `appDependenciesProvider`, which has loading/error UI.
Future<void> bootstrap(
  WidgetsBinding widgetsBinding, {
  ErrorReporter reporter = const NoopErrorReporter(),
}) async {
  _log.fine('bootstrap starting');

  // Web: path URL strategy (no `#` in URLs).
  usePathUrlStrategy();

  if (kIsWeb) {
    _log.fine('web detected, skipping native init');
    runApp(
      ProviderScope(
        observers: [ProviderLogger(reporter: reporter)],
        retry: appProviderRetry,
        child: const App(),
      ),
    );
    return;
  }

  // Default locale for intl date/number formatting.
  final localeName = PlatformDispatcher.instance.locale.toString();
  Intl.defaultLocale = localeName;
  _log.fine('Intl.defaultLocale=$localeName');

  // dart_mappable mapper init — empty for now, grows as models are added.
  initializeMappers();

  // Namespace all shared_preferences keys.
  SharedPreferences.setPrefix('craftsky.');

  if (defaultTargetPlatform == TargetPlatform.android) {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    SystemChrome.setSystemUIOverlayStyle(
      const SystemUiOverlayStyle(
        systemNavigationBarColor: Colors.transparent,
        systemNavigationBarDividerColor: Colors.transparent,
        systemNavigationBarContrastEnforced: true,
      ),
    );
  }

  final notificationService = await bootstrapFirebaseNotificationService();

  // Fail fast if --dart-define=CRAFTSKY_API_BASE_URL is missing in a
  // release build. Building the provider throws StateError before any
  // networking is attempted. We dispose the throwaway container so the
  // check stays cheap; the real app creates its own via ProviderScope.
  //
  // Also eagerly resolve deviceIdProvider so the first /v1/* request
  // has the header. The server enforces X-Craftsky-Device-Id on all
  // authenticated routes; if the provider hasn't resolved by the time
  // the session Dio fires its first request, the server 400s. The
  // eager await here guarantees the future is hot before runApp.
  final probe = ProviderContainer(
    observers: [ProviderLogger(reporter: reporter)],
    retry: appProviderRetry,
  );
  try {
    probe.read(dioProvider);
    await probe.read(deviceIdProvider.future);
  } finally {
    probe.dispose();
  }

  _log.fine('bootstrap complete');

  runApp(
    ProviderScope(
      observers: [ProviderLogger(reporter: reporter)],
      retry: appProviderRetry,
      overrides: [
        notificationServiceProvider.overrideWithValue(notificationService),
      ],
      child: const App(),
    ),
  );
}

/// Initialize all `dart_mappable` mappers here as models are added.
void initializeMappers() {
  AppDependenciesMapper.ensureInitialized();
  CraftskyDeviceInfoMapper.ensureInitialized();
  LoginResponseMapper.ensureInitialized();
  WhoAmIMapper.ensureInitialized();
  StoredSessionMapper.ensureInitialized();
  PendingAuthMapper.ensureInitialized();
  PostMapper.ensureInitialized();
  PostCommentSectionMapper.ensureInitialized();
  PostPageMapper.ensureInitialized();
  TimelinePageMapper.ensureInitialized();
  CreatePostImageMapper.ensureInitialized();
  UploadedImageBlobMapper.ensureInitialized();
  UserPostsStateMapper.ensureInitialized();
  InteractionWriteResponseMapper.ensureInitialized();
  ModerationMetadataMapper.ensureInitialized();
  ReportResultMapper.ensureInitialized();
  ReportSubmissionMapper.ensureInitialized();
  NotificationCategoryMapper.ensureInitialized();
  NotificationActorMapper.ensureInitialized();
  NotificationReplyRefMapper.ensureInitialized();
  NotificationCommonMapper.ensureInitialized();
  NotificationPreferenceScopeMapper.ensureInitialized();
  NotificationPreferenceMapper.ensureInitialized();
  NotificationsStateMapper.ensureInitialized();
  ProjectMapper.ensureInitialized();
  ProjectBrowseQueryMapper.ensureInitialized();
  ProjectBrowseFiltersMapper.ensureInitialized();
  UserProjectsStateMapper.ensureInitialized();
  ProfileMapper.ensureInitialized();
  ProfileAccountSummaryMapper.ensureInitialized();
  ProfileAccountPageMapper.ensureInitialized();
  ProjectSearchFiltersMapper.ensureInitialized();
  BlankSearchDataMapper.ensureInitialized();
  SearchSortMapper.ensureInitialized();
  SearchSuggestionQueryMapper.ensureInitialized();
  HashtagSearchQueryMapper.ensureInitialized();
  HashtagResultSearchQueryMapper.ensureInitialized();
  ProfileSearchQueryMapper.ensureInitialized();
  PostSearchQueryMapper.ensureInitialized();
  ProjectSearchQueryMapper.ensureInitialized();
  TopHashtagsQueryMapper.ensureInitialized();
  HashtagSearchResultMapper.ensureInitialized();
  HashtagSearchPageMapper.ensureInitialized();
  SearchSuggestionProfileSectionMapper.ensureInitialized();
  SearchSuggestionHashtagSectionMapper.ensureInitialized();
  SearchSuggestionsMapper.ensureInitialized();
  SearchPostResultsStateMapper.ensureInitialized();
  ProfileSearchResultsStateMapper.ensureInitialized();
  HashtagSearchResultsStateMapper.ensureInitialized();
  RecentSearchTypeMapper.ensureInitialized();
  QueryRecentSearchPayloadMapper.ensureInitialized();
  HashtagRecentSearchPayloadMapper.ensureInitialized();
  ProfileRecentSearchPayloadMapper.ensureInitialized();
  PostRecentSearchPayloadMapper.ensureInitialized();
  ProjectRecentSearchPayloadMapper.ensureInitialized();
  SaveRecentSearchRequestMapper.ensureInitialized();
  RecentSearchItemMapper.ensureInitialized();
  RecentSearchPageMapper.ensureInitialized();
  SearchPostPageMapper.ensureInitialized();
  ProfileSearchResultMapper.ensureInitialized();
  ProfileSearchPageMapper.ensureInitialized();
  TopHashtagsResponseMapper.ensureInitialized();
  TopHashtagGroupMapper.ensureInitialized();
  TopHashtagItemMapper.ensureInitialized();
  AccountSuggestionMapper.ensureInitialized();
  HashtagSuggestionMapper.ensureInitialized();
}
