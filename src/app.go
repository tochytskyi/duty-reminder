package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const sheetName = "schedule"
const settingsSheetName = "settings"

var googleSheetsAPICredentials string
var slackToken string
var spreadsheetID string
var slackChannel string

func main() {
	initParams()

	sheetsService, err := connectToSheetsAPI(googleSheetsAPICredentials)
	if err != nil {
		log.Fatalf("Unable to connect to Sheets API: %v", err)
	}

	currentDate := time.Now().Format("2006-01-02")

	row, err := findRowByDate(sheetsService, currentDate)
	if err != nil {
		log.Fatalf("Error finding row by date: %v", err)
	}

	if row != nil {
		dutyNickname := row[2].(string)
		slackUserId, err := getDutyId(sheetsService, dutyNickname)

		if err != nil {
			log.Fatalf("Failed to find Slack user %s: %v", dutyNickname, err)
		}

		api := slack.New(slackToken)

		message := fmt.Sprintf("Hello <@%s>! This is your notification for today.", slackUserId)
		_, _, err = api.PostMessage(slackChannel, slack.MsgOptionText(message, false))

		if err != nil {
			log.Fatalf("Error sending message to Slack: %v", err)
		}
	} else {
		log.Println("No matching row found for the current date.")
	}
}

func connectToSheetsAPI(apiKey string) (*sheets.Service, error) {
	ctx := context.Background()

	sheetsService, err := sheets.NewService(ctx, option.WithCredentialsJSON([]byte(googleSheetsAPICredentials)))
	if err != nil {
		return nil, err
	}

	return sheetsService, nil
}

func findRowByDate(srv *sheets.Service, currentDate string) ([]interface{}, error) {
	readRange := fmt.Sprintf("%s!B1:D1000", sheetName)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, err
	}

	for _, row := range resp.Values {
		dateFrom := row[0].(string)
		dateTo := row[1].(string)

		if currentDate >= dateFrom && currentDate <= dateTo {
			return row, nil
		}
	}

	return nil, nil
}

func getDutyId(srv *sheets.Service, dutyNickname string) (string, error) {
	readRange := fmt.Sprintf("%s!A1:B1000", settingsSheetName)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return "", err
	}

	for _, row := range resp.Values {
		nickname := row[0].(string)
		userId := row[1].(string)

		if userId == "" {
			return "", errors.New("Slack user ID is empty")
		}

		if nickname == dutyNickname {
			return userId, nil
		}
	}

	return "", errors.New("Slack user ID not found")
}

func initParams() {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	googleSheetsAPICredentials = os.Getenv("GOOGLE_SHEETS_API_CREDENTIALS")
	if googleSheetsAPICredentials == "" {
		log.Fatal("GOOGLE_SHEETS_API_CREDENTIALS environment variable is not set")
	}

	slackToken = os.Getenv("SLACK_API_TOKEN")
	if slackToken == "" {
		log.Fatal("SLACK_API_TOKEN environment variable is not set")
	}

	spreadsheetID = os.Getenv("SPREADSHEET_ID")
	if spreadsheetID == "" {
		log.Fatal("SPREADSHEET_ID environment variable is not set")
	}

	slackChannel = os.Getenv("SLACK_CHANNEL_ID")
	if slackChannel == "" {
		log.Fatal("SLACK_CHANNEL_ID environment variable is not set")
	}
}
