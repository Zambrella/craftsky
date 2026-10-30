import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'profile_link.mapper.dart';

@MappableClass()
class ProfileLink with ProfileLinkMappable {
  const new({
    required this.url,
    this.uri,
    this.cid,
    this.title,
    this.icon,
    this.iconBlob,
    this.createdAt,
    this.rkey,
  });

  final String url;
  final String? uri;
  final String? cid;
  final String? title;
  final String? icon;
  final UploadedBlob? iconBlob;
  final String? createdAt;
  // Unsaved links receive a TID once, before any save/retry begins.
  final String? rkey;

  String get recordKey => rkey ?? uri!.split('/').last;
  String get label => title?.isNotEmpty == true
      ? title!
      : Uri.parse(url).host.replaceFirst(RegExp(r'^www\.'), '');

  Map<String, dynamic> toInput() => {
    'rkey': recordKey,
    'cid': ?cid,
    'url': url,
    if (title?.isNotEmpty == true) 'title': title,
    'icon': ?iconBlob?.toMap(),
    'createdAt': createdAt,
  };
}
