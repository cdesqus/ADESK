package main

import (
	"fmt"
	"os"
	"log"
	
	"ai-desk/internal/ai"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY is empty in .env")
	}

	fmt.Println("API Key length:", len(apiKey))

	client := ai.NewOpenAIClient(apiKey, nil)
	reply, err := client.GenerateAutoReply("My laptop is not turning on. Please help.", "Budi", "Company XYZ", 123)
	if err != nil {
		log.Fatalf("AI Generation Failed: %v", err)
	}

	fmt.Println("AI Reply Success!")
	fmt.Println("-----------------")
	fmt.Println(reply)
}
