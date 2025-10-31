package token

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockParser is a mock implementation of the Parser interface
type MockParser struct {
	mock.Mock
}

func (m *MockParser) ParseString(str string) (*User, error) {
	args := m.Called(str)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockParser) ParseHeader(headers http.Header, header string) (*User, error) {
	args := m.Called(headers, header)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func TestMiddleware(t *testing.T) {
	t.Run("successful authentication", func(t *testing.T) {
		mockParser := new(MockParser)
		user := &User{
			ID:     "user-123",
			Email:  "test@example.com",
			Groups: []string{"users"},
		}
		mockParser.On("ParseHeader", mock.Anything, "Authorization").Return(user, nil)

		var capturedEmail string
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedEmail = GetEmail(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		middleware := Middleware(mockParser)
		handler := middleware(nextHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "test@example.com", capturedEmail)
		mockParser.AssertExpectations(t)
	})

	t.Run("authentication failure", func(t *testing.T) {
		mockParser := new(MockParser)
		mockParser.On("ParseHeader", mock.Anything, "Authorization").Return(nil, errors.New("invalid token"))

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Next handler should not be called")
		})

		middleware := Middleware(mockParser)
		handler := middleware(nextHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		mockParser.AssertExpectations(t)
	})
}
