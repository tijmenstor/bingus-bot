package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	token         string
	commandPrefix string
	soundsFolder  string
	commandsFile  string
	commandsMap   = make(map[string]string)
)

func init() {
	flag.StringVar(&token, "t", "", "Bot token")
	flag.StringVar(&commandPrefix, "p", "~", "Prefix for commands")
	flag.StringVar(&soundsFolder, "s", "sounds", "Folder where .mp3 sounds reside")
	flag.StringVar(&commandsFile, "c", "commands.json", "File containing all commands in JSON format")
	flag.Parse()
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if token == "" {
		log.Panic().Msg("No token provided. Please pass it using the -t flag.")
	}
	loadSounds()

	dg := setupDiscordGo()
	log.Info().Msg("Bot is running. Press CTRL-C to exit.")

	// Simple way to keep program running until CTRL-C is pressed.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

func setupDiscordGo() *discordgo.Session {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Panic().Err(err).Msg("Error creating Discord session.")
	}

	// Add the messageCreate func as a callback for the messageCreate events.
	dg.AddHandler(messageCreate)

	// Open the websocket and begin listening.
	err = dg.Open()
	if err != nil {
		log.Panic().Err(err).Msg("Error opening Discord session.")
	}
	log.Info().Msg("Discord session created.")
	return dg
}

type BotCommands struct {
	Commands []string `json:"commands"`
	FileName string   `json:"fileName"`
}

func loadSounds() {
	commandsJsonFile, err := os.Open(commandsFile)
	if err != nil {
		log.Panic().Err(err).Msg("Error opening commands file.")
	}
	defer commandsJsonFile.Close()

	byteValue, err := io.ReadAll(commandsJsonFile)
	if err != nil {
		log.Panic().Err(err).Msg("Error reading commands file.")
	}

	var botCommands []BotCommands
	err = json.Unmarshal(byteValue, &botCommands)
	if err != nil {
		log.Panic().Err(err).Msg("Error unmarshalling commands file")
	}

	for _, command := range botCommands {
		for _, commandName := range command.Commands {
			commandsMap[commandName] = command.FileName
		}
	}
	log.Info().Msg("All sounds loaded.")
}

// Handler that will be called whenever a Discord message is created
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Check if the message starts with the command prefix
	if !strings.HasPrefix(m.Content, commandPrefix) {
		return
	}

	// Find the channel that the message came from.
	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		return
	}

	// Find the guild for that channel.
	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		return
	}

	// Look for the message sender in that guild's current voice states.
	var guildID, channelID string
	for _, vs := range g.VoiceStates {
		if vs.UserID == m.Author.ID {
			guildID = g.ID
			channelID = vs.ChannelID
		}
	}

	// Fail if the message sender cannot be found in guild's current voice states.
	if guildID == "" || channelID == "" {
		return
	}

	prefixlessCommand := strings.TrimPrefix(m.Content, commandPrefix)
	if sound, ok := commandsMap[prefixlessCommand]; ok {
		playSound(s, guildID, channelID, fmt.Sprintf("%s/%s.mp3", soundsFolder, sound))
	}
}

func playSound(s *discordgo.Session, guildID, channelID, audioPath string) {
	voice, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		log.Error().Err(err).Msg("Error joining voice channel.")
		return
	}
	dgvoice.PlayAudioFile(voice, audioPath, make(chan bool))
	err = voice.Disconnect()
	if err != nil {
		log.Error().Err(err).Msg("Error disconnecting from voice channel.")
	}
	log.Info().Str("sound", audioPath).Msg("Sound played.")
}
