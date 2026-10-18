import 'dart:typed_data';

/// Reads the primary HEIF item's dimensions without decoding its pixels.
/// Only HEVC-based HEIF is accepted; AVIF shares the container, not the codec.
final class HeifImageInfo {
  const new({required this.width, required this.height, required this.rotated});

  factory read(Uint8List bytes) {
    if (!hasHeicBrand(bytes)) {
      throw const FormatException('Unsupported HEIF codec');
    }
    final meta = _boxes(
      bytes,
      0,
      bytes.length,
    ).where((b) => b.type == 'meta').firstOrNull;
    if (meta == null) throw const FormatException('Missing HEIF metadata');
    final children = _boxes(bytes, meta.start + 4, meta.end).toList();
    final primary = children.where((b) => b.type == 'pitm').firstOrNull;
    final iprp = children.where((b) => b.type == 'iprp').firstOrNull;
    if (primary == null || iprp == null || primary.start + 6 > primary.end) {
      throw const FormatException('Missing HEIF primary item');
    }
    final version = bytes[primary.start];
    final primaryId = version == 0
        ? _u16(bytes, primary.start + 4)
        : _u32(bytes, primary.start + 4);
    final propertyBoxes = _boxes(bytes, iprp.start, iprp.end).toList();
    final ipco = propertyBoxes.where((b) => b.type == 'ipco').firstOrNull;
    if (ipco == null) throw const FormatException('Missing HEIF properties');
    final properties = _boxes(bytes, ipco.start, ipco.end).toList();
    final selected = <int>{};
    for (final ipma in propertyBoxes.where((b) => b.type == 'ipma')) {
      var offset = ipma.start;
      if (offset + 8 > ipma.end) {
        throw const FormatException('Invalid HEIF associations');
      }
      final largeIds = bytes[offset] >= 1;
      final largeIndices = bytes[offset + 3] & 1 != 0;
      final entries = _u32(bytes, offset + 4);
      offset += 8;
      for (var entry = 0; entry < entries; entry++) {
        final idSize = largeIds ? 4 : 2;
        if (offset + idSize + 1 > ipma.end) {
          throw const FormatException('Invalid HEIF item');
        }
        final id = largeIds ? _u32(bytes, offset) : _u16(bytes, offset);
        offset += idSize;
        final count = bytes[offset++];
        for (var n = 0; n < count; n++) {
          if (offset + (largeIndices ? 2 : 1) > ipma.end) {
            throw const FormatException('Invalid HEIF property');
          }
          final index = largeIndices
              ? _u16(bytes, offset) & 0x7fff
              : bytes[offset] & 0x7f;
          offset += largeIndices ? 2 : 1;
          if (id == primaryId && index != 0) selected.add(index);
        }
      }
    }
    int? width;
    int? height;
    var rotated = false;
    for (final index in selected) {
      if (index > properties.length) {
        throw const FormatException('Invalid HEIF property index');
      }
      final property = properties[index - 1];
      if (property.type == 'ispe' && property.start + 12 <= property.end) {
        width = _u32(bytes, property.start + 4);
        height = _u32(bytes, property.start + 8);
      } else if (property.type == 'irot' && property.start < property.end) {
        rotated = bytes[property.start] & 1 != 0;
      }
    }
    if (width == null || height == null || width <= 0 || height <= 0) {
      throw const FormatException('Missing HEIF primary dimensions');
    }
    return HeifImageInfo(width: width, height: height, rotated: rotated);
  }

  final int width;
  final int height;
  final bool rotated;

  static bool hasHeicBrand(Uint8List bytes) {
    if (bytes.length < 12 || _type(bytes, 4) != 'ftyp') return false;
    const brands = {'heic', 'heix', 'hevc', 'hevx', 'heim', 'heis'};
    if (brands.contains(_type(bytes, 8))) return true;
    if (bytes.length < 16) return false;
    final end = _u32(bytes, 0).clamp(16, bytes.length);
    for (var offset = 16; offset + 4 <= end; offset += 4) {
      if (brands.contains(_type(bytes, offset))) return true;
    }
    return false;
  }
}

Iterable<({String type, int start, int end})> _boxes(
  Uint8List bytes,
  int start,
  int end,
) sync* {
  if (start > end) throw const FormatException('Invalid HEIF container');
  var offset = start;
  while (offset < end) {
    if (offset + 8 > end) throw const FormatException('Truncated HEIF box');
    var size = _u32(bytes, offset);
    var header = 8;
    if (size == 1) {
      if (offset + 16 > end) throw const FormatException('Truncated HEIF box');
      size = ByteData.sublistView(bytes).getUint64(offset + 8);
      header = 16;
    } else if (size == 0) {
      size = end - offset;
    }
    if (size < header || size > end - offset) {
      throw const FormatException('Invalid HEIF box size');
    }
    yield (
      type: _type(bytes, offset + 4),
      start: offset + header,
      end: offset + size,
    );
    offset += size;
  }
}

int _u16(Uint8List bytes, int offset) {
  _requireBytes(bytes, offset, 2);
  return ByteData.sublistView(bytes).getUint16(offset);
}

int _u32(Uint8List bytes, int offset) {
  _requireBytes(bytes, offset, 4);
  return ByteData.sublistView(bytes).getUint32(offset);
}

String _type(Uint8List bytes, int offset) {
  _requireBytes(bytes, offset, 4);
  return String.fromCharCodes(bytes.sublist(offset, offset + 4));
}

void _requireBytes(Uint8List bytes, int offset, int count) {
  if (offset < 0 || count > bytes.length - offset) {
    throw const FormatException('Truncated HEIF metadata');
  }
}
