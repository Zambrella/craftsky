// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post_summary.dart';

class PostSummaryDataMapper extends ClassMapperBase<PostSummaryData> {
  PostSummaryDataMapper._();

  static PostSummaryDataMapper? _instance;
  static PostSummaryDataMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostSummaryDataMapper._());
      PostAuthorMapper.ensureInitialized();
      PostImageMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostSummaryData';

  static PostSummaryState _$state(PostSummaryData v) => v.state;
  static const Field<PostSummaryData, PostSummaryState> _f$state = Field(
    'state',
    _$state,
  );
  static PostAuthor? _$author(PostSummaryData v) => v.author;
  static const Field<PostSummaryData, PostAuthor> _f$author = Field(
    'author',
    _$author,
    opt: true,
  );
  static String? _$text(PostSummaryData v) => v.text;
  static const Field<PostSummaryData, String> _f$text = Field(
    'text',
    _$text,
    opt: true,
  );
  static DateTime? _$createdAt(PostSummaryData v) => v.createdAt;
  static const Field<PostSummaryData, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
    opt: true,
  );
  static String? _$projectTitle(PostSummaryData v) => v.projectTitle;
  static const Field<PostSummaryData, String> _f$projectTitle = Field(
    'projectTitle',
    _$projectTitle,
    opt: true,
  );
  static PostImage? _$image(PostSummaryData v) => v.image;
  static const Field<PostSummaryData, PostImage> _f$image = Field(
    'image',
    _$image,
    opt: true,
  );
  static bool _$revealable(PostSummaryData v) => v.revealable;
  static const Field<PostSummaryData, bool> _f$revealable = Field(
    'revealable',
    _$revealable,
    opt: true,
    def: false,
  );

  @override
  final MappableFields<PostSummaryData> fields = const {
    #state: _f$state,
    #author: _f$author,
    #text: _f$text,
    #createdAt: _f$createdAt,
    #projectTitle: _f$projectTitle,
    #image: _f$image,
    #revealable: _f$revealable,
  };

  static PostSummaryData _instantiate(DecodingData data) {
    return PostSummaryData(
      state: data.dec(_f$state),
      author: data.dec(_f$author),
      text: data.dec(_f$text),
      createdAt: data.dec(_f$createdAt),
      projectTitle: data.dec(_f$projectTitle),
      image: data.dec(_f$image),
      revealable: data.dec(_f$revealable),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin PostSummaryDataMappable {
  PostSummaryDataCopyWith<PostSummaryData, PostSummaryData, PostSummaryData>
  get copyWith =>
      _PostSummaryDataCopyWithImpl<PostSummaryData, PostSummaryData>(
        this as PostSummaryData,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return PostSummaryDataMapper.ensureInitialized().equalsValue(
      this as PostSummaryData,
      other,
    );
  }

  @override
  int get hashCode {
    return PostSummaryDataMapper.ensureInitialized().hashValue(
      this as PostSummaryData,
    );
  }
}

extension PostSummaryDataValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostSummaryData, $Out> {
  PostSummaryDataCopyWith<$R, PostSummaryData, $Out> get $asPostSummaryData =>
      $base.as((v, t, t2) => _PostSummaryDataCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostSummaryDataCopyWith<$R, $In extends PostSummaryData, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor>? get author;
  PostImageCopyWith<$R, PostImage, PostImage>? get image;
  $R call({
    PostSummaryState? state,
    PostAuthor? author,
    String? text,
    DateTime? createdAt,
    String? projectTitle,
    PostImage? image,
    bool? revealable,
  });
  PostSummaryDataCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostSummaryDataCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostSummaryData, $Out>
    implements PostSummaryDataCopyWith<$R, PostSummaryData, $Out> {
  _PostSummaryDataCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostSummaryData> $mapper =
      PostSummaryDataMapper.ensureInitialized();
  @override
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor>? get author =>
      $value.author?.copyWith.$chain((v) => call(author: v));
  @override
  PostImageCopyWith<$R, PostImage, PostImage>? get image =>
      $value.image?.copyWith.$chain((v) => call(image: v));
  @override
  $R call({
    PostSummaryState? state,
    Object? author = $none,
    Object? text = $none,
    Object? createdAt = $none,
    Object? projectTitle = $none,
    Object? image = $none,
    bool? revealable,
  }) => $apply(
    FieldCopyWithData({
      if (state != null) #state: state,
      if (author != $none) #author: author,
      if (text != $none) #text: text,
      if (createdAt != $none) #createdAt: createdAt,
      if (projectTitle != $none) #projectTitle: projectTitle,
      if (image != $none) #image: image,
      if (revealable != null) #revealable: revealable,
    }),
  );
  @override
  PostSummaryData $make(CopyWithData data) => PostSummaryData(
    state: data.get(#state, or: $value.state),
    author: data.get(#author, or: $value.author),
    text: data.get(#text, or: $value.text),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    projectTitle: data.get(#projectTitle, or: $value.projectTitle),
    image: data.get(#image, or: $value.image),
    revealable: data.get(#revealable, or: $value.revealable),
  );

  @override
  PostSummaryDataCopyWith<$R2, PostSummaryData, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostSummaryDataCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

