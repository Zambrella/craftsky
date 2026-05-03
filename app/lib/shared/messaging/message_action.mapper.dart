// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'message_action.dart';

class MessageActionMapper extends ClassMapperBase<MessageAction> {
  MessageActionMapper._();

  static MessageActionMapper? _instance;
  static MessageActionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = MessageActionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'MessageAction';

  static String _$label(MessageAction v) => v.label;
  static const Field<MessageAction, String> _f$label = Field('label', _$label);
  static Function _$onPressed(MessageAction v) =>
      (v as dynamic).onPressed as Function;
  static dynamic _arg$onPressed(f) => f<void Function()>();
  static const Field<MessageAction, Function> _f$onPressed = Field(
    'onPressed',
    _$onPressed,
    arg: _arg$onPressed,
  );
  static bool _$dismissOnTap(MessageAction v) => v.dismissOnTap;
  static const Field<MessageAction, bool> _f$dismissOnTap = Field(
    'dismissOnTap',
    _$dismissOnTap,
    opt: true,
    def: true,
  );

  @override
  final MappableFields<MessageAction> fields = const {
    #label: _f$label,
    #onPressed: _f$onPressed,
    #dismissOnTap: _f$dismissOnTap,
  };

  static MessageAction _instantiate(DecodingData data) {
    return MessageAction(
      label: data.dec(_f$label),
      onPressed: data.dec(_f$onPressed),
      dismissOnTap: data.dec(_f$dismissOnTap),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static MessageAction fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<MessageAction>(map);
  }

  static MessageAction fromJson(String json) {
    return ensureInitialized().decodeJson<MessageAction>(json);
  }
}

mixin MessageActionMappable {
  String toJson() {
    return MessageActionMapper.ensureInitialized().encodeJson<MessageAction>(
      this as MessageAction,
    );
  }

  Map<String, dynamic> toMap() {
    return MessageActionMapper.ensureInitialized().encodeMap<MessageAction>(
      this as MessageAction,
    );
  }

  MessageActionCopyWith<MessageAction, MessageAction, MessageAction>
  get copyWith => _MessageActionCopyWithImpl<MessageAction, MessageAction>(
    this as MessageAction,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return MessageActionMapper.ensureInitialized().stringifyValue(
      this as MessageAction,
    );
  }

  @override
  bool operator ==(Object other) {
    return MessageActionMapper.ensureInitialized().equalsValue(
      this as MessageAction,
      other,
    );
  }

  @override
  int get hashCode {
    return MessageActionMapper.ensureInitialized().hashValue(
      this as MessageAction,
    );
  }
}

extension MessageActionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, MessageAction, $Out> {
  MessageActionCopyWith<$R, MessageAction, $Out> get $asMessageAction =>
      $base.as((v, t, t2) => _MessageActionCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class MessageActionCopyWith<$R, $In extends MessageAction, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? label, void Function()? onPressed, bool? dismissOnTap});
  MessageActionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _MessageActionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, MessageAction, $Out>
    implements MessageActionCopyWith<$R, MessageAction, $Out> {
  _MessageActionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<MessageAction> $mapper =
      MessageActionMapper.ensureInitialized();
  @override
  $R call({String? label, void Function()? onPressed, bool? dismissOnTap}) =>
      $apply(
        FieldCopyWithData({
          if (label != null) #label: label,
          if (onPressed != null) #onPressed: onPressed,
          if (dismissOnTap != null) #dismissOnTap: dismissOnTap,
        }),
      );
  @override
  MessageAction $make(CopyWithData data) => MessageAction(
    label: data.get(#label, or: $value.label),
    onPressed: data.get(#onPressed, or: $value.onPressed),
    dismissOnTap: data.get(#dismissOnTap, or: $value.dismissOnTap),
  );

  @override
  MessageActionCopyWith<$R2, MessageAction, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _MessageActionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

