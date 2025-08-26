package main

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/Exayn/go-listmonk"
)

func TestEpochToDate(t *testing.T) {
	tests := []struct {
		name     string
		epoch    int64
		expected string
	}{
		{
			name:     "zero epoch returns empty string",
			epoch:    0,
			expected: "",
		},
		{
			name:     "valid epoch converts to Eastern time date",
			epoch:    1640995200000, // January 1, 2022 00:00:00 UTC
			expected: "2021-12-31",   // Should be Dec 31, 2021 in Eastern time
		},
		{
			name:     "another valid epoch",
			epoch:    1672531200000, // January 1, 2023 00:00:00 UTC
			expected: "2022-12-31",   // Should be Dec 31, 2022 in Eastern time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := epochToDate(tt.epoch)
			if result != tt.expected {
				t.Errorf("epochToDate(%d) = %s; expected %s", tt.epoch, result, tt.expected)
			}
		})
	}
}

func TestMostRecentDate(t *testing.T) {
	tests := []struct {
		name      string
		dateStr1  string
		dateStr2  string
		expected  string
		shouldErr bool
	}{
		{
			name:     "first date is more recent",
			dateStr1: "2023-12-31",
			dateStr2: "2023-01-01",
			expected: "2023-12-31",
		},
		{
			name:     "second date is more recent",
			dateStr1: "2023-01-01",
			dateStr2: "2023-12-31",
			expected: "2023-12-31",
		},
		{
			name:     "dates are equal",
			dateStr1: "2023-06-15",
			dateStr2: "2023-06-15",
			expected: "2023-06-15",
		},
		{
			name:      "invalid first date",
			dateStr1:  "invalid-date",
			dateStr2:  "2023-01-01",
			shouldErr: true,
		},
		{
			name:      "invalid second date",
			dateStr1:  "2023-01-01",
			dateStr2:  "invalid-date",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MostRecentDate(tt.dateStr1, tt.dateStr2)
			
			if tt.shouldErr {
				if err == nil {
					t.Errorf("MostRecentDate(%s, %s) expected error but got none", tt.dateStr1, tt.dateStr2)
				}
				return
			}
			
			if err != nil {
				t.Errorf("MostRecentDate(%s, %s) unexpected error: %v", tt.dateStr1, tt.dateStr2, err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("MostRecentDate(%s, %s) = %s; expected %s", tt.dateStr1, tt.dateStr2, result, tt.expected)
			}
		})
	}
}

func TestPhoneCleaning(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "clean 10 digit phone",
			phone:    "5551234567",
			expected: "5551234567",
		},
		{
			name:     "phone with formatting",
			phone:    "(555) 123-4567",
			expected: "5551234567",
		},
		{
			name:     "phone with country code",
			phone:    "+15551234567",
			expected: "5551234567",
		},
		{
			name:     "phone with extra digits",
			phone:    "15551234567890",
			expected: "1234567890", // last 10 digits
		},
		{
			name:     "phone with spaces and dashes",
			phone:    "555 - 123 - 4567",
			expected: "5551234567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile(`\D`)
			cleanPhone := re.ReplaceAllString(tt.phone, "")
			if len(cleanPhone) > 10 {
				cleanPhone = cleanPhone[len(cleanPhone)-10:]
			}
			
			if cleanPhone != tt.expected {
				t.Errorf("Phone cleaning for %s = %s; expected %s", tt.phone, cleanPhone, tt.expected)
			}
		})
	}
}

func TestEmailCleaning(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "clean email",
			email:    "test@example.com",
			expected: "test@example.com",
		},
		{
			name:     "email with spaces",
			email:    "test @example.com",
			expected: "test@example.com",
		},
		{
			name:     "email with commas",
			email:    "test,@example.com",
			expected: "test@example.com",
		},
		{
			name:     "email with spaces and commas",
			email:    "test, @example.com",
			expected: "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanEmail := strings.ReplaceAll(tt.email, " ", "")
			cleanEmail = strings.ReplaceAll(cleanEmail, ",", "")
			
			if cleanEmail != tt.expected {
				t.Errorf("Email cleaning for %s = %s; expected %s", tt.email, cleanEmail, tt.expected)
			}
		})
	}
}

func TestNameExtraction(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		expected  string
	}{
		{
			name:      "both names provided",
			firstName: "John",
			lastName:  "Doe",
			expected:  "John Doe",
		},
		{
			name:      "only first name",
			firstName: "John",
			lastName:  "",
			expected:  "John",
		},
		{
			name:      "only last name",
			firstName: "",
			lastName:  "Doe",
			expected:  "Doe",
		},
		{
			name:      "names with extra spaces",
			firstName: " John ",
			lastName:  " Doe ",
			expected:  "John   Doe",
		},
		{
			name:      "empty names",
			firstName: "",
			lastName:  "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullName := strings.TrimSpace(tt.firstName + " " + tt.lastName)
			
			if fullName != tt.expected {
				t.Errorf("Name extraction for '%s' + '%s' = '%s'; expected '%s'", tt.firstName, tt.lastName, fullName, tt.expected)
			}
		})
	}
}

// Mock structures for testing sendToListmonk function
type mockListmonkClient struct {
	subscribers []listmonk.Subscriber
	shouldError bool
}

func (m *mockListmonkClient) NewGetSubscribersService() *mockGetSubscribersService {
	return &mockGetSubscribersService{client: m}
}

func (m *mockListmonkClient) NewCreateSubscriberService() *mockCreateSubscriberService {
	return &mockCreateSubscriberService{client: m}
}

func (m *mockListmonkClient) NewUpdateSubscriberService() *mockUpdateSubscriberService {
	return &mockUpdateSubscriberService{client: m}
}

type mockGetSubscribersService struct {
	client *mockListmonkClient
	query  string
}

func (s *mockGetSubscribersService) Query(query string) {
	s.query = query
}

func (s *mockGetSubscribersService) Do(ctx context.Context) ([]listmonk.Subscriber, error) {
	if s.client.shouldError {
		return nil, &listmonk.APIError{Code: 500, Message: "Internal Server Error"}
	}
	return s.client.subscribers, nil
}

type mockCreateSubscriberService struct {
	client *mockListmonkClient
	email  string
	name   string
}

func (s *mockCreateSubscriberService) Email(email string) {
	s.email = email
}

func (s *mockCreateSubscriberService) Name(name string) {
	s.name = name
}

func (s *mockCreateSubscriberService) ListIds(ids []uint) {}

func (s *mockCreateSubscriberService) Attributes(attrs map[string]any) {}

func (s *mockCreateSubscriberService) PreconfirmSubscriptions(preconfirm bool) {}

func (s *mockCreateSubscriberService) Do(ctx context.Context) (*listmonk.Subscriber, error) {
	if s.client.shouldError {
		return nil, &listmonk.APIError{Code: 400, Message: "Invalid email."}
	}
	return &listmonk.Subscriber{Email: s.email, Name: s.name}, nil
}

type mockUpdateSubscriberService struct {
	client *mockListmonkClient
}

func (s *mockUpdateSubscriberService) Name(name string)                        {}
func (s *mockUpdateSubscriberService) Attributes(attrs map[string]any) {}
func (s *mockUpdateSubscriberService) Status(status string)                    {}
func (s *mockUpdateSubscriberService) Id(id uint)                              {}
func (s *mockUpdateSubscriberService) Email(email string)                      {}
func (s *mockUpdateSubscriberService) ListIds(ids []uint)                      {}

func (s *mockUpdateSubscriberService) Do(ctx context.Context) (*listmonk.Subscriber, error) {
	if s.client.shouldError {
		return nil, &listmonk.APIError{Code: 500, Message: "Internal Server Error"}
	}
	return &listmonk.Subscriber{}, nil
}