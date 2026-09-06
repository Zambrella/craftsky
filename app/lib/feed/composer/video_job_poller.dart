enum VideoPollingStop { completed, failed, canceled, interrupted, expired }

Duration videoPollingDelay(int completedPolls, {Duration? retryAfter}) {
  final backoff = Duration(seconds: (completedPolls + 1).clamp(1, 5));
  return retryAfter != null && retryAfter > backoff ? retryAfter : backoff;
}

bool shouldStopVideoPolling(VideoPollingStop reason) => true;

bool isVideoJobExpired({required DateTime startedAt, required DateTime now}) =>
    !now.isBefore(startedAt.add(const Duration(hours: 1)));
