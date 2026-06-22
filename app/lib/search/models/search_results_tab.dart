enum SearchResultsTab {
  posts,
  projects,
  profiles,
  tags;

  String get wireValue => name;

  static SearchResultsTab fromWire(String? value) {
    return SearchResultsTab.values.firstWhere(
      (tab) => tab.wireValue == value,
      orElse: () => SearchResultsTab.posts,
    );
  }
}
