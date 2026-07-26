package api

import (
	"time"

	"social.craftsky/appview/internal/api/envelope"
)

type SavedPostScope string

const (
	SavedPostScopeAll     SavedPostScope = "all"
	SavedPostScopeFolder  SavedPostScope = "folder"
	SavedPostScopeUnfiled SavedPostScope = "unfiled"
)

type SavedPostSort string

const (
	SavedPostSortNewest SavedPostSort = "newest"
	SavedPostSortOldest SavedPostSort = "oldest"
)

type SavedPostCursor struct {
	Scope    SavedPostScope
	FolderID string
	Sort     SavedPostSort
	SavedAt  time.Time
	URI      string
}

type SavedPostFolderCursor struct {
	FoldedName string
	FolderID   string
}

func EncodeSavedPostCursor(scope SavedPostScope, folderID string, sort SavedPostSort, savedAt time.Time, uri string) (string, error) {
	if !validSavedPostScope(scope) || !validSavedPostSort(sort) || savedAt.IsZero() || uri == "" {
		return "", envelope.ErrInvalidCursor
	}
	if (scope == SavedPostScopeFolder) != (folderID != "") {
		return "", envelope.ErrInvalidCursor
	}
	payload := map[string]any{
		"kind":    "savedPost",
		"scope":   string(scope),
		"sort":    string(sort),
		"savedAt": savedAt.UTC().Format(time.RFC3339Nano),
		"uri":     uri,
	}
	if scope == SavedPostScopeFolder {
		payload["folderId"] = folderID
	}
	return envelope.EncodeCursor(payload)
}

func DecodeSavedPostCursor(cursor string, scope SavedPostScope, folderID string, sort SavedPostSort) (SavedPostCursor, error) {
	if cursor == "" {
		return SavedPostCursor{}, nil
	}
	if !validSavedPostScope(scope) || !validSavedPostSort(sort) || (scope == SavedPostScopeFolder) != (folderID != "") {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	if payload["kind"] != "savedPost" || payload["scope"] != string(scope) || payload["sort"] != string(sort) {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	if _, encodedOwner := payload["ownerDid"]; encodedOwner {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	encodedFolderID, hasFolderID := payload["folderId"].(string)
	if scope == SavedPostScopeFolder {
		if !hasFolderID || encodedFolderID != folderID {
			return SavedPostCursor{}, envelope.ErrInvalidCursor
		}
	} else if _, present := payload["folderId"]; present {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	rawSavedAt, ok := payload["savedAt"].(string)
	if !ok || rawSavedAt == "" {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	savedAt, err := time.Parse(time.RFC3339Nano, rawSavedAt)
	if err != nil {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	uri, ok := payload["uri"].(string)
	if !ok || uri == "" {
		return SavedPostCursor{}, envelope.ErrInvalidCursor
	}
	return SavedPostCursor{Scope: scope, FolderID: folderID, Sort: sort, SavedAt: savedAt, URI: uri}, nil
}

func EncodeSavedPostFolderCursor(foldedName, folderID string) (string, error) {
	if foldedName == "" || folderID == "" {
		return "", envelope.ErrInvalidCursor
	}
	return envelope.EncodeCursor(map[string]any{
		"kind":       "savedPostFolder",
		"foldedName": foldedName,
		"folderId":   folderID,
	})
}

func DecodeSavedPostFolderCursor(cursor string) (SavedPostFolderCursor, error) {
	if cursor == "" {
		return SavedPostFolderCursor{}, nil
	}
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil || payload["kind"] != "savedPostFolder" {
		return SavedPostFolderCursor{}, envelope.ErrInvalidCursor
	}
	if _, encodedOwner := payload["ownerDid"]; encodedOwner {
		return SavedPostFolderCursor{}, envelope.ErrInvalidCursor
	}
	foldedName, ok := payload["foldedName"].(string)
	if !ok || foldedName == "" {
		return SavedPostFolderCursor{}, envelope.ErrInvalidCursor
	}
	folderID, ok := payload["folderId"].(string)
	if !ok || folderID == "" {
		return SavedPostFolderCursor{}, envelope.ErrInvalidCursor
	}
	return SavedPostFolderCursor{FoldedName: foldedName, FolderID: folderID}, nil
}

func validSavedPostScope(scope SavedPostScope) bool {
	switch scope {
	case SavedPostScopeAll, SavedPostScopeFolder, SavedPostScopeUnfiled:
		return true
	default:
		return false
	}
}

func validSavedPostSort(sort SavedPostSort) bool {
	return sort == SavedPostSortNewest || sort == SavedPostSortOldest
}
