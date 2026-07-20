import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
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
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.actor,
    required this.createdAt,
    required this.indexedAt,
  });

  final String id;
  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
  final NotificationActor actor;
  final DateTime createdAt;
  final DateTime indexedAt;

  NotificationCategory get type;

  static CraftskyNotification fromMap(Map<String, dynamic> map) {
    final type = map['type'] as String;
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
}

final class FollowNotification extends CraftskyNotification {
  FollowNotification(NotificationCommon common)
    : super(
        id: common.id,
        uri: common.uri,
        cid: common.cid,
        rkey: common.rkey,
        actor: common.actor,
        createdAt: common.createdAt,
        indexedAt: common.indexedAt,
      );

  @override
  NotificationCategory get type => NotificationCategory.follow;
}

sealed class SubjectPostNotification extends CraftskyNotification {
  SubjectPostNotification(
    NotificationCommon common, {
    required this.subjectPost,
  }) : super(
         uri: common.uri,
         id: common.id,
         cid: common.cid,
         rkey: common.rkey,
         actor: common.actor,
         createdAt: common.createdAt,
         indexedAt: common.indexedAt,
       );

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

final class GenericNotification extends CraftskyNotification {
  GenericNotification(
    NotificationCommon common, {
    required this.originalType,
  }) : super(
         id: common.id,
         uri: common.uri,
         cid: common.cid,
         rkey: common.rkey,
         actor: common.actor,
         createdAt: common.createdAt,
         indexedAt: common.indexedAt,
       );

  final NotificationCategory originalType;

  @override
  NotificationCategory get type => originalType;
}

final class UnavailableNotification extends CraftskyNotification {
  UnavailableNotification(
    NotificationCommon common, {
    required this.originalType,
  }) : super(
         id: common.id,
         uri: common.uri,
         cid: common.cid,
         rkey: common.rkey,
         actor: common.actor,
         createdAt: common.createdAt,
         indexedAt: common.indexedAt,
       );

  final NotificationCategory originalType;

  @override
  NotificationCategory get type => originalType;
}

@MappableClass(
  generateMethods: _notificationValueMethods,
  includeCustomMappers: [DidMapper(), HandleMapper(), CidMapper()],
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
