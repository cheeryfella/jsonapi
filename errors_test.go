package jsonapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/cheeryfella/jsonapi"
)

func TestErrorObjectWritesExpectedErrorMessage(t *testing.T) {
	input := &jsonapi.ErrorObject{Title: "Title test.", Detail: "Detail test."}

	output := input.Error()

	if output != fmt.Sprintf("Error: %s %s\n", input.Title, input.Detail) {
		t.Fatal("Unexpected output.")
	}
}

func TestMarshalErrorsWritesTheExpectedPayload(t *testing.T) {
	var marshalErrorsTableTasts = map[string]struct {
		In  []*jsonapi.ErrorObject
		Out map[string]interface{}
	}{
		"TestFieldsAreSerializedAsNeeded": {
			In: []*jsonapi.ErrorObject{{
				ID:     "0",
				Title:  "Test title.",
				Detail: "Test detail",
				Status: "400",
				Code:   "E1100",
			}},
			Out: map[string]interface{}{
				"errors": []interface{}{map[string]interface{}{
					"id":     "0",
					"title":  "Test title.",
					"detail": "Test detail",
					"status": "400",
					"code":   "E1100",
				}},
			},
		},
		"TestMetaFieldIsSerializedProperly": {
			In: []*jsonapi.ErrorObject{{
				Title:  "Test title.",
				Detail: "Test detail",
				Meta: &map[string]interface{}{
					"key": "val",
				},
			}},
			Out: map[string]interface{}{
				"errors": []interface{}{map[string]interface{}{
					"title":  "Test title.",
					"detail": "Test detail",
					"meta": map[string]interface{}{
						"key": "val",
					}},
				},
			},
		},
		"TestSourceFieldIsSerializedProperly": {
			In: []*jsonapi.ErrorObject{{
				Title:  "Test title.",
				Detail: "Test detail",
				Source: &jsonapi.ErrorSource{
					Pointer:   "/data/attributes/field",
					Parameter: "filter",
				},
			}},
			Out: map[string]interface{}{
				"errors": []interface{}{map[string]interface{}{
					"title":  "Test title.",
					"detail": "Test detail",
					"source": map[string]interface{}{
						"parameter": "filter",
						"pointer":   "/data/attributes/field",
					},
				},
				}},
		},
		"TestLinksFieldIsSerializedProperly": {
			In: []*jsonapi.ErrorObject{{
				Title:  "Test title.",
				Detail: "Test detail",
				Links: &jsonapi.ErrorLink{
					About: "/url/to/details",
				},
			}},
			Out: map[string]interface{}{
				"errors": []interface{}{map[string]interface{}{
					"title":  "Test title.",
					"detail": "Test detail",
					"links": map[string]interface{}{
						"about": "/url/to/details",
					},
				}},
			},
		},
	}

	for name, test := range marshalErrorsTableTasts {
		t.Run(name, func(t *testing.T) {
			buffer, output := bytes.NewBuffer(nil), map[string]interface{}{}
			var writer io.Writer = buffer

			_ = jsonapi.MarshalErrors(writer, test.In)
			json.Unmarshal(buffer.Bytes(), &output)

			if !reflect.DeepEqual(output, test.Out) {
				t.Fatalf("Expected: \n%#v \nto equal: \n%#v", output, test.Out)
			}
		})
	}
}
