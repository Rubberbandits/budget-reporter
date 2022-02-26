package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/99designs/keyring"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"gopkg.in/Iwark/spreadsheet.v2"
)

// BudgetConfig
type BudgetConfig struct {
	Email string `json:"email"`

	TestMode   bool       `json:"testmode"`
	IgnoreList IgnoreList `json:"ignore"`
}

type IgnoreList struct {
}

type MintTime time.Time

func (timestamp *MintTime) UnmarshalJSON(b []byte) error {
	uintTime, _ := strconv.ParseInt(string(b), 10, 64)
	*timestamp = MintTime(time.UnixMilli(uintTime))

	return nil
}

// TransactionData
type TransactionData struct {
	Date MintTime `json:"odate"`

	Note string `json:"note"`

	IsPercent   bool `json:"isPercent"`
	IsEdited    bool `json:"isEdited"`
	IsPending   bool `json:"isPending"`
	IsMatched   bool `json:"isMatched"`
	IsFirstDate bool `json:"isFirstDate"`
	IsDuplicate bool `json:"isDuplicate"`
	IsChild     bool `json:"isChild"`
	IsSpending  bool `json:"isSpending"`
	IsTransfer  bool `json:"isTransfer"`
	IsCheck     bool `json:"isCheck"`
	IsDebit     bool `json:"isDebit"`

	Amount float64 `json:"amount"`

	FinancialInstitution string `json:"fi"`
	TransactionType      uint   `json:"txnType"`
	NumberMatchedByRule  int    `json:"numberMatchedByRule"`

	Merchant string `json:"merchant"`

	Category string `json:"category"`
}

var Config BudgetConfig
var spService spreadsheet.Service
var drvService drive.Service

const TEMPLATE_ID string = "1ZEOkPYJtFnNoNa6fruTghp45q7A5UCUL2eeqGB3kLMU"
const DATE_FORMAT string = "01/02/2006"

// Functionality
func ProcessTransactions(transactions *[]TransactionData) {
	newFile := &drive.File{
		Name: time.Now().Local().Format(DATE_FORMAT) + " Budget Report",
	}

	newFile, err := drvService.Files.Copy(TEMPLATE_ID, newFile).Do()
	if err != nil {
		log.Fatal(err)
	}

	spread, err := spService.FetchSpreadsheet(newFile.Id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(spread.Properties.Title)

	txSheet, err := spread.SheetByIndex(1)
	if err != nil {
		log.Fatal(err)
	}

	lastMonth := time.Now().AddDate(0, -1, 0)
	expenseRow := 4
	incomeRow := 4

	for _, txData := range *transactions {
		curRow := expenseRow

		if txData.IsTransfer {
			continue
		}

		startColumn := 1

		if !txData.IsSpending {
			curRow = incomeRow
			startColumn = 6
		}

		txAmount := txData.Amount
		if txData.Amount < 0 {
			curRow = incomeRow
			startColumn = 6
			txAmount *= -1
		}

		txDate := time.Time(txData.Date)
		if txDate.Before(lastMonth) {
			continue
		}

		txSheet.Update(curRow, startColumn, txDate.Format(DATE_FORMAT))
		txSheet.Update(curRow, startColumn+1, "=TO_DOLLARS("+strconv.FormatFloat(txAmount, 'f', 2, 64)+")")
		txSheet.Update(curRow, startColumn+2, "=T(\""+txData.Merchant+"\")")
		txSheet.Update(curRow, startColumn+3, txData.Category)

		if startColumn == 6 {
			incomeRow++
		} else {
			expenseRow++
		}
	}

	txSheet.Synchronize()
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

	ring, _ := keyring.Open(keyring.Config{
		ServiceName: "mintapi",
	})

	password, _ := ring.Get(Config.Email)

	txData, mintErr := exec.Command("PATH=/usr/bin:$PATH", "mintapi", "--extended-transactions", "--headless", Config.Email, string(password.Data)).Output()
	if mintErr != nil {
		log.Fatal(mintErr)
	}

	//Load testing data
	// txData, fsErr := os.ReadFile("./test.json")
	// if fsErr != nil {
	// 	log.Fatal(fsErr)
	// }

	transactions := []TransactionData{}
	jsonErr := json.Unmarshal(txData, &transactions)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	ProcessTransactions(&transactions)
}
