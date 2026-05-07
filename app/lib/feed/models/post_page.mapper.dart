// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post_page.dart';

class PostPageMapper extends ClassMapperBase<PostPage> {
  PostPageMapper._();

  static PostPageMapper? _instance;
  static PostPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostPageMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostPage';

  static List<Post> _$items(PostPage v) => v.items;
  static const Field<PostPage, List<Post>> _f$items = Field('items', _$items);
  static String? _$cursor(PostPage v) => v.cursor;
  static const Field<PostPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<PostPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static PostPage _instantiate(DecodingData data) {
    return PostPage(items: data.dec(_f$items), cursor: data.dec(_f$cursor));
  }

  @override
  final Function instantiate = _instantiate;

  static PostPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostPage>(map);
  }

  static PostPage fromJson(String json) {
    return ensureInitialized().decodeJson<PostPage>(json);
  }
}

mixin PostPageMappable {
  String toJson() {
    return PostPageMapper.ensureInitialized().encodeJson<PostPage>(
      this as PostPage,
    );
  }

  Map<String, dynamic> toMap() {
    return PostPageMapper.ensureInitialized().encodeMap<PostPage>(
      this as PostPage,
    );
  }

  PostPageCopyWith<PostPage, PostPage, PostPage> get copyWith =>
      _PostPageCopyWithImpl<PostPage, PostPage>(
        this as PostPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostPageMapper.ensureInitialized().stringifyValue(this as PostPage);
  }

  @override
  bool operator ==(Object other) {
    return PostPageMapper.ensureInitialized().equalsValue(
      this as PostPage,
      other,
    );
  }

  @override
  int get hashCode {
    return PostPageMapper.ensureInitialized().hashValue(this as PostPage);
  }
}

extension PostPageValueCopy<$R, $Out> on ObjectCopyWith<$R, PostPage, $Out> {
  PostPageCopyWith<$R, PostPage, $Out> get $asPostPage =>
      $base.as((v, t, t2) => _PostPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostPageCopyWith<$R, $In extends PostPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items;
  $R call({List<Post>? items, String? cursor});
  PostPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostPage, $Out>
    implements PostPageCopyWith<$R, PostPage, $Out> {
  _PostPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostPage> $mapper =
      PostPageMapper.ensureInitialized();
  @override
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items =>
      ListCopyWith(
        $value.items,
        (v, t) => v.copyWith.$chain(t),
        (v) => call(items: v),
      );
  @override
  $R call({List<Post>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  PostPage $make(CopyWithData data) => PostPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  PostPageCopyWith<$R2, PostPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

