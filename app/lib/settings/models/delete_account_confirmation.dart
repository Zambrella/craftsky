bool matchesDeletionConfirmationDid({
  required String confirmationDid,
  required String input,
}) => confirmationDid.isNotEmpty && input == confirmationDid;
