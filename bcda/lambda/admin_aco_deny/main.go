package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jackc/pgx/v5"

	log "github.com/sirupsen/logrus"
)

// var isTesting = os.Getenv("IS_TESTING") == "true"

type Payload struct {
	DenyACOIDs []string `json:"deny_aco_ids"`
	// Amount float64 `json:"amount"`
	// Item   string  `json:"item"`
}

func main() {
	// if isTesting {
	// 	var addresses, err = updateIpSet(context.Background())
	// 	if err != nil {
	// 		log.Error(err)
	// 	} else {
	// 		log.Println(addresses)
	// 	}
	// } else {
	// 	lambda.Start(handler)
	// }
	lambda.Start(handler)
}

func handler(ctx context.Context, event json.RawMessage) error {
	log.SetFormatter(&log.JSONFormatter{
		DisableHTMLEscape: true,
		TimestampFormat:   time.RFC3339Nano,
	})
	log.Info("Starting ACO Deny administrative task")

	var payload Payload
	err := json.Unmarshal(event, &payload)
	if err != nil {
		log.Errorf("Failed to unmarshal event: %v", err)
		return err
	}

	err = handleACODenies(ctx, payload)
	if err != nil {
		return err
	}

	log.Info("Completed ACO Deny administrative task")

	return nil
}

func handleACODenies(ctx context.Context, payload Payload) error {
	dbURL, err := getDBURL()
	if err != nil {
		log.Errorf("Unable to extract DB URL from parameter store: %+v", err)
		return err
	}

	// slack start message, mention env

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		log.Errorf("Unable to connect to database: %+v", err)
		return err
	}
	defer conn.Close(ctx)

	err = denyACOs(ctx, conn, payload)
	if err != nil {
		log.Errorf("Error finding and denying ACOs: %+v", err)
		// slack failure message
		return err
	}

	// slack success message

	return nil
}
