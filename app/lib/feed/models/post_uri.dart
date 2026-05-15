class PostUriParts {
  const PostUriParts({required this.did, required this.rkey});

  final String did;
  final String rkey;
}

PostUriParts? parseCraftskyPostUri(String uri) {
  const prefix = 'at://';
  const collection = '/social.craftsky.feed.post/';
  if (!uri.startsWith(prefix)) return null;
  final body = uri.substring(prefix.length);
  final collectionStart = body.indexOf(collection);
  if (collectionStart <= 0) return null;
  final did = body.substring(0, collectionStart);
  final rkey = body.substring(collectionStart + collection.length);
  if (did.isEmpty || rkey.isEmpty || rkey.contains('/')) {
    return null;
  }
  return PostUriParts(did: did, rkey: rkey);
}
