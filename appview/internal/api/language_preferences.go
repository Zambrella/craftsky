package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
)

type LanguagePreferenceStore interface {
	Get(context.Context, syntax.DID) (languages.Preferences, error)
	Replace(
		context.Context,
		syntax.DID,
		languages.Preferences,
	) (languages.Preferences, error)
	Initialize(
		context.Context,
		syntax.DID,
		languages.Preferences,
	) (languages.Preferences, error)
}

func GetLanguagePreferencesHandler(store LanguagePreferenceStore) http.Handler {
	return languagePreferencesHandler(store, false, false)
}

func PutLanguagePreferencesHandler(store LanguagePreferenceStore) http.Handler {
	return languagePreferencesHandler(store, true, false)
}

func InitializeLanguagePreferencesHandler(store LanguagePreferenceStore) http.Handler {
	return languagePreferencesHandler(store, true, true)
}

func languagePreferencesHandler(
	store LanguagePreferenceStore,
	replacing bool,
	initializing bool,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if len(r.URL.Query()) != 0 {
			envelope.WriteError(
				w,
				http.StatusBadRequest,
				"invalid_request",
				"language preference routes do not accept query parameters",
				runID,
				nil,
			)
			return
		}
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(
				w,
				http.StatusInternalServerError,
				"missing_authenticated_did",
				"authenticated DID missing",
				runID,
				nil,
			)
			return
		}

		var (
			preferences languages.Preferences
			err         error
		)
		if replacing {
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if decodeErr := decoder.Decode(&preferences); decodeErr != nil ||
				preferences.ContentLanguages == nil {
				envelope.WriteError(
					w,
					http.StatusBadRequest,
					"invalid_request",
					"invalid language preferences",
					runID,
					nil,
				)
				return
			}
			if decodeErr := decoder.Decode(&struct{}{}); !errors.Is(decodeErr, io.EOF) {
				envelope.WriteError(
					w,
					http.StatusBadRequest,
					"invalid_request",
					"invalid language preferences",
					runID,
					nil,
				)
				return
			}
			if initializing {
				preferences, err = store.Initialize(r.Context(), did, preferences)
			} else {
				preferences, err = store.Replace(r.Context(), did, preferences)
			}
		} else {
			preferences, err = store.Get(r.Context(), did)
		}
		if err != nil {
			if errors.Is(err, languages.ErrInvalidPreferences) {
				envelope.WriteError(
					w,
					http.StatusBadRequest,
					"invalid_request",
					"invalid language preferences",
					runID,
					nil,
				)
				return
			}
			if errors.Is(err, languages.ErrPreferencesNotFound) {
				envelope.WriteError(
					w,
					http.StatusNotFound,
					"language_preferences_not_found",
					"language preferences not found",
					runID,
					nil,
				)
				return
			}
			envelope.WriteError(
				w,
				http.StatusInternalServerError,
				"internal_error",
				"language preferences operation failed",
				runID,
				nil,
			)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(preferences)
	})
}
