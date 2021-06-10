package main

import (
	"context"
	"fmt"
	"log"

	"github.com/x0000ff/getpocket-go-sdk/pocket"
)

func main() {
	ctx := context.Background()

	CONSUMER_KEY := "<YOUR_KEY_HERE>"
	client, err := pocket.NewClient(CONSUMER_KEY)
	if err != nil {
		log.Fatal(err)
	}

	CALLBACK_URL := "<YOUR_CALLBACK_URL>"
	requestToken, err := client.GetRequestToken(ctx, CALLBACK_URL)
	if err != nil {
		log.Fatalf("failed to get request token: %s", err.Error())
	}
	log.Printf("Got request token: %s\n", requestToken)

	url, err := client.GetAuthorizationURL(requestToken, CALLBACK_URL)
	if err != nil {
		log.Fatalf("failed to get authorization url: %s", err.Error())
	}
	log.Printf("Got authorization URL: %s\n", url)

	fmt.Println("Open the URL and confirm the authorization")
	fmt.Println("Press any key to continue...")
	fmt.Scanln()

	authResponse, err := client.Authorize(ctx, requestToken)
	if err != nil {
		log.Fatalf("failed to authorize: %s", err)
	}
	fmt.Printf("Saved successfully!\n%v", authResponse)

	err = client.Add(ctx, pocket.AddInput{
		URL:         "https://github.com/zhashkevych/go-pocket-sdk",
		AccessToken: authResponse.AccessToken,
	})

	if err != nil {
		log.Fatalf("failed to add item: %s", err)
	}
}
