package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Exayn/go-listmonk"
)

// OAuthResponse represents the expected JSON response from the OAuth token endpoint
type OAuthResponse struct {
	Value string `json:"value"`
}

// Customer represents each customer in the response array
type Customer struct {
	ServerID      int       `json:"serverId"`
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	EmailAddress  string    `json:"emailAddress"`
	Point         int       `json:"point"`
	PhoneNumbers  []string  `json:"phoneNumbers"`
	LastVisitDate int64     `json:"lastVisitDate"`
	Addresses     []Address `json:"addresses"`
}

// Address represents a customer's address
type Address struct {
	Address  string `json:"address"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
	ZipCode  string `json:"zipCode"`
}

func main() {
	// Load environment variables
	edgeservPOSHost := os.Getenv("EDGESERV_POS_HOST")
	restaurantCode := os.Getenv("RESTAURANT_CODE")
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	listmonkHost := os.Getenv("LISTMONK_HOST")
	listmonkUser := os.Getenv("LISTMONK_USER")
	listmonkToken := os.Getenv("LISTMONK_TOKEN")

	// Step 1: Get OAuth Token
	token := getOAuthToken(edgeservPOSHost, restaurantCode, clientID, clientSecret, username, password)

	// Step 2: Get Customer Data
	customers := getCustomerData(edgeservPOSHost, restaurantCode, token)

	// Step 3: Process and Send to Listmonk
	listmonkClient := listmonk.NewClient(listmonkHost, &listmonkUser, &listmonkToken)
	for _, customer := range customers {
		if customer.EmailAddress != "" {
			sendToListmonk(listmonkClient, customer)
		}
	}
}

// Function to retrieve OAuth token
func getOAuthToken(edgeservPOSHost, restaurantCode, clientID, clientSecret, username, password string) string {
	tokenURL := fmt.Sprintf("%s/%s/oauth/token?grant_type=password&client_id=%s&client_secret=%s&username=%s&password=%s",
		edgeservPOSHost, restaurantCode, clientID, clientSecret, username, password)

	resp, err := http.Get(tokenURL)
	if err != nil {
		log.Fatalf("Error fetching token: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading token response: %v", err)
	}

	var tokenResp OAuthResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		log.Fatalf("Error parsing token JSON: %v", err)
	}

	return tokenResp.Value
}

// Function to retrieve customer data
func getCustomerData(edgeservPOSHost, restaurantCode, token string) []Customer {
	customerListURL := fmt.Sprintf("%s/%s/backofhouse/customer/list", edgeservPOSHost, restaurantCode)

	requestBody := map[string]interface{}{
		"serverId":        nil,
		"searchValue":     "",
		"addressRequired": false,
		"zipRequired":     false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatalf("Error marshalling request body: %v", err)
	}

	req, err := http.NewRequest("POST", customerListURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error creating customer request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error fetching customer list: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading customer response: %v", err)
	}

	var customers []Customer
	if err := json.Unmarshal(body, &customers); err != nil {
		log.Fatalf("Error parsing customer JSON: %v", err)
	}

	return customers
}

// Function to send customer data to FluentCRM
func sendToListmonk(client *listmonk.Client, customer Customer) {
	// Extract and clean phone number
	phone := ""
	if len(customer.PhoneNumbers) > 0 {
		re := regexp.MustCompile(`\D`)
		phone = re.ReplaceAllString(customer.PhoneNumbers[0], "")
		if len(phone) > 10 {
			phone = phone[len(phone)-10:]
		}
	}
	// Extract and clean name
	fullName := strings.TrimSpace(customer.FirstName + " " + customer.LastName)
	// Clean email
	customer.EmailAddress = strings.ReplaceAll(customer.EmailAddress, " ", "")
	customer.EmailAddress = strings.ReplaceAll(customer.EmailAddress, ",", "")

	// Convert LastVisitDate to Eastern Time
	lastVisit := epochToDate(customer.LastVisitDate)

	// Extract ZIP
	zipCode := ""
	if len(customer.Addresses) > 0 {
		zipCode = customer.Addresses[0].ZipCode
	}

	subAttribs := map[string]interface{}{
		"lastVisit": lastVisit,
		"zipCode":   zipCode,
		"phone":     phone,
	}

	log.Printf("Querying for subscriber: %s", customer.EmailAddress)
	gSubSvc := client.NewGetSubscribersService()
	gSubSvc.Query(fmt.Sprintf("email ILIKE '%s'", customer.EmailAddress))
	subList, err := gSubSvc.Do(context.Background())
	if err != nil {
		log.Fatalf("Error getting subscriber: %v", err)
	}

	if len(subList) == 0 {
		// New subscriber
		cSubSvc := client.NewCreateSubscriberService()
		cSubSvc.Email(customer.EmailAddress)
		cSubSvc.Name(fullName)
		cSubSvc.ListIds([]uint{3})
		cSubSvc.Attributes(subAttribs)
		sub, err := cSubSvc.Do(context.Background())
		if err != nil {
			if apiErr, ok := err.(*listmonk.APIError); ok && apiErr.Code == 400 && apiErr.Message == "Invalid email." {
				log.Printf("Invalid email while creating subscriber: %s", customer.EmailAddress)
				return
			} else {
				log.Fatalf("Error creating subscriber: %v", err)
			}
		}
		log.Printf("Successfully created subscriber: %s", sub.Email)
	} else {
		sub := subList[0]

		nameMatch := sub.Name == fullName

		zipCodeMatch := sub.Attributes["zipCode"] == subAttribs["zipCode"]
		phoneMatch := sub.Attributes["phone"] == subAttribs["phone"]

		lastVisitMatch := true
		if sub.Attributes["lastVisitMatch"] != nil && subAttribs["lastVisitMatch"] != nil {
			mostRecentDate, err := MostRecentDate(sub.Attributes["lastVisitMatch"].(string), subAttribs["lastVisitMatch"].(string))
			if err != nil {
				log.Printf("Unable to parse date: %v", err)
			}

			lastVisitMatch = sub.Attributes["lastVisit"] == mostRecentDate
		}

		if !nameMatch || !lastVisitMatch || !zipCodeMatch || !phoneMatch {
			// Update needed
			var ids []uint
			for _, s := range sub.Lists {
				ids = append(ids, s.Id)
			}
			uSubSvc := client.NewUpdateSubscriberService()
			uSubSvc.Name(fullName)
			uSubSvc.Attributes(subAttribs)
			uSubSvc.Status(sub.Status)
			uSubSvc.Id(sub.Id)
			uSubSvc.Email(sub.Email)
			uSubSvc.ListIds(ids)
			_, err := uSubSvc.Do(context.Background())
			if err != nil {
				log.Fatalf("Error updating subscriber: %v", err)
			}
			log.Printf("Successfully updated subscriber: %s", sub.Email)
		}
	}
}

// Function to convert Epoch time to Eastern Time
func epochToDate(epoch int64) string {
	if epoch == 0 {
		return ""
	}
	loc, _ := time.LoadLocation("America/New_York")
	t := time.Unix(epoch/1000, 0).In(loc)
	return t.Format("2006-01-02")
}

func MostRecentDate(dateStr1, dateStr2 string) (string, error) {
	layout := "2006-01-02"
	date1, err1 := time.Parse(layout, dateStr1)
	date2, err2 := time.Parse(layout, dateStr2)

	if err1 != nil {
		return "", fmt.Errorf("failed to parse date1: %w", err1)
	}
	if err2 != nil {
		return "", fmt.Errorf("failed to parse date2: %w", err2)
	}

	if date1.After(date2) {
		return dateStr1, nil
	}
	return dateStr2, nil
}
