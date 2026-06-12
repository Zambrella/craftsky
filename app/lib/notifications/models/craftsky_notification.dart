import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';

enum CraftskyNotificationType { follow, like, repost, reply, mention }

sealed class CraftskyNotification {
  const CraftskyNotification({
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.actor,
    required this.createdAt,
    required this.indexedAt,
  });

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
  final NotificationActor actor;
  final DateTime createdAt;
  final DateTime indexedAt;

  CraftskyNotificationType get type;

  static CraftskyNotification fromMap(Map<String, dynamic> map) {
    final type = map['type'] as String;
    final common = NotificationCommon.fromMap(map);
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
      _ => throw FormatException('unknown notification type: $type'),
    };
  }
}

final class FollowNotification extends CraftskyNotification {
  FollowNotification(NotificationCommon common)
    : super(
        uri: common.uri,
        cid: common.cid,
        rkey: common.rkey,
        actor: common.actor,
        createdAt: common.createdAt,
        indexedAt: common.indexedAt,
      );

  @override
  CraftskyNotificationType get type => CraftskyNotificationType.follow;
}

sealed class SubjectPostNotification extends CraftskyNotification {
  SubjectPostNotification(
    NotificationCommon common, {
    required this.subjectPost,
  }) : super(
         uri: common.uri,
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
  CraftskyNotificationType get type => CraftskyNotificationType.like;
}

final class RepostNotification extends SubjectPostNotification {
  RepostNotification(super.common, {required super.subjectPost});

  @override
  CraftskyNotificationType get type => CraftskyNotificationType.repost;
}

final class ReplyNotification extends SubjectPostNotification {
  ReplyNotification(super.common, {required super.subjectPost, this.reply});

  final NotificationReplyRef? reply;

  @override
  CraftskyNotificationType get type => CraftskyNotificationType.reply;
}

final class MentionNotification extends SubjectPostNotification {
  MentionNotification(super.common, {required super.subjectPost});

  @override
  CraftskyNotificationType get type => CraftskyNotificationType.mention;
}

final class NotificationActor {
  NotificationActor({
    required String did,
    required String handle,
    this.displayName,
    String? avatarCid,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle),
       avatarCid = avatarCid == null ? null : Cid.parse(avatarCid);

  factory NotificationActor.fromMap(Map<String, dynamic> map) =>
      NotificationActor(
        did: map['did'] as String,
        handle: map['handle'] as String,
        displayName: map['displayName'] as String?,
        avatarCid: map['avatarCid'] as String?,
      );

  final Did did;
  final Handle handle;
  final String? displayName;
  final Cid? avatarCid;

  String get displayLabel => displayName ?? handle.toString();
}

final class NotificationReplyRef {
  NotificationReplyRef({
    required String uri,
    required String cid,
    required String rkey,
  }) : uri = AtUri.parse(uri),
       cid = Cid.parse(cid),
       rkey = RecordKey.parse(rkey);

  factory NotificationReplyRef.fromMap(Map<String, dynamic> map) =>
      NotificationReplyRef(
        uri: map['uri'] as String,
        cid: map['cid'] as String,
        rkey: map['rkey'] as String,
      );

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
}

final class NotificationCommon {
  NotificationCommon({
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.actor,
    required this.createdAt,
    required this.indexedAt,
  });

  factory NotificationCommon.fromMap(Map<String, dynamic> map) =>
      NotificationCommon(
        uri: AtUri.parse(map['uri'] as String),
        cid: Cid.parse(map['cid'] as String),
        rkey: RecordKey.parse(map['rkey'] as String),
        actor: NotificationActor.fromMap(map['actor'] as Map<String, dynamic>),
        createdAt: DateTime.parse(map['createdAt'] as String),
        indexedAt: DateTime.parse(map['indexedAt'] as String),
      );

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
  final NotificationActor actor;
  final DateTime createdAt;
  final DateTime indexedAt;
}
