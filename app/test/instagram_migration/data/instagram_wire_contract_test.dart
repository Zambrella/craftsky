import 'dart:convert';
import 'dart:io';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/instagram_migration/data/instagram_migration_api_client.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_page.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/providers/error_mapping_interceptor.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  late Map<String, dynamic> corpus;

  setUpAll(() {
    initializeMappers();
    corpus = _loadCorpus();
  });

  test('IT-021 decodes and canonically re-encodes every public state', () {
    expect(corpus['schemaVersion'], 1);
    expect(
      _map(corpus['fixturePolicy'])['classification'],
      'whollySynthetic',
    );
    expect(
      _map(corpus['fixturePolicy'])['containsUserDerivedData'],
      isFalse,
    );

    final verificationStates = <InstagramVerificationState>{};
    for (final fixture in _listOfMaps(corpus['verificationResponses'])) {
      final body = _map(fixture['body']);
      final creation = fixture['shape'] == 'creation';
      final model = creation
          ? InstagramVerificationAttempt.fromCreationMap(body)
          : InstagramVerificationAttempt.fromMap(body);
      verificationStates.add(model.state);
      expect(_encodeVerification(model, creation: creation), body);
    }
    expect(verificationStates, {
      InstagramVerificationState.pendingDm,
      InstagramVerificationState.processing,
      InstagramVerificationState.pendingConfirmation,
      InstagramVerificationState.confirmed,
      InstagramVerificationState.expired,
      InstagramVerificationState.cancelled,
      InstagramVerificationState.superseded,
      InstagramVerificationState.rejected,
      InstagramVerificationState.conflicted,
    });

    for (final fixture in _listOfMaps(corpus['confirmationResponses'])) {
      final body = _map(fixture['body']);
      final model = InstagramVerificationConfirmation.fromMap(body);
      expect(_encodeConfirmation(model), body);
    }

    final accountStates = <InstagramAccountLinkState>{};
    for (final fixture in _listOfMaps(corpus['accountResponses'])) {
      final body = _map(fixture['body']);
      final model = InstagramAccountStatus.fromMap(body);
      if (model.account case final account?) {
        accountStates.add(account.state);
      }
      expect(_encodeAccountStatus(model), body);
    }
    expect(accountStates, {
      InstagramAccountLinkState.active,
      InstagramAccountLinkState.membershipInactive,
      InstagramAccountLinkState.revoked,
      InstagramAccountLinkState.superseded,
      InstagramAccountLinkState.disputed,
    });

    for (final fixture in _listOfMaps(corpus['importCreateResponses'])) {
      final body = _map(fixture['body']);
      final model = InstagramImportCreateResult.fromMap(body);
      expect(_encodeImportCreate(model), body);
    }

    final importStates = <InstagramImportState>{};
    for (final fixture in _listOfMaps(corpus['importResponses'])) {
      final body = _map(fixture['body']);
      final model = InstagramImportSummary.fromMap(body);
      importStates.add(model.state);
      expect(_encodeImport(model), body);
    }
    expect(importStates, {
      InstagramImportState.active,
      InstagramImportState.membershipInactive,
    });

    final suggestionStates = <InstagramSuggestionState>{};
    for (final fixture in _listOfMaps(corpus['suggestionResponses'])) {
      final body = _map(fixture['body']);
      final model = InstagramSuggestion.fromMap(body);
      suggestionStates.add(model.state);
      expect(_encodeSuggestion(model), body);
    }
    expect(suggestionStates, {
      InstagramSuggestionState.pending,
      InstagramSuggestionState.accepting,
      InstagramSuggestionState.accepted,
      InstagramSuggestionState.alreadyFollowing,
      InstagramSuggestionState.dismissed,
      InstagramSuggestionState.invalidated,
    });

    for (final fixture in _listOfMaps(
      corpus['suggestionActionResponses'],
    )) {
      final body = _map(fixture['body']);
      final model = InstagramSuggestionActionResult.fromMap(body);
      expect(_encodeSuggestionAction(model), body);
    }
  });

  test('IT-021 request models emit only exact bounded fields', () {
    final manual = _fixture(corpus['requests'], 'import.create.manual');
    final manualBody = _map(manual['body']);
    final manualRequest = InstagramImportRequest(
      sourceType: InstagramImportSourceType.fromWire(
        manualBody['sourceType'] as String,
      ),
      entries: [
        for (final entry in _listOfMaps(manualBody['entries']))
          InstagramImportEntry(
            username: entry['username'] as String,
          ),
      ],
    );
    expect(manualRequest.toMap(), manualBody);

    final jsonFixture = _fixture(
      corpus['requests'],
      'import.create.instagramJson',
    );
    final jsonBody = _map(jsonFixture['body']);
    final jsonRequest = InstagramImportRequest(
      sourceType: InstagramImportSourceType.fromWire(
        jsonBody['sourceType'] as String,
      ),
      entries: [
        for (final entry in _listOfMaps(jsonBody['entries']))
          InstagramImportEntry(
            username: entry['username'] as String,
          ),
      ],
    );
    expect(jsonRequest.toMap(), jsonBody);

    expect(
      const InstagramAccountSettingsPatch(discoverable: false).toMap(),
      _map(_fixture(corpus['requests'], 'settings.discovery')['body']),
    );
    expect(
      const InstagramAccountSettingsPatch(
        discoverable: true,
        reactivate: true,
      ).toMap(),
      _map(_fixture(corpus['requests'], 'settings.reactivate')['body']),
    );
    expect(
      const InstagramImportPatch(reactivate: true).toMap(),
      _map(_fixture(corpus['requests'], 'import.reactivate')['body']),
    );

    final allowedImportKeys = {'sourceType', 'entries'};
    expect(jsonRequest.toMap().keys.toSet(), allowedImportKeys);
    for (final forbidden in [
      'rawArchive',
      'rawBody',
      'filename',
      'url',
      'message',
      'profileResponse',
    ]) {
      expect(jsonEncode(jsonRequest.toMap()), isNot(contains(forbidden)));
    }
  });

  test('IT-021 consumes shared successes at the real API boundary', () async {
    final dio = _buildDio();
    final adapter = DioAdapter(dio: dio);
    final createVerification = _fixture(
      corpus['verificationResponses'],
      'verification.created',
    );
    final pendingConfirmation = _fixture(
      corpus['verificationResponses'],
      'verification.pendingConfirmation',
    );
    final confirmation = _fixture(
      corpus['confirmationResponses'],
      'verification.confirm.success',
    );
    final account = _fixture(corpus['accountResponses'], 'account.active');
    final importCreate = _fixture(
      corpus['importCreateResponses'],
      'import.create.success',
    );
    final importDetail = _fixture(
      corpus['importResponses'],
      'import.active',
    );
    final importsPage = _fixture(
      corpus['pageContracts'],
      'imports.default.omittedCursor',
    );
    final suggestionsPage = _fixture(
      corpus['pageContracts'],
      'suggestions.default.omittedCursor',
    );
    final suggestionAction = _fixture(
      corpus['suggestionActionResponses'],
      'suggestion.accept.success',
    );
    final importRequest = _map(
      _fixture(corpus['requests'], 'import.create.instagramJson')['body'],
    );

    adapter
      ..onPost(
        '/v1/migrations/instagram/verifications',
        (server) => server.reply(
          createVerification['status'] as int,
          createVerification['body'],
        ),
        data: <String, Object?>{},
      )
      ..onGet(
        pendingConfirmation['path'] as String,
        (server) => server.reply(
          pendingConfirmation['status'] as int,
          pendingConfirmation['body'],
        ),
      )
      ..onPost(
        confirmation['path'] as String,
        (server) => server.reply(
          confirmation['status'] as int,
          confirmation['body'],
        ),
        data: _map(
          _fixture(corpus['requests'], 'verification.confirm')['body'],
        ),
      )
      ..onGet(
        '/v1/migrations/instagram/account',
        (server) => server.reply(account['status'] as int, account['body']),
      )
      ..onPatch(
        '/v1/migrations/instagram/settings',
        (server) => server.reply(account['status'] as int, account['body']),
        data: _map(_fixture(corpus['requests'], 'settings.discovery')['body']),
      )
      ..onDelete(
        '/v1/migrations/instagram/account',
        (server) => server.reply(204, null),
      )
      ..onPost(
        '/v1/migrations/instagram/imports',
        (server) => server.reply(
          importCreate['status'] as int,
          importCreate['body'],
        ),
        data: importRequest,
      )
      ..onGet(
        '/v1/migrations/instagram/imports',
        (server) => server.reply(
          importsPage['status'] as int,
          importsPage['body'],
        ),
      )
      ..onGet(
        importDetail['path'] as String,
        (server) => server.reply(
          importDetail['status'] as int,
          importDetail['body'],
        ),
      )
      ..onPatch(
        importDetail['path'] as String,
        (server) => server.reply(
          importDetail['status'] as int,
          importDetail['body'],
        ),
        data: _map(
          _fixture(corpus['requests'], 'import.reactivate')['body'],
        ),
      )
      ..onDelete(
        importDetail['path'] as String,
        (server) => server.reply(204, null),
      )
      ..onGet(
        '/v1/migrations/instagram/suggestions',
        (server) => server.reply(
          suggestionsPage['status'] as int,
          suggestionsPage['body'],
        ),
      )
      ..onPost(
        suggestionAction['path'] as String,
        (server) => server.reply(
          suggestionAction['status'] as int,
          suggestionAction['body'],
        ),
      )
      ..onDelete(
        '/v1/migrations/instagram/suggestions/'
        'synthetic-suggestion-0001',
        (server) => server.reply(204, null),
      )
      ..onDelete(
        '/v1/migrations/instagram/verifications/'
        'synthetic-verification-0003',
        (server) => server.reply(204, null),
      );

    final api = InstagramMigrationApiClient(dio);
    final created = await api.createVerification();
    final status = await api.getVerification('synthetic-verification-0003');
    final confirmed = await api.confirmVerification(
      'synthetic-verification-0003',
      discoverable: true,
    );
    final accountStatus = await api.getAccount();
    final updatedAccount = await api.updateSettings(
      const InstagramAccountSettingsPatch(discoverable: false),
    );
    await api.revokeAccount();

    final request = InstagramImportRequest(
      sourceType: InstagramImportSourceType.instagramJson,
      entries: const [
        InstagramImportEntry(username: 'synthetic_following_01'),
        InstagramImportEntry(username: 'synthetic_following_02'),
      ],
    );
    final imported = await api.createImport(request);
    final importPage = await api.listImports();
    final importItem = await api.getImport('synthetic-import-0001');
    final updatedImport = await api.updateImport(
      'synthetic-import-0001',
      const InstagramImportPatch(reactivate: true),
    );
    await api.deleteImport('synthetic-import-0001');

    final suggestionPage = await api.listSuggestions();
    final accepted = await api.acceptSuggestion('synthetic-suggestion-0001');
    await api.dismissSuggestion('synthetic-suggestion-0001');
    await api.cancelVerification('synthetic-verification-0003');

    expect(created.state, InstagramVerificationState.pendingDm);
    expect(status.state, InstagramVerificationState.pendingConfirmation);
    expect(confirmed.state, InstagramVerificationState.confirmed);
    expect(accountStatus.account!.state, InstagramAccountLinkState.active);
    expect(updatedAccount.account!.state, InstagramAccountLinkState.active);
    expect(imported.import.state, InstagramImportState.active);
    expect(importPage.cursor, isNull);
    expect(importItem.sourceType, InstagramImportSourceType.instagramJson);
    expect(updatedImport.state, InstagramImportState.active);
    expect(suggestionPage.cursor, isNull);
    expect(suggestionPage.items.single.state, InstagramSuggestionState.pending);
    expect(accepted.state, InstagramSuggestionState.accepted);
  });

  test('IT-021 locks default/max pages and optional cursor wire shape', () {
    final limits = _map(corpus['limits']);
    expect(limits, {'defaultPageSize': 20, 'maxPageSize': 50});
    final coverage = <String, Set<String>>{};

    for (final fixture in _listOfMaps(corpus['pageContracts'])) {
      final body = _map(fixture['body']);
      final requestedLimit = fixture['requestedLimit'] as int?;
      final kind = requestedLimit == null ? 'default' : 'max';
      coverage.putIfAbsent(fixture['resource'] as String, () => {}).add(kind);
      if (kind == 'default') {
        expect(fixture['effectiveLimit'], limits['defaultPageSize']);
        expect(fixture['requestCursor'], isNull);
        expect(body, isNot(contains('cursor')));
      } else {
        expect(requestedLimit, limits['maxPageSize']);
        expect(fixture['effectiveLimit'], limits['maxPageSize']);
        expect(fixture['requestCursor'], isA<String>());
        expect(body['cursor'], isA<String>());
      }

      switch (fixture['resource']) {
        case 'imports':
          final model = InstagramImportPage.fromMap(body);
          expect(_encodeImportPage(model), body);
        case 'suggestions':
          final model = InstagramSuggestionPage.fromMap(body);
          expect(_encodeSuggestionPage(model), body);
        default:
          fail('unknown page resource ${fixture['resource']}');
      }
    }

    expect(coverage['imports'], {'default', 'max'});
    expect(coverage['suggestions'], {'default', 'max'});
  });

  test('IT-021 decodes exact social/system notification union', () {
    final contract = _map(corpus['notificationContract']);
    final body = _map(contract['body']);
    final page = NotificationPage.fromMap(body);

    expect(page.cursor, 'synthetic-notification-output-cursor');
    expect(page.items, hasLength(8));
    expect(page.items[0], isA<FollowNotification>());
    expect(page.items[1], isA<LikeNotification>());
    expect(page.items[2], isA<RepostNotification>());
    expect(page.items[3], isA<ReplyNotification>());
    expect(page.items[4], isA<MentionNotification>());
    expect(page.items[5], isA<QuoteNotification>());
    expect(page.items[6], isA<GenericNotification>());
    expect(page.items[7], isA<InstagramMatchNotification>());

    final system = page.items[7] as InstagramMatchNotification;
    expect(system.count, 99);
    expect(system.countCapped, isTrue);
    expect(
      system.destination,
      InstagramSystemDestination.instagramMigration,
    );
    expect(system, isNot(isA<SocialNotification>()));

    final rawSystem = _listOfMaps(body['items']).last;
    expect(rawSystem.keys.toSet(), {
      'id',
      'kind',
      'type',
      'createdAt',
      'indexedAt',
      'system',
    });
    for (final socialField in [
      'actor',
      'uri',
      'cid',
      'rkey',
      'references',
      'subjectPost',
      'reply',
    ]) {
      expect(rawSystem, isNot(contains(socialField)));
    }
  });

  test('IT-021 keeps every designed unknown enum safe and inert', () {
    final clientSafety = _map(corpus['clientSafety']);
    expect(
      InstagramVerificationAttempt.fromMap(
        _map(clientSafety['unknownVerificationState']),
      ).state,
      InstagramVerificationState.unknown,
    );
    expect(
      InstagramVerificationAttempt.fromMap(
        _map(clientSafety['unknownRetryCode']),
      ).retryCode,
      InstagramVerificationRetryCode.unknown,
    );
    expect(
      InstagramAccountStatus.fromMap(
        _map(clientSafety['unknownAccountState']),
      ).account!.state,
      InstagramAccountLinkState.unknown,
    );
    expect(
      InstagramImportSummary.fromMap(
        _map(clientSafety['unknownImportState']),
      ).state,
      InstagramImportState.unknown,
    );
    expect(
      InstagramImportSummary.fromMap(
        _map(clientSafety['unknownImportSource']),
      ).sourceType,
      InstagramImportSourceType.unknown,
    );
    expect(
      InstagramSuggestion.fromMap(
        _map(clientSafety['unknownSuggestionState']),
      ).state,
      InstagramSuggestionState.unknown,
    );
    expect(
      InstagramSuggestion.fromMap(
        _map(clientSafety['unknownSuggestionReason']),
      ).reason,
      InstagramSuggestionReason.unknown,
    );

    final notification = _map(corpus['notificationContract']);
    for (final fixture in _listOfMaps(notification['unknownClientCases'])) {
      final decoded = CraftskyNotification.fromMap(_map(fixture['body']));
      switch (fixture['expectedClientVariant']) {
        case 'genericSystem':
          expect(decoded, isA<GenericSystemNotification>());
          expect(decoded, isNot(isA<SocialNotification>()));
        case 'genericSocial':
          expect(decoded, isA<GenericNotification>());
          expect(decoded, isA<SocialNotification>());
        default:
          fail('unknown safe variant ${fixture['expectedClientVariant']}');
      }
    }
  });

  test(
    'IT-021 maps standard errors without retaining server messages',
    () async {
      for (final fixture in _listOfMaps(corpus['errorContracts'])) {
        final body = _map(fixture['body']);
        expect(body.keys.toSet(), {'error', 'message', 'requestId'});
        expect(body['error'], isA<String>());
        expect(body['message'], isA<String>());
        expect(body['requestId'], isA<String>());
      }

      final unavailable = _fixture(
        corpus['errorContracts'],
        'verification.unavailable',
      );
      final unavailableDio = _buildDio();
      DioAdapter(dio: unavailableDio).onPost(
        '/v1/migrations/instagram/verifications',
        (server) => server.reply(
          unavailable['status'] as int,
          unavailable['body'],
        ),
        data: <String, Object?>{},
      );
      final unavailableError = await _capture(
        InstagramMigrationApiClient(unavailableDio).createVerification,
      );
      expect(unavailableError, isA<ApiServerError>());
      final serverError = unavailableError as ApiServerError;
      expect(
        serverError.details.appViewError,
        'instagram_verification_unavailable',
      );
      expect(serverError.details.requestId, 'synthetic-request-0001');
      expect(
        serverError.toString(),
        isNot(contains(_map(unavailable['body'])['message'] as String)),
      );

      final conflict = _fixture(corpus['errorContracts'], 'link.conflict');
      final conflictDio = _buildDio();
      DioAdapter(dio: conflictDio).onPost(
        conflict['path'] as String,
        (server) => server.reply(conflict['status'] as int, conflict['body']),
        data: {'discoverable': true},
      );
      final conflictError = await _capture(
        () => InstagramMigrationApiClient(conflictDio).confirmVerification(
          'synthetic-verification-0011',
          discoverable: true,
        ),
      );
      expect(conflictError, isA<ApiBadRequest>());
      expect(
        (conflictError as ApiBadRequest).code,
        'instagram_link_conflict',
      );
    },
  );

  test('IT-021 locks DELETE privacy and callback retry metadata', () {
    final resources = <String>{};
    for (final contract in _listOfMaps(corpus['deleteContracts'])) {
      resources.add(contract['resource'] as String);
      expect(contract['method'], 'DELETE');
      expect(contract['status'], 204);
      expect(contract['bodyPresent'], isFalse);
      expect(contract['mutatesOwnedOnly'], isTrue);
      expect(
        _list(
          contract['variants'],
        ).whereType<String>().any((value) => value.startsWith('owned')),
        isTrue,
      );
      if (contract['resource'] != 'account') {
        expect(_list(contract['variants']), contains('foreign'));
      }
      expect(_list(contract['variants']), contains('absent'));
      expect(_list(contract['variants']), contains('purged'));
    }
    expect(resources, {'verification', 'account', 'import', 'suggestion'});

    final callbacks = {
      for (final contract in _listOfMaps(corpus['callbackContracts']))
        contract['id'] as String: contract,
    };
    expect(
      callbacks['callback.verify.success']!['responseBody'],
      'synthetic-callback-challenge',
    );
    final forbidden = callbacks['callback.verify.forbidden']!;
    expect(forbidden['reflectsChallenge'], isFalse);
    expect(
      forbidden['responseBody'] as String,
      isNot(contains(_map(forbidden['query'])['hub.challenge'] as String)),
    );

    for (final id in [
      'callback.delivery.sourceIpLimited',
      'callback.delivery.globalLimited',
    ]) {
      final contract = callbacks[id]!;
      expect(contract['status'], 429);
      expect(_map(contract['headers'])['Retry-After'], '60');
      expect(contract['persistPartial'], isFalse);
    }
    final limited = callbacks['callback.delivery.invalidRedemptionLimited']!;
    expect(limited['status'], 200);
    expect(limited['terminalDeduplicatedIgnored'], isTrue);
    expect(limited['sensitiveFieldsCleared'], isTrue);
    expect(limited['lookupAllowed'], isFalse);
  });
}

Map<String, dynamic> _loadCorpus() {
  final current = Directory.current;
  final candidates = [
    File(
      '${current.path}/docs/changes/'
      '2026-07-11-instagram-dm-verification/fixtures/'
      'instagram_wire/corpus.json',
    ),
    File(
      '${current.parent.path}/docs/changes/'
      '2026-07-11-instagram-dm-verification/fixtures/'
      'instagram_wire/corpus.json',
    ),
  ];
  for (final file in candidates) {
    if (file.existsSync()) {
      return _map(jsonDecode(file.readAsStringSync()));
    }
  }
  throw StateError('shared_instagram_wire_corpus_not_found');
}

Dio _buildDio() => Dio(
  BaseOptions(baseUrl: 'https://appview.synthetic.invalid'),
)..interceptors.add(const ErrorMappingInterceptor());

Future<Object> _capture(Future<Object?> Function() operation) async {
  try {
    await operation();
  } on Object catch (error) {
    return error;
  }
  throw StateError('expected_operation_to_fail');
}

Map<String, dynamic> _fixture(Object? fixtures, String id) {
  for (final fixture in _listOfMaps(fixtures)) {
    if (fixture['id'] == id) return fixture;
  }
  throw StateError('fixture_not_found:$id');
}

Map<String, dynamic> _map(Object? value) {
  if (value is! Map) throw StateError('expected_map');
  return value.map((key, value) => MapEntry(key as String, value));
}

List<dynamic> _list(Object? value) {
  if (value is! List) throw StateError('expected_list');
  return value;
}

List<Map<String, dynamic>> _listOfMaps(Object? value) => [
  for (final item in _list(value)) _map(item),
];

String _wireTime(DateTime value) =>
    value.toUtc().toIso8601String().replaceFirst(RegExp(r'\.000Z$'), 'Z');

Map<String, Object?> _encodeVerification(
  InstagramVerificationAttempt value, {
  required bool creation,
}) => {
  'verificationId': value.verificationId,
  'state': value.state.name,
  if (creation) 'challenge': value.challenge,
  'expiresAt': _wireTime(value.expiresAt),
  if (creation) 'dmUrl': value.dmUrl.toString(),
  if (value.candidateUsername != null)
    'candidateUsername': value.candidateUsername,
  if (value.retryCode != null) 'retryCode': value.retryCode!.name,
};

Map<String, Object?> _encodeAccount(InstagramAccountLink value) => {
  'state': value.state.name,
  'username': value.username,
  'discoverable': value.discoverable,
  'conflictPending': value.conflictPending,
  'reactivationRequired': value.reactivationRequired,
  'verifiedAt': _wireTime(value.verifiedAt),
};

Map<String, Object?> _encodeAccountStatus(InstagramAccountStatus value) => {
  'integrationAvailable': value.integrationAvailable,
  'account': value.account == null ? null : _encodeAccount(value.account!),
};

Map<String, Object?> _encodeConfirmation(
  InstagramVerificationConfirmation value,
) => {'state': value.state.name, 'account': _encodeAccount(value.account)};

Map<String, Object?> _encodeImport(InstagramImportSummary value) => {
  'importId': value.importId,
  'state': value.state.name,
  'sourceType': value.sourceType.wireValue,
  'followingCount': value.followingCount,
  'createdAt': _wireTime(value.createdAt),
};

Map<String, Object?> _encodeImportCreate(InstagramImportCreateResult value) => {
  'import': _encodeImport(value.import),
  'counts': {'followingCount': value.followingCount},
  'initialSuggestionCount': value.initialSuggestionCount,
};

Map<String, Object?> _encodeSuggestion(InstagramSuggestion value) => {
  'suggestionId': value.suggestionId,
  'profile': {
    'did': value.profile.did,
    'handle': value.profile.handle,
    if (value.profile.displayName != null)
      'displayName': value.profile.displayName,
    if (value.profile.avatar != null) 'avatar': value.profile.avatar,
  },
  'reason': value.reason.name,
  'state': value.state.name,
};

Map<String, Object?> _encodeSuggestionAction(
  InstagramSuggestionActionResult value,
) => {'suggestionId': value.suggestionId, 'state': value.state.name};

Map<String, Object?> _encodeImportPage(InstagramImportPage value) => {
  'items': value.items.map(_encodeImport).toList(growable: false),
  if (value.cursor != null) 'cursor': value.cursor,
};

Map<String, Object?> _encodeSuggestionPage(InstagramSuggestionPage value) => {
  'items': value.items.map(_encodeSuggestion).toList(growable: false),
  if (value.cursor != null) 'cursor': value.cursor,
};
