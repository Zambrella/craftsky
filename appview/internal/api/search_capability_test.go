package api_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/relationships"
)

type profileSearchCapability struct{}

func (profileSearchCapability) SearchProfiles(
	context.Context,
	string,
	api.ProfileSearchRequest,
) ([]api.ProfileSearchRow, string, error) {
	return nil, "", nil
}

type hashtagSearchCapability struct{}

func (hashtagSearchCapability) SearchHashtags(
	context.Context,
	api.HashtagSearchRequest,
	time.Time,
) ([]api.HashtagSearchResult, string, error) {
	return nil, "", nil
}

func TestSearchHashtagsHandlerAcceptsHashtagSearchCapability(t *testing.T) {
	if handler := api.SearchHashtagsHandler(hashtagSearchCapability{}, nil); handler == nil {
		t.Fatal("SearchHashtagsHandler returned nil")
	}
}

type searchSuggestionsCapability struct {
	profileSearchCapability
	hashtagSearchCapability
}

func TestSearchSuggestionsHandlerAcceptsSuggestionCapabilities(t *testing.T) {
	if handler := api.SearchSuggestionsHandler(searchSuggestionsCapability{}, nil); handler == nil {
		t.Fatal("SearchSuggestionsHandler returned nil")
	}
}

type searchPostHydrationCapability struct{}

func (searchPostHydrationCapability) EngagementSummaries(
	context.Context,
	string,
	[]string,
) (map[string]api.EngagementSummary, error) {
	return map[string]api.EngagementSummary{}, nil
}

func (searchPostHydrationCapability) QuoteViewRows(
	context.Context,
	[]api.ResponseStrongRef,
) (map[string]*api.QuoteViewRow, error) {
	return map[string]*api.QuoteViewRow{}, nil
}

func (searchPostHydrationCapability) RelationshipStates(
	context.Context,
	syntax.DID,
	[]syntax.DID,
) (map[syntax.DID]relationships.State, error) {
	return map[syntax.DID]relationships.State{}, nil
}

func (searchPostHydrationCapability) BlockedPairs(
	context.Context,
	[]api.RelationshipPair,
) (map[api.RelationshipPair]bool, error) {
	return map[api.RelationshipPair]bool{}, nil
}

type postSearchCapability struct {
	searchPostHydrationCapability
}

func (postSearchCapability) SearchPostsWithLanguages(
	context.Context,
	string,
	[]string,
	api.PostSearchRequest,
	time.Time,
) ([]api.SearchPostRow, string, error) {
	return nil, "", nil
}

func TestSearchPostsHandlerAcceptsPostSearchAndHydrationCapabilities(t *testing.T) {
	if handler := api.SearchPostsHandler(postSearchCapability{}, nil, nil); handler == nil {
		t.Fatal("SearchPostsHandler returned nil")
	}
}

type projectSearchCapability struct {
	searchPostHydrationCapability
}

func (projectSearchCapability) SearchProjectsWithLanguages(
	context.Context,
	string,
	[]string,
	api.ProjectSearchRequest,
	time.Time,
) ([]api.SearchPostRow, string, error) {
	return nil, "", nil
}

func TestSearchProjectsHandlerAcceptsProjectSearchAndHydrationCapabilities(t *testing.T) {
	if handler := api.SearchProjectsHandler(projectSearchCapability{}, nil, nil); handler == nil {
		t.Fatal("SearchProjectsHandler returned nil")
	}
}

func TestListProjectsHandlerAcceptsProjectSearchAndHydrationCapabilities(t *testing.T) {
	if handler := api.ListProjectsHandler(projectSearchCapability{}, nil, nil); handler == nil {
		t.Fatal("ListProjectsHandler returned nil")
	}
}

type hashtagPostSearchCapability struct {
	searchPostHydrationCapability
}

func (hashtagPostSearchCapability) SearchHashtagPostsWithLanguages(
	context.Context,
	string,
	[]string,
	string,
	api.SearchSort,
	int,
	string,
	time.Time,
) ([]api.SearchPostRow, string, error) {
	return nil, "", nil
}

func TestSearchHashtagPostsHandlerAcceptsHashtagPostAndHydrationCapabilities(t *testing.T) {
	if handler := api.SearchHashtagPostsHandler(hashtagPostSearchCapability{}, nil, nil); handler == nil {
		t.Fatal("SearchHashtagPostsHandler returned nil")
	}
}

type topHashtagCapability struct{}

func (topHashtagCapability) TopHashtags(
	context.Context,
	api.TopHashtagsRequest,
	time.Time,
) ([]api.TopHashtagGroup, error) {
	return nil, nil
}

func TestTopHashtagsHandlerAcceptsTopHashtagCapability(t *testing.T) {
	if handler := api.TopHashtagsHandler(topHashtagCapability{}, nil); handler == nil {
		t.Fatal("TopHashtagsHandler returned nil")
	}
}

type recentSearchListCapability struct{}

func (recentSearchListCapability) ListRecentSearches(context.Context, string) ([]api.RecentSearchRow, error) {
	return nil, nil
}

func TestListRecentSearchesHandlerAcceptsListCapability(t *testing.T) {
	if handler := api.ListRecentSearchesHandler(recentSearchListCapability{}, nil); handler == nil {
		t.Fatal("ListRecentSearchesHandler returned nil")
	}
}

type recentSearchSaveCapability struct{}

func (recentSearchSaveCapability) SaveRecentSearch(
	context.Context,
	string,
	api.SaveRecentSearchRequest,
	time.Time,
) (api.RecentSearchRow, error) {
	return api.RecentSearchRow{}, nil
}

func TestSaveRecentSearchHandlerAcceptsSaveCapability(t *testing.T) {
	if handler := api.SaveRecentSearchHandler(recentSearchSaveCapability{}, nil); handler == nil {
		t.Fatal("SaveRecentSearchHandler returned nil")
	}
}

type recentSearchDeleteCapability struct{}

func (recentSearchDeleteCapability) DeleteRecentSearch(context.Context, string, string) error {
	return nil
}

func TestDeleteRecentSearchHandlerAcceptsDeleteCapability(t *testing.T) {
	if handler := api.DeleteRecentSearchHandler(recentSearchDeleteCapability{}, nil); handler == nil {
		t.Fatal("DeleteRecentSearchHandler returned nil")
	}
}

func TestSearchStoreDoesNotOwnConcretePostStore(t *testing.T) {
	storeType := reflect.TypeOf(*api.NewSearchStore(nil, nil))
	postStoreType := reflect.TypeOf((*api.PostStore)(nil))
	for i := 0; i < storeType.NumField(); i++ {
		field := storeType.Field(i)
		if field.Type == postStoreType {
			t.Fatalf("SearchStore field %q still depends on concrete *PostStore", field.Name)
		}
	}
}

func TestSearchQueriesLiveInCapabilityOwnedFiles(t *testing.T) {
	expected := map[string][]string{
		"search_profile_store.go": {"SearchProfiles", "searchProfilesObserved"},
		"search_hashtag_store.go": {"SearchHashtags", "SearchHashtagPosts", "SearchHashtagPostsWithLanguages", "TopHashtags"},
		"search_post_store.go":    {"SearchPosts", "SearchPostsWithLanguages", "searchPostsObserved", "searchPostsByRelevance", "searchPosts"},
		"search_project_store.go": {"SearchProjects", "SearchProjectsWithLanguages", "searchProjectsObserved", "searchProjectsByRelevance"},
	}
	for filename, methods := range expected {
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		declarations := map[string]bool{}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv != nil {
				declarations[function.Name.Name] = true
			}
		}
		for _, method := range methods {
			if !declarations[method] {
				t.Errorf("%s does not own %s", filename, method)
			}
		}
	}
}

func TestSearchProfilesHandlerAcceptsProfileSearchCapability(t *testing.T) {
	if handler := api.SearchProfilesHandler(profileSearchCapability{}, nil); handler == nil {
		t.Fatal("SearchProfilesHandler returned nil")
	}
}
