package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Collection of all Bot Settings
type config struct {
	Ready             bool
	BotToken          string
	ListenAllChannels bool
	ChannelList       []string
}

var regex *regexp.Regexp
var loadedBotConfig config

func main() {
	fmt.Println("Bot init!")

	loadedBotConfig = botPrep()

	dg, err := discordgo.New("Bot " + loadedBotConfig.BotToken)

	if err != nil {
		fmt.Println("Error Initializing Bot", err)
	}

	dg.AddHandler(messageCreate)
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	err = dg.Open()
	if err != nil {
		fmt.Println("Error Connecting to Discord ", err)
	}

	fmt.Println("Bot is Running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-sc

	fmt.Println("Bot shutting down")
	dg.Close()
}

// Does all the required preparations for the bot to run
func botPrep() config {
	//Seeding the random library with the current time
	rand.Seed(time.Now().UnixNano())

	//Reading the config file for the bot
	file, err := os.Open("config.cfg")

	if err != nil {
		writeDefaultConfig()
		os.Exit(1337)
	}

	defer func() {
		if err = file.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	b, err := io.ReadAll(file)
	checkErr(err)

	var configFile = string(b[:])
	configLines := strings.Split(configFile, "\n")

	if configLines[0] == "false" {
		fmt.Println("Config file is not Configured correctly")
		os.Exit(1336)
	}

	var loadedConfig config
	loadedConfig.Ready = true
	loadedConfig.BotToken = configLines[1]

	if configLines[2] == "all" {
		loadedConfig.ListenAllChannels = true
	} else {
		loadedConfig.ListenAllChannels = false
		loadedConfig.ChannelList = strings.Split(configLines[2], `,`)
	}

	//Compiling the regex for the message reader
	regex, err = regexp.Compile(`\/(r|roll) ([0-9]*)d([0-9]+)\+?([0-9]?)`)
	checkErr(err)

	return loadedConfig
}

// checks if err is nil if not writes an error message and panics
func checkErr(err error) {
	if err != nil {
		fmt.Println("Encountered Error: ", err)
		panic(err)
	}
}

// Handles Received Bot Messages
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Only commands are checked further
	if m.Content[0] != '/' {
		return
	}

	//if set to listen to all channels continue
	//if not check that the message was posted to a channel the bot is listening to or return out of the function
	if !loadedBotConfig.ListenAllChannels {
		var isInValidChannel = false
		for _, channelID := range loadedBotConfig.ChannelList {
			if m.ChannelID == channelID {
				isInValidChannel = true
			}
		}

		if !isInValidChannel {
			return
		}
	}

	//Message is a valid command continue processing it

	arr := regex.FindStringSubmatch(m.Content)
	if arr == nil {
		return
	}

	var diceCount, diceSides, modifier int

	var err error

	if arr[2] == "" {
		diceCount = 1
	} else {
		diceCount, err = strconv.Atoi(arr[2])
		if err != nil {
			fmt.Println("conversion error dice count ", err)
			return
		}
	}

	diceSides, err = strconv.Atoi(arr[3])
	if err != nil {
		fmt.Println("conversion error Dice sides ", err)
		return
	}

	if arr[4] == "" {
		modifier = 0
	} else {
		modifier, err = strconv.Atoi(arr[4])
		if err != nil {
			fmt.Println("conversion error modifier", err)
			return
		}
	}

	var throwRes = modifier
	var explanation = "("
	for i := 0; i < diceCount; i++ {
		throw := rand.Intn(diceSides) + 1
		throwRes += throw

		if i != 0 {
			explanation += " + "
		}

		explanation += formatThrow(throw, diceSides)
	}
	explanation += ") + " + strconv.Itoa(modifier) + " = " + strconv.Itoa(throwRes)

	fmt.Println(explanation)
	s.ChannelMessageSend(m.ChannelID, explanation)
}

// Formats a throw in bold if the result is the max possible with the dice or in italic if it is a nat 1
func formatThrow(throw int, sides int) string {
	if throw == 1 {
		return "*" + strconv.Itoa(throw) + "*"
	}
	if throw == sides {
		return "**" + strconv.Itoa(throw) + "**"
	}
	return strconv.Itoa(throw)
}

// Writes a new config file with explanation value
func writeDefaultConfig() {
	var defaultConfig = "false\n<BotToken>\n<all or comma seperated list of channel ids>"
	os.WriteFile("config.cfg", []byte(defaultConfig), os.ModeAppend)
}
