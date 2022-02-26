package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"gopkg.in/Iwark/spreadsheet.v2"
)

// BudgetConfig
type BudgetConfig struct {
	Email string `json:"email"`

	TestMode bool `json:"testmode"`
}

type MintTime time.Time

func (timestamp *MintTime) UnmarshalJSON(b []byte) error {
	uintTime, _ := strconv.ParseInt(string(b), 10, 64)
	*timestamp = MintTime(time.UnixMilli(uintTime))

	return nil
}

// TransactionData
type TransactionData struct {
	Date MintTime `json:"date"`

	Description         string `json:"description"`
	OriginalDescription string `json:"original_description"`

	Amount          float32 `json:"amount"`
	TransactionType string  `json:"transaction_type"`

	Category    string `json:"category"`
	AccountName string `json:"account_name"`

	Labels string `json:"labels"`
	Notes  string `json:"notes"`
}

var Config BudgetConfig
var spService spreadsheet.Service
var drvService drive.Service

const TEMPLATE_ID string = "1ZEOkPYJtFnNoNa6fruTghp45q7A5UCUL2eeqGB3kLMU"

// Functionality
func ProcessTransactions(transactions *[]TransactionData) {
	newFile := &drive.File{
		Name: "Template Copy Haha",
	}

	newFile, err := drvService.Files.Copy(TEMPLATE_ID, newFile).Do()
	if err != nil {
		log.Fatal(err)
	}

	spread, err := spService.FetchSpreadsheet(newFile.Id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(spread)
}

func LoadConfig() {
	configJson, fsErr := os.ReadFile("./config/config.json")
	if fsErr != nil {
		log.Fatal(fsErr)
	}

	Config = BudgetConfig{}
	json.Unmarshal(configJson, &Config)
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	// startArgs := os.Args[1:]
	// startDate := startArgs[0]
	// endDate := startArgs[1]

	LoadConfig()

	data, err := os.ReadFile("./config/creds.json")
	if err != nil {
		return
	}

	ctx := context.Background()

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(data, drive.DriveScope, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := getClient(config)

	service := spreadsheet.NewServiceWithClient(client)
	spService = *service

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	drvService = *srv

	// ring, _ := keyring.Open(keyring.Config{
	// 	ServiceName: "mintapi",
	// })

	// password, _ := ring.Get(Config.Email)

	// txData, mintErr := exec.Command("mintapi", "-t", Config.Email, string(password.Data)).Output()
	// if mintErr != nil {
	// 	log.Fatal(mintErr)
	// }

	// Load testing data
	txData, fsErr := os.ReadFile("./test.json")
	if fsErr != nil {
		log.Fatal(fsErr)
	}

	transactions := []TransactionData{}
	jsonErr := json.Unmarshal(txData, &transactions)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	ProcessTransactions(&transactions)
}
