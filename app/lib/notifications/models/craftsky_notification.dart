import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'craftsky_notification.mapper.dart';

const int _notificationValueMethods =
    GenerateMethods.decode |
    GenerateMethods.encode |
    GenerateMethods.equals |
    GenerateMethods.copy;

sealed class CraftskyNotification {
  const CraftskyNotification({
    required this.id,
    required this.createdAt,
    required this.indexedAt,
  });

  final String id;
  final DateTime createdAt;
  final DateTime indexedAt;

  NotificationCategory get type;

  static CraftskyNotification fromMap(Map<String, dynamic> map) {
    final type = map['type'];
    if (type is! String) {
      throw const FormatException('invalid_notification_type');
    }
    if (type == 'instagramMatch') return _instagramMatchFromMap(map);
    if (!_socialTypes.contains(type) &&
        (map['actor'] is! Map ||
            map['uri'] is! String ||
            map['cid'] is! String ||
            map['rkey'] is! String)) {
      return GenericSystemNotification(
        SystemNotificationCommon.fromMap(map),
        originalType: NotificationCategory.unknown,
      );
    }
    final common = NotificationCommon.fromMap(map);
    final category = _category(type);
    if (!common.actor.available || map['contentAvailable'] == false) {
      return UnavailableNotification(common, originalType: category);
    }
    return switch (type) {
      'follow' => FollowNotification(common),
      'like' => LikeNotification(
        common,
        subjectPost: PostMapper.fromMap(
          map['subjectPost'] as Map<String, dynamic>,
        ),
      ),
      'repost' => RepostNotification(
        common,
        subjectPost: PostMapper.fromMap(
          map['subjectPost'] as Map<String, dynamic>,
        ),
      ),
      'reply' => ReplyNotification(
        common,
        subjectPost: PostMapper.fromMap(
          map['subjectPost'] as Map<String, dynamic>,
        ),
        reply: map['reply'] == null
            ? null
            : NotificationReplyRef.fromMap(
                map['reply'] as Map<String, dynamic>,
              ),
      ),
      'mention' => MentionNotification(
        common,
        subjectPost: PostMapper.fromMap(
          map['subjectPost'] as Map<String, dynamic>,
        ),
      ),
      'quote' => QuoteNotification(
        common,
        subjectPost: PostMapper.fromMap(
          map['subjectPost'] as Map<String, dynamic>,
        ),
      ),
      'everythingElse' => GenericNotification(
        common,
        originalType: NotificationCategory.everythingElse,
      ),
      _ => GenericNotification(
        common,
        originalType: NotificationCategory.unknown,
      ),
    };
  }

  static NotificationCategory _category(String value) =>
      NotificationCategory.fromWireValue(value);

  static const _socialTypes = {
    'follow',
    'like',
    'repost',
    'reply',
    'mention',
    'quote',
    'everythingElse',
  };

  static CraftskyNotification _instagramMatchFromMap(
    Map<String, dynamic> map,
  ) {
    if (map['actor'] case final Map<String, dynamic> actorMap) {
      final common = ActorNotificationCommon.fromMap({
        ...map,
        'actor': actorMap,
      });
      if (common.actor.available) {
        return InstagramMatchNotification(common);
      }
    }
    return GenericSystemNotification(
      SystemNotificationCommon.fromMap(map),
      originalType: NotificationCategory.instagramMatch,
    );
  }
}

sealed class ActorNotification extends CraftskyNotification {
  ActorNotification({
    required super.id,
    required this.actor,
    required super.createdAt,
    required super.indexedAt,
  });

  final NotificationActor actor;
}

sealed class SocialNotification extends ActorNotification {
  SocialNotification(NotificationCommon common)
    : uri = common.uri,
      cid = common.cid,
      rkey = common.rkey,
      super(
        id: common.id,
        actor: common.actor,
        createdAt: common.createdAt,
        indexedAt: common.indexedAt,
      );

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
}

final class FollowNotification extends SocialNotification {
  FollowNotification(super.common);

  @override
  NotificationCategory get type => NotificationCategory.follow;
}

sealed class SubjectPostNotification extends SocialNotification {
  SubjectPostNotification(
    super.common, {
    required this.subjectPost,
  });

  final Post subjectPost;
}

final class LikeNotification extends SubjectPostNotification {
  LikeNotification(super.common, {required super.subjectPost});

  @override
  NotificationCategory get type => NotificationCategory.like;
}

final class RepostNotification extends SubjectPostNotification {
  RepostNotification(super.common, {required super.subjectPost});

  @override
  NotificationCategory get type => NotificationCategory.repost;
}

final class ReplyNotification extends SubjectPostNotification {
  ReplyNotification(super.common, {required super.subjectPost, this.reply});

  final NotificationReplyRef? reply;

  @override
  NotificationCategory get type => NotificationCategory.reply;
}

final class MentionNotification extends SubjectPostNotification {
  MentionNotification(super.common, {required super.subjectPost});

  @override
  NotificationCategory get type => NotificationCategory.mention;
}

final class QuoteNotification extends SubjectPostNotification {
  QuoteNotification(super.common, {required super.subjectPost});

  @override
  NotificationCategory get type => NotificationCategory.quote;
}

final class GenericNotification extends SocialNotification {
  GenericNotification(
    super.common, {
    required this.originalType,
  });

  final NotificationCategory originalType;

  @override
  NotificationCategory get type => originalType;
}

final class UnavailableNotification extends SocialNotification {
  UnavailableNotification(
    super.common, {
    required this.originalType,
  });

  final NotificationCategory originalType;

  @override
  NotificationCategory get type => originalType;
}

sealed class SystemNotification extends CraftskyNotification {
  SystemNotification(SystemNotificationCommon common)
    : super(
        id: common.id,
        createdAt: common.createdAt,
        indexedAt: common.indexedAt,
      );
}

final class InstagramMatchNotification extends ActorNotification {
  InstagramMatchNotification(ActorNotificationCommon common)
    : super(
        id: common.id,
        actor: common.actor,
        createdAt: common.createdAt,
        indexedAt: common.indexedAt,
      );

  @override
  NotificationCategory get type => NotificationCategory.instagramMatch;
}

@MappableClass(
  generateMethods:
      GenerateMethods.decode | GenerateMethods.copy | GenerateMethods.equals,
)
final class ActorNotificationCommon with ActorNotificationCommonMappable {
  const ActorNotificationCommon({
    required this.id,
    required this.actor,
    required this.createdAt,
    required this.indexedAt,
  });

  factory ActorNotificationCommon.fromMap(Map<String, dynamic> map) =>
      ActorNotificationCommonMapper.fromMap(map);

  final String id;
  final NotificationActor actor;
  final DateTime createdAt;
  final DateTime indexedAt;
}

final class GenericSystemNotification extends SystemNotification {
  GenericSystemNotification(
    super.common, {
    required this.originalType,
  });

  final NotificationCategory originalType;

  @override
  NotificationCategory get type => originalType;
}

@MappableClass(
  generateMethods:
      GenerateMethods.decode | GenerateMethods.copy | GenerateMethods.equals,
)
final class SystemNotificationCommon with SystemNotificationCommonMappable {
  const SystemNotificationCommon({
    required this.id,
    required this.createdAt,
    required this.indexedAt,
  });

  factory SystemNotificationCommon.fromMap(Map<String, dynamic> map) =>
      SystemNotificationCommonMapper.fromMap(map);

  final String id;
  final DateTime createdAt;
  final DateTime indexedAt;
}

@MappableClass(
  generateMethods: _notificationValueMethods,
  includeCustomMappers: [
    DidMapper(),
    HandleMapper(),
    CidMapper(),
    ProfileCustomisationMapper(),
  ],
)
final class NotificationActor with NotificationActorMappable {
  const NotificationActor({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatar,
    this.avatarCid,
    this.viewerIsFollowing = false,
    this.available = true,
    this.muted,
    this.blocking,
    this.blockedBy,
    this.customisation = ProfileCustomisation.defaults,
  });

  factory NotificationActor.fromMap(Map<String, dynamic> map) =>
      NotificationActorMapper.fromMap({
        ...map,
        'did': (map['did'] as String?)?.isNotEmpty == true
            ? map['did']
            : 'did:plc:unavailable',
        'handle': (map['handle'] as String?)?.isNotEmpty == true
            ? map['handle']
            : 'unavailable.invalid',
        'available': map['available'] as bool? ?? true,
      });

  final Did did;
  final Handle handle;
  final String? displayName;
  final String? avatar;
  final Cid? avatarCid;
  final bool viewerIsFollowing;
  final bool available;
  final bool? muted;
  final bool? blocking;
  final bool? blockedBy;
  final ProfileCustomisation customisation;

  bool get hasViewerState =>
      muted != null || blocking != null || blockedBy != null;

  String get displayLabel =>
      available ? displayName ?? handle.toString() : 'Unavailable account';

  /// Prefer the AppView's display-ready URL, while supporting notification
  /// responses from an older local AppView that expose only the public CID.
  String? get displayAvatarUrl {
    if (avatar case final value? when value.isNotEmpty) return value;
    if (avatarCid case final cid? when !cid.startsWith('devmedia:')) {
      return 'https://cdn.bsky.app/img/avatar/plain/$did/$cid@jpeg';
    }
    return null;
  }
}

@MappableClass(
  generateMethods: _notificationValueMethods,
  includeCustomMappers: [AtUriMapper(), CidMapper(), RecordKeyMapper()],
)
final class NotificationReplyRef with NotificationReplyRefMappable {
  const NotificationReplyRef({
    required this.uri,
    required this.cid,
    required this.rkey,
  });

  factory NotificationReplyRef.fromMap(Map<String, dynamic> map) =>
      NotificationReplyRefMapper.fromMap(map);

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
}

@MappableClass(
  generateMethods: _notificationValueMethods,
  includeCustomMappers: [AtUriMapper(), CidMapper(), RecordKeyMapper()],
)
final class NotificationCommon with NotificationCommonMappable {
  const NotificationCommon({
    required this.id,
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.actor,
    required this.createdAt,
    required this.indexedAt,
  });

  factory NotificationCommon.fromMap(Map<String, dynamic> map) {
    final uri =
        map['uri'] as String? ??
        'at://did:plc:unavailable/social.craftsky.notification/unavailable';
    final actor = NotificationActor.fromMap(
      map['actor'] as Map<String, dynamic>,
    );
    return NotificationCommonMapper.fromMap({
      ...map,
      'id': map['id'] as String? ?? uri,
      'uri': uri,
      'cid': map['cid'] as String? ?? 'unavailable',
      'rkey': map['rkey'] as String? ?? 'unavailable',
      'actor': actor.toMap(),
    });
  }

  final AtUri uri;
  final String id;
  final Cid cid;
  final RecordKey rkey;
  final NotificationActor actor;
  final DateTime createdAt;
  final DateTime indexedAt;
}
