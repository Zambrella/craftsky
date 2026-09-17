import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as model;
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/services/session_validation_coordinator.dart';
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/foreground_notification_event.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/models/notification_permission.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_service_provider.dart';
import 'package:craftsky_app/notifications/services/notification_service.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

final class CriticalJourneyHarness {
  CriticalJourneyHarness._({
    required this.server,
    required this.storage,
    required this.notificationService,
    required this.container,
  });

  final LoopbackAppView server;
  final InMemorySessionRegistryStorage storage;
  final IntegrationNotificationService notificationService;
  final ProviderContainer container;

  static Future<CriticalJourneyHarness> start() async {
    initializeMappers();
    final preferences = await SharedPreferences.getInstance();
    await preferences.clear();

    final server = await LoopbackAppView.start();
    final storage = InMemorySessionRegistryStorage(_initialRegistry());
    final notificationService = IntegrationNotificationService();
    final newness = ZeroNotificationNewnessRepository();

    final container = ProviderContainer(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(storage),
        sessionValidationLauncherProvider.overrideWithValue((_) async {}),
        activeAccountInitializationProvider.overrideWith((ref) async {
          final registry = await ref.watch(sessionRegistryProvider.future);
          final lease = registry.activeLease;
          if (lease == null) return null;
          return ActiveAccountInitialization(
            lease: lease,
            languagePreferences: const LanguagePreferences(
              primaryLanguage: 'en',
              contentLanguages: ['en'],
            ),
            onboardingComplete: true,
          );
        }),
        dioProvider.overrideWith((ref) {
          final registry = ref.watch(sessionRegistryProvider).requireValue;
          final activeDid = registry.activeDid;
          final token = activeDid == null
              ? null
              : registry.sessions[activeDid]?.token;
          final dio = server.client(token: token);
          ref.onDispose(() => dio.close(force: true));
          return dio;
        }),
        accountDioProvider.overrideWith((ref, account) async {
          final registry = await ref.watch(sessionRegistryProvider.future);
          final token = registry.sessions[account.did]?.token;
          final dio = server.client(token: token);
          ref.onDispose(() => dio.close(force: true));
          return dio;
        }),
        notificationServiceProvider.overrideWithValue(notificationService),
        notificationNewnessRepositoryProvider.overrideWithValue(newness),
        accountNotificationNewnessRepositoryProvider.overrideWith(
          (ref, account) async => newness,
        ),
      ],
    );

    return CriticalJourneyHarness._(
      server: server,
      storage: storage,
      notificationService: notificationService,
      container: container,
    );
  }

  Widget get app => UncontrolledProviderScope(
    container: container,
    child: const App(onInitializationResolved: _ignore),
  );

  Future<AppDependencies> loadPlatformDependencies() =>
      container.read(appDependenciesProvider.future);

  Future<void> close() async {
    container.dispose();
    await notificationService.shutdown();
    await server.close();
  }
}

void configureCriticalJourneyPreferences() {
  SharedPreferences.setPrefix('craftsky.integration.');
}

void _ignore() {}

model.SessionRegistry _initialRegistry() {
  final registry = model.SessionRegistry.empty()
      .upsertAndActivate(
        token: 'bob-token',
        did: 'did:plc:bob',
        handle: 'bob.test',
        cachedDisplayName: 'Bob',
      )
      .upsertAndActivate(
        token: 'alice-token',
        did: 'did:plc:alice',
        handle: 'alice.test',
        cachedDisplayName: 'Alice',
      );
  return registry.saveRoutingBinding(
    registry.activeLease!.session,
    'integration-alice-binding',
  );
}

final class InMemorySessionRegistryStorage implements SessionRegistryStorage {
  InMemorySessionRegistryStorage(this.value);

  model.SessionRegistry value;

  @override
  Future<model.SessionRegistry> read() async => value;

  @override
  Future<void> write(model.SessionRegistry registry) async => value = registry;
}

final class ZeroNotificationNewnessRepository
    implements NotificationNewnessRepository {
  @override
  Future<int> count() async => 0;

  @override
  Future<void> markSeen() async {}
}

final class IntegrationNotificationService implements NotificationService {
  final _opens = StreamController<NotificationOpenAttempt>.broadcast();
  bool initialized = false;

  @override
  Future<void> initialize() async => initialized = true;

  @override
  Future<void> dispose() async {}

  Future<void> shutdown() => _opens.close();

  @override
  Future<void> deleteToken() async {}

  @override
  Stream<ForegroundNotificationEvent> get foregroundEvents =>
      const Stream.empty();

  @override
  Future<NotificationPermission> getPermission() async =>
      NotificationPermission.denied;

  @override
  Future<String?> getToken() async => null;

  @override
  Stream<NotificationOpenAttempt> get openedNotifications => _opens.stream;

  @override
  Future<void> openSystemNotificationSettings() async {}

  @override
  Future<NotificationPermission> requestPermission() async =>
      NotificationPermission.denied;

  @override
  Future<NotificationOpenAttempt?> takeInitialOpen() async => null;

  @override
  Stream<String> get tokenRefreshes => const Stream.empty();
}

final class RecordedRequest {
  const RecordedRequest({
    required this.method,
    required this.path,
    required this.authorization,
    this.body,
  });

  final String method;
  final String path;
  final String? authorization;
  final Map<String, dynamic>? body;
}

final class LoopbackAppView {
  LoopbackAppView._(this._server);

  final HttpServer _server;
  final requests = <RecordedRequest>[];
  final createBodies = <Map<String, dynamic>>[];

  static Future<LoopbackAppView> start() async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final appView = LoopbackAppView._(server);
    server.listen(appView._handle);
    return appView;
  }

  String get baseUrl => 'http://${_server.address.address}:${_server.port}';

  Dio client({String? token}) => Dio(
    BaseOptions(
      baseUrl: baseUrl,
      headers: {
        'Content-Type': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      },
    ),
  );

  Future<void> close() => _server.close(force: true);

  Future<void> _handle(HttpRequest request) async {
    Map<String, dynamic>? body;
    if (request.method != 'GET' && request.method != 'HEAD') {
      final source = await utf8.decoder.bind(request).join();
      if (source.isNotEmpty) {
        body = jsonDecode(source) as Map<String, dynamic>;
      }
    }
    final authorization = request.headers.value(
      HttpHeaders.authorizationHeader,
    );
    requests.add(
      RecordedRequest(
        method: request.method,
        path: request.uri.path,
        authorization: authorization,
        body: body,
      ),
    );

    final response = switch ((
      request.method,
      request.uri.path,
    )) {
      ('GET', '/v1/feed/timeline') => _timeline(authorization),
      ('POST', '/v1/posts') => _createPost(body!, authorization),
      ('GET', '/v1/profiles/me') => _profileForToken(authorization),
      ('GET', final path) when path.startsWith('/v1/profiles/@') =>
        _profileResponse(path),
      ('GET', '/v1/scheduled-posts') => <String, dynamic>{
        'items': <dynamic>[],
      },
      _ => <String, dynamic>{'items': <dynamic>[]},
    };

    request.response
      ..statusCode = request.method == 'POST'
          ? HttpStatus.created
          : HttpStatus.ok
      ..headers.contentType = ContentType.json
      ..write(jsonEncode(response));
    await request.response.close();
  }

  Map<String, dynamic> _timeline(String? authorization) {
    final account = authorization == 'Bearer bob-token' ? 'bob' : 'alice';
    final post = _post(
      did: 'did:plc:$account',
      handle: '$account.test',
      rkey: '$account-read',
      text: '${_title(account)} timeline',
    );
    return {
      'items': [
        {'itemKey': 'post:${post['uri']}', 'post': post},
      ],
    };
  }

  Map<String, dynamic> _createPost(
    Map<String, dynamic> body,
    String? authorization,
  ) {
    createBodies.add(body);
    final account = authorization == 'Bearer bob-token' ? 'bob' : 'alice';
    return _post(
      did: 'did:plc:$account',
      handle: '$account.test',
      rkey: '$account-created',
      text: body['text']! as String,
      langs: (body['langs']! as List<dynamic>).cast<String>(),
    );
  }

  Map<String, dynamic> _profileForToken(String? authorization) {
    final account = authorization == 'Bearer bob-token' ? 'bob' : 'alice';
    return _profile(account, _title(account));
  }

  Map<String, dynamic> _profileResponse(String path) {
    if (path.endsWith('/posts') ||
        path.endsWith('/comments') ||
        path.endsWith('/projects')) {
      return {'items': <Map<String, dynamic>>[]};
    }
    final encoded = path.substring('/v1/profiles/@'.length);
    final identifier = Uri.decodeComponent(encoded);
    if (identifier == 'did:plc:notified') {
      return _profile('notified', 'Notified Maker');
    }
    final account = identifier.contains('bob') ? 'bob' : 'alice';
    return _profile(account, _title(account));
  }

  static String _title(String account) =>
      '${account[0].toUpperCase()}${account.substring(1)}';

  static Map<String, dynamic> _profile(String account, String displayName) => {
    'did': 'did:plc:$account',
    'handle': '$account.test',
    'displayName': displayName,
    'description': 'Deterministic integration profile',
    'crafts': ['sewing'],
  };

  static Map<String, dynamic> _post({
    required String did,
    required String handle,
    required String rkey,
    required String text,
    List<String> langs = const ['en'],
  }) => {
    'uri': 'at://$did/social.craftsky.feed.post/$rkey',
    'cid': 'bafy$rkey',
    'rkey': rkey,
    'text': text,
    'langs': langs,
    'tags': <String>[],
    'likeCount': 0,
    'repostCount': 0,
    'replyCount': 0,
    'viewerHasLiked': false,
    'viewerHasReposted': false,
    'viewerHasSaved': false,
    'sponsored': false,
    'viewerHasReplied': false,
    'createdAt': '2026-09-17T10:00:00.000Z',
    'indexedAt': '2026-09-17T10:00:01.000Z',
    'author': {'did': did, 'handle': handle, 'displayName': _title(handle[0])},
  };
}
