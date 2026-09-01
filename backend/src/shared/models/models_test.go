package models

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type bodyParserIndexedRequest struct {
	NestedContent []*struct {
		Value string `form:"value"`
	} `form:"nested-content"`
	Test []*struct{} `form:"test"`
}

func TestGetBodyMiddlewareRejectsUnsafeSliceIndexes(t *testing.T) {
	tests := []struct {
		name    string
		request func(t *testing.T) *http.Request
	}{
		{
			name: "oversized index",
			request: func(_ *testing.T) *http.Request {
				request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test.18446744073704="))
				request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
				return request
			},
		},
		{
			name: "negative index",
			request: func(t *testing.T) *http.Request {
				t.Helper()
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				if err := writer.WriteField("nested-content[-1].value", "unsafe"); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(http.MethodPost, "/", &body)
				request.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
				return request
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/", GetBodyMiddleware[bodyParserIndexedRequest], func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})

			response, err := app.Test(test.request(t))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode < http.StatusBadRequest {
				t.Fatalf("unsafe slice index was accepted with status %d", response.StatusCode)
			}
		})
	}
}
