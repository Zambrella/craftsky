bool matchesDeletionConfirmationHandle({
  required String requiredHandle,
  required String input,
}) => requiredHandle.isNotEmpty && input == requiredHandle;
