package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/recordjson"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/video"
)

type EditPostHandlerOptions struct {
	VideoCompletionVerifier VideoCompletionVerifier
	DisableVideo            bool
}

// CommandEditPostHandler is the addressed published-post editing boundary.
func CommandEditPostHandler(store postAuthorReader, commands AddressedPutCommandExecutor, resolver HandleResolver, limits MediaLimits, options ...EditPostHandlerOptions) http.Handler {
	var media EditPostHandlerOptions
	if len(options) > 0 {
		media = options[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid owner", runID, nil)
			return
		}
		caller, ok := middleware.GetDID(r.Context())
		if !ok || caller != owner {
			envelope.WriteError(w, 403, "forbidden", "cannot edit this post", runID, nil)
			return
		}
		key, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid record key", runID, nil)
			return
		}
		generation, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		operationKey, ok := requireCommandOperationKey(w, r, runID)
		if !ok {
			return
		}
		expected, err := ParseBusinessIfMatch(r)
		if err != nil || expected == "*" {
			WritePDSRecordConflict(w, runID)
			return
		}
		var input struct {
			Changes map[string]json.RawMessage `json:"changes"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			envelope.WriteError(w, 400, "malformed_body", "invalid edit", runID, nil)
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF || input.Changes == nil {
			envelope.WriteError(w, 400, "malformed_body", "invalid edit", runID, nil)
			return
		}
		for field := range input.Changes {
			if field != "text" && field != "sponsored" && field != "langs" && field != "facets" && field != "project" && field != "images" && field != "embed" {
				envelope.WriteError(w, 400, "unexpected_field", "unsupported edit field", runID, nil)
				return
			}
		}
		uri := syntax.ATURI("at://" + owner.String() + "/" + craftskyPostNSID + "/" + key.String())
		intent, _ := json.Marshal(input)
		// The immutable request identifies a replacement even when a resumed
		// dispatch uses its stored plan without rebuilding/verifying it again.
		var replacementVideo struct {
			JobID string          `json:"jobId"`
			Video json.RawMessage `json:"video"`
		}
		_ = json.Unmarshal(input.Changes["embed"], &replacementVideo)
		session, _ := middleware.GetOAuthSessionID(r.Context())
		result, err := commands.Put(r.Context(), pdscommands.AddressedPutCommandRequest{
			Owner: owner, OwnerGeneration: generation, SessionID: session,
			OperationKind: "post.edit", OperationKey: operationKey, URI: uri, ExpectedCID: expected, Intent: intent,
			StrictExpectedCID: true,
			BuildPreparedRecord: func(current pdscommands.AuthoritativeRecord, prepared pdscommands.PreparedCommand) (json.RawMessage, error) {
				if reader, ok := store.(postEditEligibilityStore); ok {
					if err := reader.CheckPostEditAllowed(r.Context(), owner, key); err != nil {
						if errors.Is(err, ErrPostNotFound) {
							return nil, err
						}
						return nil, errors.Join(pdscommands.ErrDispatchUnavailable, err)
					}
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(current.Record, &fields); err != nil {
					return nil, err
				}
				if text, present := input.Changes["text"]; present {
					var original, changed string
					_ = json.Unmarshal(fields["text"], &original)
					_ = json.Unmarshal(text, &changed)
					if _, supplied := input.Changes["facets"]; !supplied && original != changed {
						delete(fields, "facets")
					}
				}
				for field, value := range input.Changes {
					if field == "embed" {
						merged, err := mergePostEditEmbed(fields[field], value)
						if err != nil {
							return nil, err
						}
						if len(merged) == 0 {
							delete(fields, field)
						} else {
							fields[field] = merged
						}
						continue
					}
					if field == "images" {
						if string(value) == "null" {
							delete(fields, field)
							continue
						}
						merged, err := mergePostEditEntries([]string{"image", "alt", "aspectRatio"})(fields[field], value)
						if err != nil {
							return nil, err
						}
						fields[field] = merged
						continue
					}
					if field == "project" {
						merged, err := mergePostProjectEdit(fields[field], value)
						if err != nil {
							return nil, err
						}
						fields[field] = merged
						continue
					}
					if (field == "langs" || field == "facets") && string(value) == "null" {
						delete(fields, field)
						continue
					}
					fields[field] = value
				}
				desired, err := json.Marshal(fields)
				if err != nil {
					return nil, err
				}
				candidate, err := postEditCandidate(desired)
				if err != nil {
					return nil, err
				}
				if err := validatePostEditCandidate(candidate, current.Record, limits); err != nil {
					return nil, err
				}
				if candidate.Embed != nil && candidate.Embed.Video != nil {
					original, err := postEditCandidate(current.Record)
					if err != nil {
						return nil, err
					}
					if original.Embed == nil || original.Embed.Video == nil || candidate.Embed.Video.Blob != original.Embed.Video.Blob {
						var upload struct {
							JobID string `json:"jobId"`
						}
						if err := json.Unmarshal(input.Changes["embed"], &upload); err != nil {
							return nil, err
						}
						if upload.JobID == "" {
							return nil, &FieldError{Code: "video_verification_failed", Fields: map[string]string{"embed.video.jobId": "a replacement video requires a verified completion job"}}
						}
						if media.DisableVideo || media.VideoCompletionVerifier == nil {
							return nil, pdscommands.ErrDispatchUnavailable
						}
						submitted := video.Blob{CID: syntax.CID(candidate.Embed.Video.Blob.Ref.Link), MIMEType: candidate.Embed.Video.Blob.MIMEType, Size: candidate.Embed.Video.Blob.Size}
						verified, err := media.VideoCompletionVerifier.Verify(r.Context(), owner, upload.JobID, submitted)
						if err != nil {
							var rejected *video.VerificationError
							if errors.As(err, &rejected) && rejected.Kind == video.VerificationRejected {
								return nil, &FieldError{Code: "video_verification_failed", Fields: map[string]string{"embed.video": "video could not be verified"}}
							}
							return nil, errors.Join(pdscommands.ErrDispatchUnavailable, err)
						}
						if verified != submitted {
							return nil, &FieldError{Code: "video_verification_failed", Fields: map[string]string{"embed.video": "verified video does not match the submitted blob"}}
						}
					}
				}
				if err := lexiconschema.ValidateCraftskyRecord(desired, craftskyPostNSID); err != nil {
					return nil, err
				}
				if err := validatePostEditRichText(candidate); err != nil {
					return nil, err
				}
				if authorizer, ok := store.(DirectedInteractionAuthorizer); ok {
					original, err := postEditCandidate(current.Record)
					if err != nil {
						return nil, err
					}
					prior, err := mentionedDIDs(original)
					if err != nil {
						return nil, err
					}
					retained := map[syntax.DID]bool{}
					for _, did := range prior {
						retained[did] = true
					}
					mentioned, err := mentionedDIDs(candidate)
					if err != nil {
						return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{"facets": "mention DID is invalid"}}
					}
					for _, did := range mentioned {
						if retained[did] {
							continue
						}
						if err := authorizer.AuthorizeDirectedInteraction(r.Context(), owner, did, relationships.OperationMentionCreate); err != nil {
							if errors.Is(err, ErrInteractionBlocked) || errors.Is(err, relationships.ErrProfileNotFound) {
								return nil, err
							}
							return nil, errors.Join(pdscommands.ErrDispatchUnavailable, err)
						}
					}
				}
				var before, after any
				if err := recordjson.Unmarshal(current.Record, &before); err != nil {
					return nil, err
				}
				if err := recordjson.Unmarshal(desired, &after); err != nil {
					return nil, err
				}
				if reflect.DeepEqual(before, after) {
					return current.Record, nil
				}
				fields["editedAt"], _ = json.Marshal(prepared.CreatedAt.UTC().Format(time.RFC3339Nano))
				return json.Marshal(fields)
			},
			Accepted: func(record pdscommands.AuthoritativeRecord) (pdscommands.TerminalResult, error) {
				return acceptedPostEditResult(record, expected)
			},
			Rejected: func(err error) pdscommands.TerminalResult {
				if replacementVideo.JobID != "" && len(replacementVideo.Video) > 0 && pdsAPIErrorName(err) == "BlobNotFound" {
					body, _ := json.Marshal(envelope.Error{Error: "video_blob_missing", Message: "processed video blob is unavailable", RequestID: runID})
					return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: http.StatusBadGateway, ResponseBody: body}
				}
				if errors.Is(err, ErrInteractionBlocked) {
					body, _ := json.Marshal(envelope.Error{Error: "interaction_blocked", Message: "interaction is not allowed across a block", RequestID: runID})
					return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: 403, ResponseBody: body}
				}
				if errors.Is(err, relationships.ErrProfileNotFound) {
					body, _ := json.Marshal(envelope.Error{Error: "profile_not_found", Message: "profile not found", RequestID: runID})
					return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: 404, ResponseBody: body}
				}
				if errors.Is(err, ErrPostNotFound) {
					body, _ := json.Marshal(envelope.Error{Error: "post_not_found", Message: "post not found", RequestID: runID})
					return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: 404, ResponseBody: body}
				}
				var fieldError *FieldError
				if errors.As(err, &fieldError) {
					body, _ := json.Marshal(envelope.Error{Error: fieldError.Code, Message: "validation failed", RequestID: runID, Fields: fieldError.Fields})
					return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: http.StatusUnprocessableEntity, ResponseBody: body}
				}
				return RejectedCommandResult(runID, err, r.Context())
			},
		})
		if err != nil {
			WriteCommandError(w, runID, err, r.Context())
			return
		}
		WriteCommandResponse(w, CommandResultFromStored(result))
	})
}
