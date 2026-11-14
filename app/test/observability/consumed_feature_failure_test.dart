import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/business/data/business_repository.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/business/providers/owner_business_events_provider.dart';
import 'package:craftsky_app/business/providers/products_controller.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_preferences_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_seen_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/observability/sentry_error_reporter.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sentry_flutter/sentry_flutter.dart';

import '../profile/fakes/fake_profile_repository.dart';
import '../test_support/serialized_sentry_transport.dart';
import '../test_support/session_registry_storage.dart';

const _requestId = '10000000-0000-4000-8000-000000000013';
const _serverFailure = ApiServerError(
  'PRIVATE_SERVER_PROSE',
  details: ApiFailureDetails(
    statusCode: 502,
    requestId: _requestId,
    fields: {'PRIVATE_FIELD': 'PRIVATE_DRAFT'},
  ),
);
final _stack = StackTrace.fromString(
  '#0 nativeOperation (package:craftsky_app/native.dart:18:3)',
);
final _account = AccountKey('did:plc:alice');

void main() {
  for (final flow in _Flow.values) {
    for (final actionable in [true, false]) {
      test(
        '${flow.name} owns its consumed failure (actionable=$actionable)',
        () async {
          final reporter = _Reporter();
          final error = actionable
              ? _serverFailure
              : const ApiNetworkError('PRIVATE_OFFLINE');
          final container = await _container(
            error,
            reporter,
            guest: flow == _Flow.guestEventsMore,
          );
          await _exercise(container, flow);
          await container.pump();
          if (actionable) {
            expect(reporter.errors, [error]);
            expect(reporter.stacks.single.toString(), _stack.toString());
            final context = reporter.contexts.single;
            expect(context.operation, _operation(flow));
            if (flow == _Flow.guestEventsMore) {
              expect(context.workflow, isNull);
            } else {
              expect(context.workflow, isA<PrivateOperationalFailureContext>());
              expect(context.workflow!.selectedFields, {
                'operationAccountDid': 'did:plc:alice',
              });
            }
          } else {
            expect(reporter.errors, isEmpty);
          }
        },
      );
    }
  }

  for (final accountScoped in [false, true]) {
    for (final actionable in [false, true]) {
      test(
        'mark seen owns failure '
        '(account=$accountScoped, actionable=$actionable)',
        () async {
          final reporter = _Reporter();
          final error = actionable
              ? _serverFailure
              : const ApiNetworkError('PRIVATE_NETWORK');
          final container = await _container(error, reporter);
          final coordinator = accountScoped
              ? await container.read(
                  accountNotificationSeenProvider(_account).future,
                )
              : container.read(notificationSeenProvider);
          await coordinator.afterSuccessfulRender(1);
          expect(reporter.errors, actionable ? [error] : isEmpty);
          if (actionable) {
            expect(reporter.stacks.single.toString(), _stack.toString());
            expect(
              reporter.contexts.single.operation,
              'notifications.mark_seen',
            );
            expect(reporter.contexts.single.workflow!.selectedFields, {
              'operationAccountDid': 'did:plc:alice',
            });
          }
        },
      );
    }
    test(
      'mark-seen count refresh retains provider ownership '
      '(account=$accountScoped)',
      () async {
        final reporter = _Reporter();
        final container = await _container(
          _serverFailure,
          reporter,
          newness: _RefreshFails(_serverFailure),
        );
        final provider = accountScoped
            ? accountNotificationNewCountProvider(_account)
            : notificationNewCountProvider;
        await container.read(provider.future);
        final coordinator = accountScoped
            ? await container.read(
                accountNotificationSeenProvider(_account).future,
              )
            : container.read(notificationSeenProvider);
        await coordinator.afterSuccessfulRender(1);
        await container.pump();
        expect(container.read(provider).hasError, isTrue);
        expect(reporter.errors, [_serverFailure]);
        expect(reporter.contexts.single.operation, 'provider');
      },
    );
  }

  for (final error in [
    const ApiCanceled(),
    const ApiBadRequest('validation_failed'),
  ]) {
    test(
      'correctable/canceled product failures stay out of issues ($error)',
      () async {
        final reporter = _Reporter();
        final container = await _container(error, reporter);
        await _exercise(container, _Flow.products);
        expect(reporter.errors, isEmpty);
      },
    );
  }

  for (final acquisition in _Acquisition.values) {
    test(
      '${acquisition.name} acquisition keeps its provider issue owner',
      () async {
        final reporter = _Reporter();
        final container = await _container(
          _serverFailure,
          reporter,
          acquisition: acquisition,
        );
        await _exercise(
          container,
          acquisition == _Acquisition.business
              ? _Flow.products
              : _Flow.relationship,
        );
        await container.pump();
        expect(reporter.errors, [_serverFailure]);
        expect(reporter.contexts.single.operation, 'provider');
      },
    );
  }

  for (final flow in [_Flow.products, _Flow.eventCreate, _Flow.eventDelete]) {
    test(
      '${flow.name} reports only after its ambiguous retry budget ends',
      () async {
        const error = PdsMutationAmbiguousException(
          retryAfterSeconds: 1,
          diagnosticCause: ApiNetworkError('PRIVATE_NETWORK'),
        );
        final reporter = _Reporter();
        final container = await _container(error, reporter);
        if (flow == _Flow.products) {
          final current = await container.read(
            productsControllerProvider.future,
          );
          expect(
            await container
                .read(productsControllerProvider.notifier)
                .replaceProducts(current.products),
            isFalse,
          );
          expect(
            container.read(productsControllerProvider).requireValue.status,
            ProductsStatus.ambiguous,
          );
        } else {
          final controller = container.read(
            businessEventMutationControllerProvider.notifier,
          );
          expect(
            await (flow == _Flow.eventCreate
                ? controller.create(_draft())
                : controller.delete(_event(), confirmed: true)),
            isFalse,
          );
          expect(
            container.read(businessEventMutationControllerProvider).status,
            EventMutationStatus.ambiguous,
          );
        }
        expect(reporter.errors, [error]);
        expect(reporter.stacks.single.toString(), _stack.toString());
        expect(reporter.contexts.single.outcome, DiagnosticOutcome.terminal);
      },
    );
  }

  test(
    'a consumed product failure serializes one safe correlated issue',
    () async {
      final transport = SerializedTransport();
      await Sentry.init((options) {
        configureDiagnosticOptions(options);
        options
          ..dsn = 'https://public@example.invalid/1'
          ..transport = transport;
      });
      addTearDown(Sentry.close);
      final container = await _container(
        _serverFailure,
        const SentryErrorReporter(),
      );
      await _exercise(container, _Flow.products);
      await transport.issueReceived.future.timeout(const Duration(seconds: 5));
      final payload = transport.payloads.join();
      expect(payload, contains('ApiServerError'));
      expect(payload, contains('nativeOperation'));
      expect(payload, contains(_requestId));
      expect(payload, contains('products.save'));
      expect(payload, contains('operationAccountDid'));
      expect(payload, isNot(contains('PRIVATE_')));
      expect(payload, isNot(contains('did:plc:target')));
      expect(
        transport.payloads.where((value) => value.contains('"exception"')),
        hasLength(1),
      );
    },
  );
}

enum _Flow {
  products,
  eventCreate,
  eventDelete,
  ownerEventsMore,
  ownerEventsRefresh,
  profileEventsMore,
  profileEventsRefresh,
  guestEventsMore,
  relationship,
  accountPreferences,
  preferences,
}

enum _Acquisition { business, relationship }

String _operation(_Flow flow) => switch (flow) {
  _Flow.products => 'products.save',
  _Flow.eventCreate => 'events.create',
  _Flow.eventDelete => 'events.delete',
  _Flow.ownerEventsMore => 'events.owner_load_more',
  _Flow.ownerEventsRefresh => 'events.owner_refresh',
  _Flow.profileEventsMore ||
  _Flow.guestEventsMore => 'events.profile_load_more',
  _Flow.profileEventsRefresh => 'events.profile_refresh',
  _Flow.relationship => 'relationship.mute',
  _Flow.accountPreferences ||
  _Flow.preferences => 'notifications.preferences_patch',
};

Future<ProviderContainer> _container(
  Object error,
  ErrorReporter reporter, {
  bool guest = false,
  _Acquisition? acquisition,
  NotificationNewnessRepository? newness,
}) async {
  final registry = guest
      ? SessionRegistry.empty()
      : SessionRegistry.empty().upsertAndActivate(
          token: 'PRIVATE_SESSION',
          did: 'did:plc:alice',
          handle: 'alice.test',
        );
  final profile = Profile(
    did: 'did:plc:alice',
    handle: 'alice.test',
    crafts: const [],
    accountType: AccountType.business,
    business: BusinessProfile(cid: 'bafy-current', tagline: 'PRIVATE_BUSINESS'),
  );
  final preferences = _Preferences(error);
  final container = ProviderContainer.test(
    retry: appProviderRetry,
    observers: [ProviderLogger(reporter: reporter)],
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        InMemorySessionRegistryStorage(registry),
      ),
      errorReporterProvider.overrideWithValue(reporter),
      activeAccountIdentityProvider.overrideWith(
        (ref) async => ActiveAccountIdentity(
          lease: registry.activeLease!.session,
          profile: profile,
        ),
      ),
      if (acquisition == _Acquisition.business)
        businessRepositoryProvider.overrideWith((ref) => _fail(error))
      else
        businessRepositoryProvider.overrideWithValue(_Business(error)),
      businessTimeZoneServiceProvider.overrideWithValue(
        BusinessTimeZoneService.initialized(),
      ),
      pdsMutationDelayProvider.overrideWithValue((_) async {}),
      pdsMutationJitterProvider.overrideWithValue((_) => 0),
      accountRelationshipRepositoryProvider.overrideWith(
        (ref, account) async => acquisition == _Acquisition.relationship
            ? _fail(error)
            : FakeProfileRepository(onMute: (_) => _fail(error)),
      ),
      notificationPreferencesRepositoryProvider.overrideWithValue(preferences),
      accountNotificationPreferencesRepositoryProvider.overrideWith(
        (ref, account) async => preferences,
      ),
      accountNotificationNewnessRepositoryProvider.overrideWith(
        (ref, account) async => newness ?? _Newness(error),
      ),
      notificationNewnessRepositoryProvider.overrideWithValue(
        newness ?? _Newness(error),
      ),
    ],
  );
  await container.read(sessionRegistryProvider.future);
  return container;
}

Future<void> _exercise(ProviderContainer container, _Flow flow) async {
  switch (flow) {
    case _Flow.products:
      final state = await container.read(productsControllerProvider.future);
      expect(
        await container
            .read(productsControllerProvider.notifier)
            .replaceProducts(state.products),
        isFalse,
      );
      expect(
        container.read(productsControllerProvider).requireValue.status,
        ProductsStatus.error,
      );
    case _Flow.eventCreate:
      expect(
        await container
            .read(businessEventMutationControllerProvider.notifier)
            .create(_draft()),
        isFalse,
      );
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.error,
      );
    case _Flow.eventDelete:
      expect(
        await container
            .read(businessEventMutationControllerProvider.notifier)
            .delete(_event(), confirmed: true),
        isFalse,
      );
      expect(
        container.read(businessEventMutationControllerProvider).status,
        EventMutationStatus.error,
      );
    case _Flow.ownerEventsMore || _Flow.ownerEventsRefresh:
      final provider = ownerBusinessEventsProvider(OwnerEventFilter.upcoming);
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(provider.future);
      final controller = container.read(provider.notifier);
      if (flow == _Flow.ownerEventsMore) {
        await controller.loadMore();
      } else {
        await controller.refresh();
      }
      expect(container.read(provider).hasError, isFalse);
      expect(
        flow == _Flow.ownerEventsMore
            ? container.read(provider).requireValue.incrementalError
            : container.read(provider).requireValue.refreshError,
        isNotNull,
      );
    case _Flow.profileEventsMore ||
        _Flow.profileEventsRefresh ||
        _Flow.guestEventsMore:
      final provider = profileBusinessEventsProvider(
        ProfileBusinessEventsTarget(
          account: flow == _Flow.guestEventsMore ? null : _account,
          owner: AtIdentifier.parse('did:plc:target'),
        ),
      );
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      await container.read(provider.future);
      final controller = container.read(provider.notifier);
      if (flow == _Flow.profileEventsRefresh) {
        await controller.refresh();
      } else {
        await controller.loadMore();
      }
      expect(container.read(provider).hasError, isFalse);
      expect(
        flow == _Flow.profileEventsRefresh
            ? container.read(provider).requireValue.refreshError
            : container.read(provider).requireValue.incrementalError,
        isNotNull,
      );
    case _Flow.relationship:
      final provider = profileRelationshipProvider(
        _account,
        Did.parse('did:plc:target'),
      );
      await container
          .read(provider.notifier)
          .mutate(ProfileRelationshipAction.mute);
      expect(container.read(provider).lastError, isNotNull);
    case _Flow.accountPreferences || _Flow.preferences:
      final provider = flow == _Flow.accountPreferences
          ? accountNotificationPreferencesProvider(_account)
          : notificationPreferencesProvider;
      await container.read(provider.future);
      final result = flow == _Flow.accountPreferences
          ? await container
                .read(accountNotificationPreferencesProvider(_account).notifier)
                .setPushEnabled(NotificationCategory.like, value: false)
          : await container
                .read(notificationPreferencesProvider.notifier)
                .setPushEnabled(NotificationCategory.like, value: false);
      expect(result, isFalse);
      expect(
        container
            .read(provider)
            .requireValue
            .known[NotificationCategory.like]!
            .pushEnabled,
        isTrue,
      );
  }
}

Never _fail(Object error) => Error.throwWithStackTrace(error, _stack);

BusinessEventDraft _draft() => BusinessEventDraft(
  name: 'PRIVATE_DRAFT_NAME',
  startsAt: DateTime(2030, 9, 5, 10),
  endsAt: DateTime(2030, 9, 5, 12),
  roles: const ['vendor'],
  mode: 'in-person',
  status: 'scheduled',
  timeZone: 'UTC',
  isAllDay: false,
);

BusinessEvent _event() => BusinessEvent(
  did: 'did:plc:alice',
  rkey: 'event',
  uri: 'at://did:plc:alice/social.craftsky.business.event/event',
  cid: 'bafy-current',
  name: 'PRIVATE_EVENT_NAME',
  startsAt: DateTime.utc(2030, 9, 5, 10),
  endsAt: DateTime.utc(2030, 9, 5, 12),
  roles: const [],
  status: const BusinessOpenValue(value: 'scheduled', known: true),
  isAllDay: false,
  createdAt: DateTime.utc(2026),
  past: false,
  publicSuppressionReasons: const [],
  upcomingExclusionReasons: const [],
);

final class _Business extends Fake implements BusinessRepository {
  new(this.error);
  final Object error;
  int ownerReads = 0;
  int profileReads = 0;
  @override
  Future<RecordMutationResult> putBusinessProfile(
    Map<String, dynamic> body, {
    required String operationKey,
    required Cid? expectedCid,
  }) async => _fail(error);
  @override
  Future<RecordMutationResult> createEvent(
    BusinessEventDraft draft, {
    required String operationKey,
  }) async => _fail(error);
  @override
  Future<void> deleteEvent(
    Did owner,
    RecordKey rkey,
    Cid expectedCid, {
    required String operationKey,
  }) async => _fail(error);
  @override
  Future<BusinessEventPage> listOwnerEvents(
    OwnerEventFilter filter, {
    String? cursor,
    int limit = 20,
  }) async => ownerReads++ == 0
      ? const BusinessEventPage(items: [], cursor: 'PRIVATE_CURSOR')
      : _fail(error);
  @override
  Future<BusinessEventPage> listProfileEvents(
    AtIdentifier owner, {
    String? cursor,
    int limit = 10,
  }) async => profileReads++ == 0
      ? const BusinessEventPage(items: [], cursor: 'PRIVATE_CURSOR')
      : _fail(error);
}

final class _Preferences implements NotificationPreferencesRepository {
  new(this.error);
  final Object error;
  @override
  Future<NotificationPreferences> load() async => NotificationPreferences(
    known: {
      for (final category in NotificationCategory.preferenceValues)
        category: const NotificationPreference(
          scope: NotificationPreferenceScope.everyone,
          pushEnabled: true,
        ),
    },
    unknown: const {},
  );
  @override
  Future<NotificationPreferences> patch(
    NotificationPreferencePatch patch,
  ) async => _fail(error);
}

final class _Newness implements NotificationNewnessRepository {
  new(this.error);
  final Object error;
  @override
  Future<int> count() async => 0;
  @override
  Future<void> markSeen() async => _fail(error);
}

final class _RefreshFails implements NotificationNewnessRepository {
  new(this.error);
  final Object error;
  int reads = 0;
  @override
  Future<int> count() async => reads++ == 0 ? 0 : _fail(error);
  @override
  Future<void> markSeen() async {}
}

final class _Reporter implements ErrorReporter {
  final errors = <Object>[];
  final stacks = <StackTrace?>[];
  final contexts = <ReportContext>[];
  @override
  bool get enabled => true;
  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    errors.add(error);
    stacks.add(stackTrace);
    contexts.add(context);
    return null;
  }
}
