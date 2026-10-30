// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'chat_providers.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(accountChatRepository)
final accountChatRepositoryProvider = AccountChatRepositoryFamily._();

final class AccountChatRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatRepository>,
          ChatRepository,
          FutureOr<ChatRepository>
        >
    with $FutureModifier<ChatRepository>, $FutureProvider<ChatRepository> {
  AccountChatRepositoryProvider._({
    required AccountChatRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountChatRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountChatRepositoryHash();

  @override
  String toString() {
    return r'accountChatRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountChatRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountChatRepositoryProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountChatRepositoryHash() =>
    r'6afbd582ff368a87c52fe90dcd2226c50954919a';

final class AccountChatRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ChatRepository>, AccountKey> {
  AccountChatRepositoryFamily._()
    : super(
        retry: null,
        name: r'accountChatRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountChatRepositoryProvider call(AccountKey account) =>
      AccountChatRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'accountChatRepositoryProvider';
}

@ProviderFor(MessagingStatus)
final chatStatusProvider = MessagingStatusFamily._();

final class MessagingStatusProvider
    extends $AsyncNotifierProvider<MessagingStatus, ChatStatus> {
  MessagingStatusProvider._({
    required MessagingStatusFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'chatStatusProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$messagingStatusHash();

  @override
  String toString() {
    return r'chatStatusProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  MessagingStatus create() => MessagingStatus();

  @override
  bool operator ==(Object other) {
    return other is MessagingStatusProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$messagingStatusHash() => r'41344aac656094870099c87645b8e4d417abd843';

final class MessagingStatusFamily extends $Family
    with
        $ClassFamilyOverride<
          MessagingStatus,
          AsyncValue<ChatStatus>,
          ChatStatus,
          FutureOr<ChatStatus>,
          AccountKey
        > {
  MessagingStatusFamily._()
    : super(
        retry: null,
        name: r'chatStatusProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  MessagingStatusProvider call(AccountKey account) =>
      MessagingStatusProvider._(argument: account, from: this);

  @override
  String toString() => r'chatStatusProvider';
}

abstract class _$MessagingStatus extends $AsyncNotifier<ChatStatus> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<ChatStatus> build(AccountKey account);
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<ChatStatus>, ChatStatus>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<ChatStatus>, ChatStatus>,
              AsyncValue<ChatStatus>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, () => build(_$args));
  }
}

@ProviderFor(chatUnreadCounts)
final chatUnreadCountsProvider = ChatUnreadCountsFamily._();

final class ChatUnreadCountsProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatUnreadCounts>,
          ChatUnreadCounts,
          FutureOr<ChatUnreadCounts>
        >
    with $FutureModifier<ChatUnreadCounts>, $FutureProvider<ChatUnreadCounts> {
  ChatUnreadCountsProvider._({
    required ChatUnreadCountsFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'chatUnreadCountsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatUnreadCountsHash();

  @override
  String toString() {
    return r'chatUnreadCountsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatUnreadCounts> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatUnreadCounts> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return chatUnreadCounts(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatUnreadCountsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatUnreadCountsHash() => r'652821f16e36d016afd9713600c98033a559af71';

final class ChatUnreadCountsFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ChatUnreadCounts>, AccountKey> {
  ChatUnreadCountsFamily._()
    : super(
        retry: null,
        name: r'chatUnreadCountsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatUnreadCountsProvider call(AccountKey account) =>
      ChatUnreadCountsProvider._(argument: account, from: this);

  @override
  String toString() => r'chatUnreadCountsProvider';
}

@ProviderFor(chatConversationController)
final chatConversationControllerProvider = ChatConversationControllerFamily._();

final class ChatConversationControllerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatConversationController>,
          ChatConversationController,
          FutureOr<ChatConversationController>
        >
    with
        $FutureModifier<ChatConversationController>,
        $FutureProvider<ChatConversationController> {
  ChatConversationControllerProvider._({
    required ChatConversationControllerFamily super.from,
    required (AccountKey, ChatConversationId) super.argument,
  }) : super(
         retry: null,
         name: r'chatConversationControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatConversationControllerHash();

  @override
  String toString() {
    return r'chatConversationControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatConversationController> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatConversationController> create(Ref ref) {
    final argument = this.argument as (AccountKey, ChatConversationId);
    return chatConversationController(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatConversationControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatConversationControllerHash() =>
    r'ddb9140e691487ecf48f47153795bdcd3335a481';

final class ChatConversationControllerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatConversationController>,
          (AccountKey, ChatConversationId)
        > {
  ChatConversationControllerFamily._()
    : super(
        retry: null,
        name: r'chatConversationControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatConversationControllerProvider call(
    AccountKey account,
    ChatConversationId id,
  ) =>
      ChatConversationControllerProvider._(argument: (account, id), from: this);

  @override
  String toString() => r'chatConversationControllerProvider';
}

@ProviderFor(chatInboxController)
final chatInboxControllerProvider = ChatInboxControllerFamily._();

final class ChatInboxControllerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatInboxController>,
          ChatInboxController,
          FutureOr<ChatInboxController>
        >
    with
        $FutureModifier<ChatInboxController>,
        $FutureProvider<ChatInboxController> {
  ChatInboxControllerProvider._({
    required ChatInboxControllerFamily super.from,
    required (AccountKey, ChatBucket) super.argument,
  }) : super(
         retry: null,
         name: r'chatInboxControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatInboxControllerHash();

  @override
  String toString() {
    return r'chatInboxControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatInboxController> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatInboxController> create(Ref ref) {
    final argument = this.argument as (AccountKey, ChatBucket);
    return chatInboxController(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatInboxControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatInboxControllerHash() =>
    r'84c7d8da88975fe821fe7b94c1c1f78cf4ffeb7b';

final class ChatInboxControllerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatInboxController>,
          (AccountKey, ChatBucket)
        > {
  ChatInboxControllerFamily._()
    : super(
        retry: null,
        name: r'chatInboxControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatInboxControllerProvider call(AccountKey account, ChatBucket bucket) =>
      ChatInboxControllerProvider._(argument: (account, bucket), from: this);

  @override
  String toString() => r'chatInboxControllerProvider';
}

@ProviderFor(chatSettingsController)
final chatSettingsControllerProvider = ChatSettingsControllerFamily._();

final class ChatSettingsControllerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatSettingsController>,
          ChatSettingsController,
          FutureOr<ChatSettingsController>
        >
    with
        $FutureModifier<ChatSettingsController>,
        $FutureProvider<ChatSettingsController> {
  ChatSettingsControllerProvider._({
    required ChatSettingsControllerFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'chatSettingsControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatSettingsControllerHash();

  @override
  String toString() {
    return r'chatSettingsControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatSettingsController> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatSettingsController> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return chatSettingsController(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatSettingsControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatSettingsControllerHash() =>
    r'5d19e7fc10a99e692c820e13b8aa42229f610851';

final class ChatSettingsControllerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatSettingsController>,
          AccountKey
        > {
  ChatSettingsControllerFamily._()
    : super(
        retry: null,
        name: r'chatSettingsControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatSettingsControllerProvider call(AccountKey account) =>
      ChatSettingsControllerProvider._(argument: account, from: this);

  @override
  String toString() => r'chatSettingsControllerProvider';
}

@ProviderFor(chatRecipientController)
final chatRecipientControllerProvider = ChatRecipientControllerFamily._();

final class ChatRecipientControllerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatRecipientController>,
          ChatRecipientController,
          FutureOr<ChatRecipientController>
        >
    with
        $FutureModifier<ChatRecipientController>,
        $FutureProvider<ChatRecipientController> {
  ChatRecipientControllerProvider._({
    required ChatRecipientControllerFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'chatRecipientControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatRecipientControllerHash();

  @override
  String toString() {
    return r'chatRecipientControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatRecipientController> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatRecipientController> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return chatRecipientController(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatRecipientControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatRecipientControllerHash() =>
    r'04822d907a098e261fdb064a48c2ba860866a107';

final class ChatRecipientControllerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatRecipientController>,
          AccountKey
        > {
  ChatRecipientControllerFamily._()
    : super(
        retry: null,
        name: r'chatRecipientControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatRecipientControllerProvider call(AccountKey account) =>
      ChatRecipientControllerProvider._(argument: account, from: this);

  @override
  String toString() => r'chatRecipientControllerProvider';
}

@ProviderFor(chatReportController)
final chatReportControllerProvider = ChatReportControllerFamily._();

final class ChatReportControllerProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatReportController>,
          ChatReportController,
          FutureOr<ChatReportController>
        >
    with
        $FutureModifier<ChatReportController>,
        $FutureProvider<ChatReportController> {
  ChatReportControllerProvider._({
    required ChatReportControllerFamily super.from,
    required (AccountKey, ChatConversationId, ChatMessageId) super.argument,
  }) : super(
         retry: null,
         name: r'chatReportControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatReportControllerHash();

  @override
  String toString() {
    return r'chatReportControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatReportController> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatReportController> create(Ref ref) {
    final argument =
        this.argument as (AccountKey, ChatConversationId, ChatMessageId);
    return chatReportController(ref, argument.$1, argument.$2, argument.$3);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatReportControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatReportControllerHash() =>
    r'3c5ba2b7d0b582ad32ca00197ca89f64341851a9';

final class ChatReportControllerFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatReportController>,
          (AccountKey, ChatConversationId, ChatMessageId)
        > {
  ChatReportControllerFamily._()
    : super(
        retry: null,
        name: r'chatReportControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatReportControllerProvider call(
    AccountKey account,
    ChatConversationId id,
    ChatMessageId message,
  ) => ChatReportControllerProvider._(
    argument: (account, id, message),
    from: this,
  );

  @override
  String toString() => r'chatReportControllerProvider';
}

@ProviderFor(chatNotificationOpen)
final chatNotificationOpenProvider = ChatNotificationOpenFamily._();

final class ChatNotificationOpenProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatConversationId>,
          ChatConversationId,
          FutureOr<ChatConversationId>
        >
    with
        $FutureModifier<ChatConversationId>,
        $FutureProvider<ChatConversationId> {
  ChatNotificationOpenProvider._({
    required ChatNotificationOpenFamily super.from,
    required (AccountKey, ChatOpenReference) super.argument,
  }) : super(
         retry: null,
         name: r'chatNotificationOpenProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatNotificationOpenHash();

  @override
  String toString() {
    return r'chatNotificationOpenProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatConversationId> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatConversationId> create(Ref ref) {
    final argument = this.argument as (AccountKey, ChatOpenReference);
    return chatNotificationOpen(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatNotificationOpenProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatNotificationOpenHash() =>
    r'ae350a432c7a4e8640d60345b73fe5c9d56ca844';

final class ChatNotificationOpenFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatConversationId>,
          (AccountKey, ChatOpenReference)
        > {
  ChatNotificationOpenFamily._()
    : super(
        retry: null,
        name: r'chatNotificationOpenProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatNotificationOpenProvider call(
    AccountKey account,
    ChatOpenReference reference,
  ) => ChatNotificationOpenProvider._(
    argument: (account, reference),
    from: this,
  );

  @override
  String toString() => r'chatNotificationOpenProvider';
}

@ProviderFor(chatSyncCoordinator)
final chatSyncCoordinatorProvider = ChatSyncCoordinatorFamily._();

final class ChatSyncCoordinatorProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatSyncCoordinator>,
          ChatSyncCoordinator,
          FutureOr<ChatSyncCoordinator>
        >
    with
        $FutureModifier<ChatSyncCoordinator>,
        $FutureProvider<ChatSyncCoordinator> {
  ChatSyncCoordinatorProvider._({
    required ChatSyncCoordinatorFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'chatSyncCoordinatorProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatSyncCoordinatorHash();

  @override
  String toString() {
    return r'chatSyncCoordinatorProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatSyncCoordinator> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatSyncCoordinator> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return chatSyncCoordinator(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatSyncCoordinatorProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatSyncCoordinatorHash() =>
    r'ae0f10160b7fbb4130f0d5944e5e9380b440c2b7';

final class ChatSyncCoordinatorFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ChatSyncCoordinator>, AccountKey> {
  ChatSyncCoordinatorFamily._()
    : super(
        retry: null,
        name: r'chatSyncCoordinatorProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatSyncCoordinatorProvider call(AccountKey account) =>
      ChatSyncCoordinatorProvider._(argument: account, from: this);

  @override
  String toString() => r'chatSyncCoordinatorProvider';
}

@ProviderFor(accountChatPreviewResolver)
final accountChatPreviewResolverProvider = AccountChatPreviewResolverFamily._();

final class AccountChatPreviewResolverProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatPreviewResolver>,
          ChatPreviewResolver,
          FutureOr<ChatPreviewResolver>
        >
    with
        $FutureModifier<ChatPreviewResolver>,
        $FutureProvider<ChatPreviewResolver> {
  AccountChatPreviewResolverProvider._({
    required AccountChatPreviewResolverFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'accountChatPreviewResolverProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountChatPreviewResolverHash();

  @override
  String toString() {
    return r'accountChatPreviewResolverProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatPreviewResolver> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatPreviewResolver> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return accountChatPreviewResolver(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is AccountChatPreviewResolverProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountChatPreviewResolverHash() =>
    r'd1ca98e389ca8e3b231c23fc56cc47f3bc7ef37e';

final class AccountChatPreviewResolverFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ChatPreviewResolver>, AccountKey> {
  AccountChatPreviewResolverFamily._()
    : super(
        retry: null,
        name: r'accountChatPreviewResolverProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountChatPreviewResolverProvider call(AccountKey account) =>
      AccountChatPreviewResolverProvider._(argument: account, from: this);

  @override
  String toString() => r'accountChatPreviewResolverProvider';
}

@ProviderFor(chatPreview)
final chatPreviewProvider = ChatPreviewFamily._();

final class ChatPreviewProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatPreview?>,
          ChatPreview?,
          FutureOr<ChatPreview?>
        >
    with $FutureModifier<ChatPreview?>, $FutureProvider<ChatPreview?> {
  ChatPreviewProvider._({
    required ChatPreviewFamily super.from,
    required (AccountKey, ChatConversationId, ChatMessageId) super.argument,
  }) : super(
         retry: null,
         name: r'chatPreviewProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatPreviewHash();

  @override
  String toString() {
    return r'chatPreviewProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatPreview?> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatPreview?> create(Ref ref) {
    final argument =
        this.argument as (AccountKey, ChatConversationId, ChatMessageId);
    return chatPreview(ref, argument.$1, argument.$2, argument.$3);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatPreviewProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatPreviewHash() => r'9b270b43a367965b7dc333e083971343242775f2';

final class ChatPreviewFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatPreview?>,
          (AccountKey, ChatConversationId, ChatMessageId)
        > {
  ChatPreviewFamily._()
    : super(
        retry: null,
        name: r'chatPreviewProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatPreviewProvider call(
    AccountKey account,
    ChatConversationId id,
    ChatMessageId messageId,
  ) => ChatPreviewProvider._(argument: (account, id, messageId), from: this);

  @override
  String toString() => r'chatPreviewProvider';
}

@ProviderFor(chatRecipientAvailability)
final chatRecipientAvailabilityProvider = ChatRecipientAvailabilityFamily._();

final class ChatRecipientAvailabilityProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatAvailability>,
          ChatAvailability,
          FutureOr<ChatAvailability>
        >
    with $FutureModifier<ChatAvailability>, $FutureProvider<ChatAvailability> {
  ChatRecipientAvailabilityProvider._({
    required ChatRecipientAvailabilityFamily super.from,
    required (AccountKey, ChatRecipientId) super.argument,
  }) : super(
         retry: null,
         name: r'chatRecipientAvailabilityProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$chatRecipientAvailabilityHash();

  @override
  String toString() {
    return r'chatRecipientAvailabilityProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<ChatAvailability> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatAvailability> create(Ref ref) {
    final argument = this.argument as (AccountKey, ChatRecipientId);
    return chatRecipientAvailability(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is ChatRecipientAvailabilityProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$chatRecipientAvailabilityHash() =>
    r'98b34eda74a668822e08a2f5549b5211a1831bf8';

final class ChatRecipientAvailabilityFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<ChatAvailability>,
          (AccountKey, ChatRecipientId)
        > {
  ChatRecipientAvailabilityFamily._()
    : super(
        retry: null,
        name: r'chatRecipientAvailabilityProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ChatRecipientAvailabilityProvider call(
    AccountKey account,
    ChatRecipientId recipient,
  ) => ChatRecipientAvailabilityProvider._(
    argument: (account, recipient),
    from: this,
  );

  @override
  String toString() => r'chatRecipientAvailabilityProvider';
}

@ProviderFor(messagingChatDio)
final messagingChatDioProvider = MessagingChatDioFamily._();

final class MessagingChatDioProvider
    extends $FunctionalProvider<AsyncValue<Dio>, Dio, FutureOr<Dio>>
    with $FutureModifier<Dio>, $FutureProvider<Dio> {
  MessagingChatDioProvider._({
    required MessagingChatDioFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'messagingChatDioProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$messagingChatDioHash();

  @override
  String toString() {
    return r'messagingChatDioProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<Dio> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Dio> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return messagingChatDio(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is MessagingChatDioProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$messagingChatDioHash() => r'b690da2292dae8a187d4ce766e5bb01da412ed52';

final class MessagingChatDioFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<Dio>, AccountKey> {
  MessagingChatDioFamily._()
    : super(
        retry: null,
        name: r'messagingChatDioProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  MessagingChatDioProvider call(AccountKey account) =>
      MessagingChatDioProvider._(argument: account, from: this);

  @override
  String toString() => r'messagingChatDioProvider';
}

@ProviderFor(messagingChatRepository)
final messagingChatRepositoryProvider = MessagingChatRepositoryFamily._();

final class MessagingChatRepositoryProvider
    extends
        $FunctionalProvider<
          AsyncValue<ChatRepository>,
          ChatRepository,
          FutureOr<ChatRepository>
        >
    with $FutureModifier<ChatRepository>, $FutureProvider<ChatRepository> {
  MessagingChatRepositoryProvider._({
    required MessagingChatRepositoryFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'messagingChatRepositoryProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$messagingChatRepositoryHash();

  @override
  String toString() {
    return r'messagingChatRepositoryProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<ChatRepository> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ChatRepository> create(Ref ref) {
    final argument = this.argument as AccountKey;
    return messagingChatRepository(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is MessagingChatRepositoryProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$messagingChatRepositoryHash() =>
    r'c1327d2f3f314ed9afe03b24cd5d952027f5620b';

final class MessagingChatRepositoryFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<ChatRepository>, AccountKey> {
  MessagingChatRepositoryFamily._()
    : super(
        retry: null,
        name: r'messagingChatRepositoryProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  MessagingChatRepositoryProvider call(AccountKey account) =>
      MessagingChatRepositoryProvider._(argument: account, from: this);

  @override
  String toString() => r'messagingChatRepositoryProvider';
}
