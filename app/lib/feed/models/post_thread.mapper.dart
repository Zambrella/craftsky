// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post_thread.dart';

class PostThreadMapper extends ClassMapperBase<PostThread> {
  PostThreadMapper._();

  static PostThreadMapper? _instance;
  static PostThreadMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostThreadMapper._());
      PostMapper.ensureInitialized();
      PostThreadMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostThread';

  static Post _$post(PostThread v) => v.post;
  static const Field<PostThread, Post> _f$post = Field('post', _$post);
  static List<PostThread> _$replies(PostThread v) => v.replies;
  static const Field<PostThread, List<PostThread>> _f$replies = Field(
    'replies',
    _$replies,
  );
  static bool _$truncated(PostThread v) => v.truncated;
  static const Field<PostThread, bool> _f$truncated = Field(
    'truncated',
    _$truncated,
  );

  @override
  final MappableFields<PostThread> fields = const {
    #post: _f$post,
    #replies: _f$replies,
    #truncated: _f$truncated,
  };

  static PostThread _instantiate(DecodingData data) {
    return PostThread(
      post: data.dec(_f$post),
      replies: data.dec(_f$replies),
      truncated: data.dec(_f$truncated),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostThread fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostThread>(map);
  }

  static PostThread fromJson(String json) {
    return ensureInitialized().decodeJson<PostThread>(json);
  }
}

mixin PostThreadMappable {
  String toJson() {
    return PostThreadMapper.ensureInitialized().encodeJson<PostThread>(
      this as PostThread,
    );
  }

  Map<String, dynamic> toMap() {
    return PostThreadMapper.ensureInitialized().encodeMap<PostThread>(
      this as PostThread,
    );
  }

  PostThreadCopyWith<PostThread, PostThread, PostThread> get copyWith =>
      _PostThreadCopyWithImpl<PostThread, PostThread>(
        this as PostThread,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostThreadMapper.ensureInitialized().stringifyValue(
      this as PostThread,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostThreadMapper.ensureInitialized().equalsValue(
      this as PostThread,
      other,
    );
  }

  @override
  int get hashCode {
    return PostThreadMapper.ensureInitialized().hashValue(this as PostThread);
  }
}

extension PostThreadValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostThread, $Out> {
  PostThreadCopyWith<$R, PostThread, $Out> get $asPostThread =>
      $base.as((v, t, t2) => _PostThreadCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostThreadCopyWith<$R, $In extends PostThread, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  ListCopyWith<$R, PostThread, PostThreadCopyWith<$R, PostThread, PostThread>>
  get replies;
  $R call({Post? post, List<PostThread>? replies, bool? truncated});
  PostThreadCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostThreadCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostThread, $Out>
    implements PostThreadCopyWith<$R, PostThread, $Out> {
  _PostThreadCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostThread> $mapper =
      PostThreadMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  ListCopyWith<$R, PostThread, PostThreadCopyWith<$R, PostThread, PostThread>>
  get replies => ListCopyWith(
    $value.replies,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(replies: v),
  );
  @override
  $R call({Post? post, List<PostThread>? replies, bool? truncated}) => $apply(
    FieldCopyWithData({
      if (post != null) #post: post,
      if (replies != null) #replies: replies,
      if (truncated != null) #truncated: truncated,
    }),
  );
  @override
  PostThread $make(CopyWithData data) => PostThread(
    post: data.get(#post, or: $value.post),
    replies: data.get(#replies, or: $value.replies),
    truncated: data.get(#truncated, or: $value.truncated),
  );

  @override
  PostThreadCopyWith<$R2, PostThread, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostThreadCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

