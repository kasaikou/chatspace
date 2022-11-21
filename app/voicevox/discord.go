package voicevox

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type ManagedDiscordVoiceConnection struct {
	GuildID   string
	ChannelID string
	dvc       *DiscordVoiceConnection
	vc        *discordgo.VoiceConnection
}

func StartManagedDiscordVoiceConnection(appLogger zerolog.Logger, sess *discordgo.Session, guildID, channelID string, voiceVox *VoiceVox, replaceFn func(input string) string) (*ManagedDiscordVoiceConnection, error) {
	vc, err := sess.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, err
	}

	return &ManagedDiscordVoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		dvc:       StartDiscordVoiceConnection(appLogger, vc, voiceVox, replaceFn),
		vc:        vc,
	}, nil
}

func (m *ManagedDiscordVoiceConnection) Close() {
	m.dvc.Quit()
	if err := m.vc.Disconnect(); err != nil {
		m.vc.Close()
		return
	}
	m.vc.Close()
}

func (m *ManagedDiscordVoiceConnection) Speak(speakerID int, waitSpeaked bool, content string) {
	m.dvc.Speak(speakerID, waitSpeaked, content)
}

type DiscordVoiceConnection struct {
	quit          chan<- *sync.WaitGroup
	generateQueue chan<- generateVoiceArgs
}

type generateVoiceArgs struct {
	speakerID int
	content   string
	wg        *sync.WaitGroup
}

func StartDiscordVoiceConnection(appLogger zerolog.Logger, vc *discordgo.VoiceConnection, voiceVox *VoiceVox, replaceFn func(input string) string) *DiscordVoiceConnection {

	type speakQueueArgs struct {
		filename string
		wg       *sync.WaitGroup
	}

	var (
		quit          = make(chan *sync.WaitGroup)
		generateQueue = make(chan generateVoiceArgs)
		speakQueue    = make(chan speakQueueArgs)
	)

	go func(queue chan<- speakQueueArgs) {
		for {
			select {
			case wg := <-quit:
				defer wg.Done()
				return

			case args := <-generateQueue:
				sended := func() (sended bool) {
					if replaceFn != nil {
						args.content = replaceFn(args.content)
					}

					tempDir := filepath.Join(filepath.Dir(os.Args[0]), "./.tmp/")
					wav, err := voiceVox.GenerateVoice(args.content, args.speakerID, false)
					if err != nil {
						appLogger.Error().Err(err).Int("speakerId", args.speakerID).Msg("failed to generate voice")
						return false
					}
					defer wav.Close()

					if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
						if err != os.ErrExist {
							appLogger.Error().Err(err).Msg("failed to mkdir")
							return false
						}
					}

					fileName := filepath.Join(tempDir, uuid.NewString()+".wav")
					file, err := os.Create(fileName)
					if err != nil {
						appLogger.Error().Err(err).Str("filename", fileName).Msg("failed to create wave file")
						return false
					}
					defer file.Close()

					if _, err := io.Copy(file, wav); err != nil {
						appLogger.Error().Err(err).Str("filename", fileName).Msg("failed to save wave file")
						return false
					}

					select {
					case speakQueue <- speakQueueArgs{
						filename: fileName,
						wg:       args.wg,
					}:
						appLogger.Debug().Msg("send speak file")
						return true
					default:
						os.Remove(fileName)
						return false
					}
				}()
				if !sended {
					if args.wg != nil {
						args.wg.Done()
					}
				}
			}
		}
	}(speakQueue)

	go func(queue <-chan speakQueueArgs) {
		for {
			select {
			case wg := <-quit:
				defer wg.Done()
				return
			case args := <-queue:
				dgvoice.PlayAudioFile(vc, args.filename, make(chan bool))
				os.Remove(args.filename)
				if args.wg != nil {
					args.wg.Done()
				}
			}
		}
	}(speakQueue)

	return &DiscordVoiceConnection{
		quit:          quit,
		generateQueue: generateQueue,
	}
}

func (d *DiscordVoiceConnection) Quit() {
	if d.quit != nil {
		defer func() { d.quit = nil }()
		wg := sync.WaitGroup{}
		for i := 0; i < 2; i++ {
			wg.Add(1)
			d.quit <- &wg
		}
		wg.Wait()
	}
}

func (d *DiscordVoiceConnection) Speak(speakerID int, waitSpeaked bool, content string) {
	var wg *sync.WaitGroup
	if waitSpeaked {
		wg = &sync.WaitGroup{}
		wg.Add(1)
	}

	d.generateQueue <- generateVoiceArgs{
		wg:        wg,
		speakerID: speakerID,
		content:   content,
	}

	if wg != nil {
		wg.Wait()
	}
}
