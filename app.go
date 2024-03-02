package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/PullRequestInc/go-gpt3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
)

type Config struct {
	Telegram struct {
		ApiToken string `mapstructure:"apiKey"`
	} `mapstructure:"telegram"`
	OpenAI struct {
		ApiKey string `mapstructure:"apiKey"`
	} `mapstructure:"openai"`
	Settings struct {
		PreambleText    string `mapstructure:"preambleText"`
		PreambleTextDan string `mapstructure:"preambleTextDAN"`
		DebugMode       bool   `mapstructure:"debugMode"`
	} `mapstructure:"settings"`
}

func LoadConfig(path string) (c Config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(path)

	viper.AutomaticEnv()

	err = viper.ReadInConfig()

	if err != nil {
		return
	}

	err = viper.Unmarshal(&c)
	return
}

func sendChatGPT(apiKey, sendText string) string {
	ctx := context.Background()
	client := gpt3.NewClient(apiKey)

	var response string

	err := client.CompletionStreamWithEngine(ctx, "gpt-3.5-turbo-instruct", gpt3.CompletionRequest{
		Prompt:      []string{sendText},
		MaxTokens:   gpt3.IntPtr(50),
		Temperature: gpt3.Float32Ptr(0),
	}, func(res *gpt3.CompletionResponse) {
		response += res.Choices[0].Text
	})
	if err != nil {
		log.Println(err)
		return "ChatGPT is not available"
	}
	return response
}

func initializeBot(config Config) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(config.Telegram.ApiToken)
	if err != nil {
		return nil, err
	}
	bot.Debug = config.Settings.DebugMode
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return bot, nil
}

func getUpdates(bot *tgbotapi.BotAPI) (tgbotapi.UpdatesChannel, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return bot.GetUpdatesChan(u)
}

func processUpdate(update tgbotapi.Update, config Config) (int64, string, error) {
	var userPrompt, gptPrompt string

	if strings.HasPrefix(update.Message.Text, "/topic") {
		userPrompt = strings.TrimPrefix(update.Message.Text, "/topic ")
		gptPrompt = config.Settings.PreambleText + "TOPIC: "
	} else if strings.HasPrefix(update.Message.Text, "/phrase") {
		userPrompt = strings.TrimPrefix(update.Message.Text, "/phrase ")
		gptPrompt = config.Settings.PreambleText + "PHRASE: "
	} else if strings.HasPrefix(update.Message.Text, "/anything") {
		userPrompt = strings.TrimPrefix(update.Message.Text, "/anything ")
		gptPrompt = config.Settings.PreambleTextDan + "QUESTION: "
	} else {
		return update.Message.Chat.ID, "", nil
	}

	if userPrompt != "" {
		gptPrompt += userPrompt
		res := sendChatGPT(config.OpenAI.ApiKey, gptPrompt)
		return update.Message.Chat.ID, res, nil
	} else {
		return update.Message.Chat.ID, "Please, enter your topic or phrase", nil
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, messageID int) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = messageID
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Error:", err)
	}
}

func main() {
	config, err := LoadConfig(".")
	if err != nil {
		panic(fmt.Errorf("fatal error with config.yaml: %w", err))
	}

	bot, err := initializeBot(config)
	if err != nil {
		log.Panic(err)
	}

	updates, err := getUpdates(bot)
	if err != nil {
		log.Fatalf("Error getting updates: %s", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID, text, err := processUpdate(update, config)
		if err != nil {
			log.Println("Error:", err)
			continue
		}

		if text != "" {
			sendMessage(bot, chatID, text, update.Message.MessageID)
		}
	}
}
