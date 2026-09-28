package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

func TestCommandPutBusinessProfileUsesFixedSelfAndPreservesUnknownExtensions(t *testing.T) {
	commands := &recordingAddressedPutCommands{}
	commands.put = func(request pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
		record, err := request.BuildRecord(pdscommands.AuthoritativeRecord{
			URI: request.URI, CID: businessProfileCID1, Record: json.RawMessage(`{
				"$type":"social.craftsky.business.profile",
				"hoursNote":"old hours",
				"com.example.extension":{"sequence":9007199254740993}
			}`),
		})
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		commands.record = record
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{
			URI: request.URI, CID: businessProfileCID2, Record: record,
		})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.PutBusinessProfileHandler(
		nil,
		api.BusinessProfileHandlerOptions{Commands: commands},
	)
	request := httptest.NewRequest(http.MethodPut, "/v1/profiles/me/business", strings.NewReader(`{
		"tagline":"Owner replacement",
		"businessTypes":["teacher","dyer"],
		"products":[{"title":"Wool","uri":"https://example.com/wool","image":{"image":{"$type":"blob","ref":{"$link":"`+businessProfileCID1+`"},"mimeType":"image/png","size":12},"alt":"Wool"}}]
	}`))
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-999999999996")
	request.Header.Set("If-Match", businessProfileCID1)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-profile-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if commands.putCalls != 1 || commands.putRequest.OperationKind != "business_profile.put" ||
		commands.putRequest.URI != "at://did:plc:owner/social.craftsky.business.profile/self" ||
		commands.putRequest.ExpectedCID != businessProfileCID1 {
		t.Fatalf("fixed-key put request = %+v calls=%d", commands.putRequest, commands.putCalls)
	}
	if len(commands.putRequest.Blobs) != 1 || commands.putRequest.Blobs[0].CID != businessProfileCID1 ||
		commands.putRequest.Blobs[0].MIMEType != "image/png" || commands.putRequest.Blobs[0].Size != 12 {
		t.Fatalf("fixed-key blob references = %+v", commands.putRequest.Blobs)
	}
	var record map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(commands.record)))
	decoder.UseNumber()
	if err := decoder.Decode(&record); err != nil {
		t.Fatal(err)
	}
	if record["tagline"] != "Owner replacement" {
		t.Fatalf("replacement record = %#v", record)
	}
	if _, present := record["hoursNote"]; present {
		t.Fatalf("omitted controlled field survived replacement: %#v", record)
	}
	extension, ok := record["com.example.extension"].(map[string]any)
	if !ok || extension["sequence"] != json.Number("9007199254740993") {
		t.Fatalf("unknown extension = %#v", record["com.example.extension"])
	}
	assertBusinessProfileCID(t, response, businessProfileCID2)
}

func TestCommandDeleteBusinessProfileUsesFixedSelfAndReturnsEmptyNoContent(t *testing.T) {
	commands := &recordingAddressedCommands{result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
	}}}
	handler := api.DeleteBusinessProfileHandler(
		nil,
		api.BusinessProfileHandlerOptions{DeleteCommands: commands},
	)
	request := httptest.NewRequest(http.MethodDelete, "/v1/profiles/me/business", nil)
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-999999999997")
	request.Header.Set("If-Match", businessProfileCID2)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-profile-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("status = %d body=%q", response.Code, response.Body.String())
	}
	if commands.calls != 1 || commands.request.OperationKind != "business_profile.delete" ||
		commands.request.URI != "at://did:plc:owner/social.craftsky.business.profile/self" ||
		commands.request.ExpectedCID != businessProfileCID2 {
		t.Fatalf("fixed-key delete request = %+v calls=%d", commands.request, commands.calls)
	}
}
