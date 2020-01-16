package bitwarden

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoginAndListItems(t *testing.T) {
	testCases := []struct {
		name               string
		username           string
		password           string
		responses          map[string]execResponse
		expectedCalls      [][]string
		expectedSession    string
		expectedSavedItems []Item
		expectedError      error
	}{
		{
			name:     "basic case",
			username: "u",
			password: "p",
			responses: map[string]execResponse{
				"login u p --response": {
					out: []byte(`{
  "success": true,
  "data": {
    "noColor": false,
    "object": "message",
    "title": "You are logged in!",
    "message": "\nTo unlock your vault, set ...",
    "raw": "not-going-to-tell-you=="
  }
}`),
				},
				"--session not-going-to-tell-you== list items": {
					out: []byte(`[
  {
    "object": "item",
    "id": "id1",
    "organizationId": "org1",
    "folderId": null,
    "type": 2,
    "name": "unsplash.com",
    "notes": null,
    "favorite": false,
    "fields": [
      {
        "name": "API Key",
        "value": "value1",
        "type": 0
      }
    ],
    "secureNote": {
      "type": 0
    },
    "collectionIds": [
      "id1"
    ],
    "revisionDate": "2019-10-11T23:33:21.970Z"
  },
  {
    "object": "item",
    "id": "id2",
    "organizationId": "org1",
    "folderId": null,
    "type": 2,
    "name": "my-credentials",
    "notes": "important notes",
    "favorite": false,
    "secureNote": {
      "type": 0
    },
    "collectionIds": [
      "id2"
    ],
    "attachments": [
      {
        "id": "a-id1",
        "fileName": "secret.auto.vars",
        "size": "161",
        "sizeName": "161 Bytes",
        "url": "https://cdn.bitwarden.net/attachments/111/222"
      }
    ],
    "revisionDate": "2019-04-04T03:43:19.503Z"
  }
]`),
				},
			},
			expectedCalls: [][]string{
				{"login", "u", "p", "--response"},
				{"--session", "not-going-to-tell-you==", "list", "items"},
			},
			expectedSession: "not-going-to-tell-you==",
			expectedSavedItems: []Item{
				{
					ID:   "id1",
					Name: "unsplash.com",
					Fields: []Field{
						{
							Name:  "API Key",
							Value: "value1",
						},
					},
				},
				{
					ID:   "id2",
					Name: "my-credentials",
					Attachments: []Attachment{
						{
							ID:       "a-id1",
							FileName: "secret.auto.vars",
						},
					},
				},
			},
		},
		{
			name:     "some unknown error on list cmd",
			username: "u",
			password: "p",
			responses: map[string]execResponse{
				"login u p --response": {
					out: []byte(`{
  "success": true,
  "data": {
    "noColor": false,
    "object": "message",
    "title": "You are logged in!",
    "message": "\nTo unlock your vault, set ...",
    "raw": "not-going-to-tell-you=="
  }
}`),
				},
				"--session not-going-to-tell-you== list items": {
					err: fmt.Errorf("some unknown error"),
				},
			},
			expectedCalls: [][]string{
				{"login", "u", "p", "--response"},
				{"--session", "not-going-to-tell-you==", "list", "items"},
			},
			expectedSession: "not-going-to-tell-you==",
			expectedError:   fmt.Errorf("some unknown error"),
		},
		{
			name:     "u/p not matching",
			username: "u",
			password: "p",
			responses: map[string]execResponse{
				"login u p --response": {
					out: []byte(`{
  "success": false,
  "message": "Username or password is incorrect. Try again."
}`),
					err: fmt.Errorf("failed to login: Username or password is incorrect. Try again"),
				},
			},
			expectedCalls: [][]string{
				{"login", "u", "p", "--response"},
			},
			expectedError: fmt.Errorf("failed to login: Username or password is incorrect. Try again"),
		},
		{
			name:     "already logged in",
			username: "u",
			password: "p",
			responses: map[string]execResponse{
				"login u p --response": {
					out: []byte(`{
  "success": false,
  "message": "You are already logged in as dptp@redhat.com."
}`),
					err: fmt.Errorf("failed to login: You are already logged in as dptp@redhat.com"),
				},
			},
			expectedCalls: [][]string{
				{"login", "u", "p", "--response"},
			},
			expectedError: fmt.Errorf("failed to login: You are already logged in as dptp@redhat.com"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := fakeExecutor{
				records:   [][]string{},
				responses: tc.responses,
			}
			client := cliClient{
				username: tc.username,
				password: tc.password,
				run:      e.Run,
			}
			actualError := client.loginAndListItems()

			equalError(t, tc.expectedError, actualError)
			equal(t, tc.expectedSession, client.session)
			equal(t, tc.expectedSavedItems, client.savedItems)
			equal(t, tc.expectedCalls, e.records)
		})
	}
}

func TestGetFieldOnItem(t *testing.T) {
	testCases := []struct {
		name        string
		client      *cliClient
		itemName    string
		fieldName   string
		expected    []byte
		expectedErr error
	}{
		{
			name: "basic case",
			client: &cliClient{
				savedItems: []Item{
					{
						ID:   "id1",
						Name: "unsplash.com",
						Fields: []Field{
							{
								Name:  "API Key",
								Value: "value1",
							},
							{
								Name:  "no name",
								Value: "value2",
							},
						},
					},
					{
						ID:   "id2",
						Name: "my-credentials",
						Attachments: []Attachment{
							{
								ID:       "a-id1",
								FileName: "secret.auto.vars",
							},
						},
					},
				},
			},
			itemName:  "unsplash.com",
			fieldName: "API Key",
			expected:  []byte("value1"),
		},
		{
			name: "item not find",
			client: &cliClient{
				savedItems: []Item{
					{
						ID:   "id1",
						Name: "unsplash.com",
						Fields: []Field{
							{
								Name:  "API Key",
								Value: "value1",
							},
							{
								Name:  "no name",
								Value: "value2",
							},
						},
					},
					{
						ID:   "id2",
						Name: "my-credentials",
						Attachments: []Attachment{
							{
								ID:       "a-id1",
								FileName: "secret.auto.vars",
							},
						},
					},
				},
			},
			itemName:    "no-item",
			fieldName:   "API Key",
			expectedErr: fmt.Errorf("failed to find field API Key in item no-item"),
		},
		{
			name: "field not found",
			client: &cliClient{
				savedItems: []Item{
					{
						ID:   "id1",
						Name: "unsplash.com",
						Fields: []Field{
							{
								Name:  "API Key",
								Value: "value1",
							},
							{
								Name:  "no name",
								Value: "value2",
							},
						},
					},
					{
						ID:   "id2",
						Name: "my-credentials",
						Attachments: []Attachment{
							{
								ID:       "a-id1",
								FileName: "secret.auto.vars",
							},
						},
					},
				},
			},
			itemName:    "unsplash.com",
			fieldName:   "no-field",
			expectedErr: fmt.Errorf("failed to find field no-field in item unsplash.com"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, actualErr := tc.client.GetFieldOnItem(tc.itemName, tc.fieldName)
			equalError(t, tc.expectedErr, actualErr)
			equal(t, tc.expected, actual)
		})
	}
}

func TestGetAttachmentOnItem(t *testing.T) {
	client := &cliClient{
		session: "abc",
		savedItems: []Item{
			{ID: "id1", Name: "unsplash.com", Fields: []Field{{Name: "API Key", Value: "value1"}}},
			{ID: "id2", Name: "my-credentials", Attachments: []Attachment{{ID: "a-id1", FileName: "secret.auto.vars"}, {ID: "a-id2", FileName: ".awsred"}}},
		},
	}
	testCases := []struct {
		name           string
		responses      map[string]execResponse
		expectedCalls  [][]string
		itemName       string
		attachmentName string
		expected       []byte
		expectedErr    error
	}{
		{
			name: "basic case",
			responses: map[string]execResponse{
				"--session abc get attachment a-id1 --itemid id2 --raw": {
					out: []byte(`bla`),
				},
			},
			expectedCalls: [][]string{
				{"--session", "abc", "get", "attachment", "a-id1", "--itemid", "id2", "--raw"},
			},
			itemName:       "my-credentials",
			attachmentName: "secret.auto.vars",
			expected:       []byte("bla"),
		},
		{
			name: "get attachment cmd err",
			responses: map[string]execResponse{
				"--session abc get attachment a-id1 --itemid id2 --raw": {
					err: fmt.Errorf("some err"),
				},
			},
			expectedCalls: [][]string{
				{"--session", "abc", "get", "attachment", "a-id1", "--itemid", "id2", "--raw"},
			},
			itemName:       "my-credentials",
			attachmentName: "secret.auto.vars",
			expectedErr:    fmt.Errorf("some err"),
		},
		{
			name:           "item not found",
			expectedCalls:  [][]string{},
			itemName:       "no-item",
			attachmentName: "secret.auto.vars",
			expectedErr:    fmt.Errorf("failed to find attachment secret.auto.vars in item no-item"),
		},
		{
			name:           "attachment not found",
			expectedCalls:  [][]string{},
			itemName:       "my-credentials",
			attachmentName: "no attachment",
			expectedErr:    fmt.Errorf("failed to find attachment no attachment in item my-credentials"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := fakeExecutor{
				records:   [][]string{},
				responses: tc.responses,
			}
			client.run = e.Run
			actual, actualErr := client.GetAttachmentOnItem(tc.itemName, tc.attachmentName)
			equalError(t, tc.expectedErr, actualErr)
			equal(t, tc.expected, actual)
			equal(t, tc.expectedCalls, e.records)
		})
	}
}

type execResponse struct {
	out []byte
	err error
}

// fakeExecutor is useful in testing for mocking an Executor
type fakeExecutor struct {
	records   [][]string
	responses map[string]execResponse
}

func (e *fakeExecutor) Run(args ...string) ([]byte, error) {
	e.records = append(e.records, args)
	key := strings.Join(args, " ")
	if response, ok := e.responses[key]; ok {
		return response.out, response.err
	}
	return []byte{}, fmt.Errorf("no response configured for %s", key)
}

func equalError(t *testing.T, expected, actual error) {
	if expected != nil && actual == nil || expected == nil && actual != nil {
		t.Errorf("expecting error %v, got %v", expected, actual)
	}
	if expected != nil && actual != nil && expected.Error() != actual.Error() {
		t.Errorf("expecting error msg %q, got %q", expected.Error(), actual.Error())
	}
}

func equal(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("actual differs from expected:\n%s", cmp.Diff(expected, actual))
	}
}