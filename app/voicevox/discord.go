package voicevox

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gammazero/deque"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"layeh.com/gopus"
)

type ManagedDiscordVoiceConnection struct {
	GuildID   string
	ChannelID string
	dvc       *DiscordVoiceConnection
	vc        *discordgo.VoiceConnection
	app       *VoiceVox
}

func StartManagedDiscordVoiceConnection(appLogger *zap.Logger, sess *discordgo.Session, guildID, channelID string, voiceVox *VoiceVox, replaceFn func(input string) string) (*ManagedDiscordVoiceConnection, error) {
	vc, err := sess.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, err
	}

	return &ManagedDiscordVoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		dvc:       StartDiscordVoiceConnection(appLogger, vc, voiceVox, replaceFn),
		vc:        vc,
		app:       voiceVox,
	}, nil
}

func (m *ManagedDiscordVoiceConnection) Close() error {
	m.dvc.Quit()
	if err := m.vc.Disconnect(); err != nil {
		m.vc.Close()
		return err
	}
	m.vc.Close()
	return nil
}

func (m *ManagedDiscordVoiceConnection) Speak(speakerID int, waitSpeaked bool, content string) {
	m.dvc.Speak(speakerID, waitSpeaked, content)
}

func (m *ManagedDiscordVoiceConnection) GetSpeakers(nameFilter string, waitResume bool) ([]VoiceSpeaker, error) {
	return m.app.GetSpeakers(nameFilter, waitResume)
}

type DiscordVoiceConnection struct {
	genQueueQuit  chan<- *sync.WaitGroup
	generateQueue chan<- generateVoiceArgs
}

type generateVoiceArgs struct {
	speakerID int
	content   string
	wg        *sync.WaitGroup
}

func StartDiscordVoiceConnection(appLogger *zap.Logger, vc *discordgo.VoiceConnection, voiceVox *VoiceVox, replaceFn func(input string) string) *DiscordVoiceConnection {

	var (
		genQueueQuit  = make(chan *sync.WaitGroup)
		generateQueue = make(chan generateVoiceArgs)
	)

	speakUUIDQueue := deque.New[string]()
	speakUUIDQueueLock := sync.Mutex{}

	go func() {
		for {
			select {
			case wg := <-genQueueQuit:
				defer wg.Done()
				return

			case args := <-generateQueue:
				func() {
					defer func() {
						// finalize operation for call event ended
						if args.wg != nil {
							args.wg.Done()
						}
					}()

					appLogger.Debug("voicevox generate request recieved", zap.String("content", args.content))

					wav, err := voiceVox.GenerateVoice(args.content, args.speakerID, false)
					if err != nil {
						appLogger.Error("failed to generate voice", zap.Int("speakerId", args.speakerID), zap.Error(err))
						return
					}
					appLogger.Debug("voicevox generate finished", zap.String("content", args.content))

					key := uuid.NewString()
					func() {
						speakUUIDQueueLock.Lock()
						defer speakUUIDQueueLock.Unlock()
						speakUUIDQueue.PushBack(key)
					}()

					go func() {
						defer wav.Close()
						appLogger.Debug("ffmpeg convert request start", zap.String("content", args.content))
						ffmpegout, process, err := ffmpegConvert(wav)
						if err != nil {
							appLogger.Error("convert error by ffmpeg", zap.Error(err))
							return
						}
						defer ffmpegout.Close()
						appLogger.Debug("ffmpeg convert finished", zap.String("content", args.content))

						for func() string {
							speakUUIDQueueLock.Lock()
							defer speakUUIDQueueLock.Unlock()
							return speakUUIDQueue.Front()
						}() != key {
							time.Sleep(50 * time.Millisecond)
						}

						if err := playAudio(appLogger, vc, ffmpegout, process); err != nil {
							appLogger.Error("cannot play audio", zap.Error(err))
						}
						func() {
							speakUUIDQueueLock.Lock()
							defer speakUUIDQueueLock.Unlock()

							speakUUIDQueue.PopFront()
						}()
					}()
				}()
			}
		}
	}()

	return &DiscordVoiceConnection{
		genQueueQuit:  genQueueQuit,
		generateQueue: generateQueue,
	}
}

func (d *DiscordVoiceConnection) Quit() {
	if d.genQueueQuit != nil {
		defer func() { d.genQueueQuit = nil }()
		wg := sync.WaitGroup{}
		wg.Add(1)
		d.genQueueQuit <- &wg
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

type Killer interface {
	Kill() error
}

const (
	frameRate = 48000
	frameSize = 960
	channels  = 2
	maxBytes  = 3840
)

func ffmpegConvert(wavReader io.Reader) (ffmpegout io.ReadCloser, processKiller Killer, err error) {

	run := exec.Command("ffmpeg", "-i", "pipe:", "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")
	run.Stdin = wavReader
	// binary := bytes.NewBuffer([]byte{})
	// run.Stdout = binary
	ffmpegout, err = run.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot ffmpeg stdout pipe: %w", err)
	}

	if err = run.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed run ffmpeg command: %w", err)
	}

	return ffmpegout, run.Process, nil
	// return io.NopCloser(binary), run.Process, nil
}

func playAudio(logger *zap.Logger, vcConn *discordgo.VoiceConnection, ffmpegout io.Reader, processKiller Killer) error {

	ffmpegbuf := bufio.NewReaderSize(ffmpegout, 16384)
	if err := vcConn.Speaking(true); err != nil {
		return fmt.Errorf("could not speaking: %w", err)
	}
	defer func() {
		if err := vcConn.Speaking(false); err != nil {
			logger.Error("could not stop speaking: %w", zap.Error(err))
		}
	}()

	send := make(chan []int16, 2)
	defer close(send)

	close := make(chan bool)
	go func() {
		if err := sendPCM(vcConn, send); err != nil {
			logger.Error("playing audio error", zap.Error(err))
		}
		close <- true
	}()

	for {
		audiobuf := make([]int16, frameSize*channels)
		err := binary.Read(ffmpegbuf, binary.LittleEndian, &audiobuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error from ffmpeg: %w", err)
		}

		select {
		case send <- audiobuf:
		case <-close:
			processKiller.Kill()
			return nil
		}
	}
}

func sendPCM(vcConn *discordgo.VoiceConnection, pcm <-chan []int16) error {
	if pcm == nil {
		return nil
	}

	opusEncoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		return fmt.Errorf("cannot make opus encoder: %w", err)
	}

	for {

		recv, ok := <-pcm
		if !ok {
			return nil
		}

		opus, err := opusEncoder.Encode(recv, frameSize, maxBytes)
		if err != nil {
			return fmt.Errorf("opus encoding error: %w", err)
		} else if !vcConn.Ready {
			return fmt.Errorf("voice connection is not ready")
		} else if vcConn.OpusSend == nil {
			return fmt.Errorf("opus sender is nil")
		} else {
			vcConn.OpusSend <- opus
		}
	}
}
