const maxVideoDraftSourceBytesPerAccount = 1000000000;

final class VideoDraftQuotaPlan {
  const new({
    required this.allowed,
    required this.resultingSourceBytes,
  });

  final bool allowed;
  final int resultingSourceBytes;
  List<String> get evictedDraftIds => const [];
}

final class VideoDraftQuota {
  const new();

  bool canSave({
    required int existingSourceBytes,
    required int replacedSourceBytes,
    required int newSourceBytes,
  }) => plan(
    existingSourceBytes: existingSourceBytes,
    replacedSourceBytes: replacedSourceBytes,
    newSourceBytes: newSourceBytes,
  ).allowed;

  VideoDraftQuotaPlan plan({
    required int existingSourceBytes,
    required int replacedSourceBytes,
    required int newSourceBytes,
  }) {
    final resulting =
        existingSourceBytes - replacedSourceBytes + newSourceBytes;
    return VideoDraftQuotaPlan(
      allowed:
          resulting >= 0 && resulting <= maxVideoDraftSourceBytesPerAccount,
      resultingSourceBytes: resulting,
    );
  }
}
