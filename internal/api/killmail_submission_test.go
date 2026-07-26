package api

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSubmittedKillmailsAcceptsESIVariantsAndDeduplicates(t *testing.T) {
	hashA := "AABBCCDDEEFF00112233"
	hashB := "1234567890abcdef1234567890abcdef"
	parsed := parseSubmittedKillmails(strings.Join([]string{
		"https://esi.evetech.net/latest/killmails/123/" + hashA + "/",
		"http://esi.evetech.net/v2/killmails/456/" + hashB,
		"https://esi.evetech.net/killmails/123/" + strings.ToLower(hashA),
		"https://example.com/latest/killmails/789/" + hashA,
	}, "\n"))

	if len(parsed) != 2 {
		t.Fatalf("parsed = %#v, want two unique ESI killmails", parsed)
	}
	if parsed[0].ID != 123 || parsed[0].Hash != strings.ToLower(hashA) {
		t.Errorf("first = %#v", parsed[0])
	}
	if parsed[1].ID != 456 || parsed[1].Hash != hashB {
		t.Errorf("second = %#v", parsed[1])
	}
}

func TestDecodeKillmailSubmissionPreservesTextPrecedence(t *testing.T) {
	body, err := decodeKillmailSubmission(strings.NewReader(
		`{"text":"","links":["https://esi.evetech.net/killmails/1/aabbccddeeff00112233"]}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if body.Text == nil || *body.Text != "" || len(body.Links) != 1 {
		t.Fatalf("body = %#v", body)
	}
}

func TestDecodeKillmailSubmissionRejectsTrailingJSON(t *testing.T) {
	_, err := decodeKillmailSubmission(strings.NewReader(`{} {}`))
	var apiErr *legacyAPIError
	if !errors.As(err, &apiErr) || apiErr.Status != 400 {
		t.Fatalf("error = %#v, want API 400", err)
	}
}
