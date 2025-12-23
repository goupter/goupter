package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/errors"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestResponse_Struct(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"key": "value"},
		TraceID: "trace-123",
	}

	if resp.Code != 0 {
		t.Errorf("Code = %d, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %s, want success", resp.Message)
	}
	if resp.TraceID != "trace-123" {
		t.Errorf("TraceID = %s, want trace-123", resp.TraceID)
	}
}

func TestPageData_Struct(t *testing.T) {
	page := PageData{
		List:     []string{"a", "b", "c"},
		Total:    100,
		Page:     1,
		PageSize: 10,
	}

	if page.Total != 100 {
		t.Errorf("Total = %d, want 100", page.Total)
	}
	if page.Page != 1 {
		t.Errorf("Page = %d, want 1", page.Page)
	}
	if page.PageSize != 10 {
		t.Errorf("PageSize = %d, want 10", page.PageSize)
	}
}

func TestJSON(t *testing.T) {
	c, w := setupTestContext()

	data := map[string]string{"key": "value"}
	JSON(c, 0, "success", data)

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Code = %d, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %s, want success", resp.Message)
	}
}

func TestJSON_WithTraceID(t *testing.T) {
	c, w := setupTestContext()
	c.Set("trace_id", "trace-abc-123")

	JSON(c, 0, "success", nil)

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.TraceID != "trace-abc-123" {
		t.Errorf("TraceID = %s, want trace-abc-123", resp.TraceID)
	}
}

func TestSuccess(t *testing.T) {
	c, w := setupTestContext()

	data := map[string]int{"count": 42}
	Success(c, data)

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != errors.CodeSuccess {
		t.Errorf("Code = %d, want %d", resp.Code, errors.CodeSuccess)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %s, want success", resp.Message)
	}
}

func TestError(t *testing.T) {
	c, w := setupTestContext()

	err := errors.New(errors.CodeBadRequest, "invalid input")
	Error(c, err)

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Code != errors.CodeBadRequest {
		t.Errorf("Code = %d, want %d", resp.Code, errors.CodeBadRequest)
	}
	if resp.Message != "invalid input" {
		t.Errorf("Message = %s, want 'invalid input'", resp.Message)
	}
}

func TestError_WithDetails(t *testing.T) {
	c, w := setupTestContext()

	err := errors.New(errors.CodeBadRequest, "validation failed").WithDetails(map[string]string{
		"field": "email",
	})
	Error(c, err)

	var resp Response
	if e := json.Unmarshal(w.Body.Bytes(), &resp); e != nil {
		t.Fatalf("Failed to unmarshal response: %v", e)
	}

	if resp.Data == nil {
		t.Error("Data should contain details")
	}
}

func TestAbort(t *testing.T) {
	c, w := setupTestContext()

	err := errors.New(errors.CodeUnauthorized, "authentication required")
	Abort(c, err)

	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if e := json.Unmarshal(w.Body.Bytes(), &resp); e != nil {
		t.Fatalf("Failed to unmarshal response: %v", e)
	}

	if resp.Code != errors.CodeUnauthorized {
		t.Errorf("Code = %d, want %d", resp.Code, errors.CodeUnauthorized)
	}

	if !c.IsAborted() {
		t.Error("Context should be aborted")
	}
}

func TestResponse_JSONSerialization(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "success",
		Data:    map[string]string{"key": "value"},
		TraceID: "trace-123",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Code != resp.Code {
		t.Errorf("Code = %d, want %d", parsed.Code, resp.Code)
	}
	if parsed.Message != resp.Message {
		t.Errorf("Message = %s, want %s", parsed.Message, resp.Message)
	}
	if parsed.TraceID != resp.TraceID {
		t.Errorf("TraceID = %s, want %s", parsed.TraceID, resp.TraceID)
	}
}

func TestPageData_JSONSerialization(t *testing.T) {
	page := PageData{
		List:     []int{1, 2, 3},
		Total:    100,
		Page:     1,
		PageSize: 10,
	}

	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var parsed PageData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if parsed.Total != page.Total {
		t.Errorf("Total = %d, want %d", parsed.Total, page.Total)
	}
	if parsed.Page != page.Page {
		t.Errorf("Page = %d, want %d", parsed.Page, page.Page)
	}
	if parsed.PageSize != page.PageSize {
		t.Errorf("PageSize = %d, want %d", parsed.PageSize, page.PageSize)
	}
}
