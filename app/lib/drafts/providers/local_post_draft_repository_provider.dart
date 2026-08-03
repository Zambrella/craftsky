import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/data/file_local_post_draft_repository.dart';
import 'package:craftsky_app/drafts/data/local_post_draft_repository.dart';
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'local_post_draft_repository_provider.g.dart';

@Riverpod(keepAlive: true)
Future<LocalPostDraftRepository> accountLocalPostDraftRepository(
  Ref ref,
  AccountKey account,
) async {
  final documents = await getApplicationDocumentsDirectory();
  return FileLocalPostDraftRepository(
    documentsRoot: documents.path,
    owner: account,
  );
}
