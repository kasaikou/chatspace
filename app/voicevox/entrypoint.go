package voicevox

import (
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/streamwest-1629/permanent-apps/lib/voicevox"
)

type InitConfig = voicevox.InitConfig

type VoiceVox struct {
	client      *voicevox.Client
	restartLock sync.RWMutex
	logger      zerolog.Logger
	config      voicevox.InitConfig
	genQueue    chan<- generateSpeakerConfig
	reloadQueue chan<- func()
	statusQueue chan<- statusMonitor
	quit        chan<- *sync.WaitGroup
}

type VoiceSpeaker struct {
	Name      string
	Character string
	Style     string
	Id        int
}

type generateSpeakerConfig struct {
	Text       string
	SpeakerId  int
	waitResume bool
	Receiver   func(io.ReadCloser, error)
}

type loadInfo struct {
	speakerIdxNameMap map[string]int
	speakerIdxIdMap   map[int]int
	speakers          []VoiceSpeaker
}

type status struct {
	info loadInfo
}

type statusMonitor struct {
	receiver func(status)
}

var (
	ErrRestarting     = errors.New("voicevox is restarting")
	ErrUnknownSpeaker = errors.New("unknown speaker")
)

func Start(appLogger zerolog.Logger, corePath, jTalkDir string, config voicevox.InitConfig) (*VoiceVox, error) {

	client, err := voicevox.LoadLib(corePath, jTalkDir)
	if err != nil {
		return &VoiceVox{}, err
	}

	var (
		genQueue    = make(chan generateSpeakerConfig)
		reloadQueue = make(chan func(), 1)
		statusQueue = make(chan statusMonitor)
		quit        = make(chan *sync.WaitGroup, 1)
	)

	v := VoiceVox{
		client:      client,
		config:      config,
		logger:      appLogger,
		genQueue:    genQueue,
		reloadQueue: reloadQueue,
		statusQueue: statusQueue,
		quit:        quit,
	}

	v.restartLock.Lock()
	v.reloadQueue <- v.restartLock.Unlock

	go func() {
		status := status{}
		// resumeQueue := v.reload(&status)

		for {
			reloadQueueReceiver := (<-chan func())(reloadQueue)
			statusQueueReceiver := (<-chan statusMonitor)(statusQueue)
			genQueueReceiver := (<-chan generateSpeakerConfig)(genQueue)
			switch {
			case len(statusQueue) > 0:
				fallthrough
			case len(genQueue) > 0:
				reloadQueueReceiver = nil
			}

			select {
			case wg := <-quit:
				defer wg.Done()
				err := client.Close()
				if err != nil {
					appLogger.Error().Err(err).Msg("failed to close voicevox client")
				}
				return

			case finalize := <-reloadQueueReceiver:
				func() {
					defer finalize()

					for {
						if _, err := v.client.Open(v.config); err != nil {
							v.logger.Error().Err(err).Msg("failed to reopen voice client, retry after 10 seconds")
							time.Sleep(10 * time.Second)
						} else {
							break
						}
					}

					v.logger.Info().Msg("successfully reopen voice client")

					loadInfo := loadInfo{
						speakerIdxNameMap: map[string]int{},
						speakerIdxIdMap:   map[int]int{},
					}

					speakers := func() []voicevox.VoiceSpeaker {
						for {
							if s, err := v.client.GetVoiceSpeakers(); err != nil {
								v.logger.Error().Err(err).Msg("failed load voice metadata or cannot parsed json file, retry after 10 seconds")
								time.Sleep(10 * time.Second)
							} else {
								return s
							}
						}
					}()

					for _, speaker := range speakers {
						for _, style := range speaker.Styles {
							name := strings.Join([]string{speaker.Name, style.Name}, "/")
							idx := len(loadInfo.speakers)

							loadInfo.speakers = append(loadInfo.speakers, VoiceSpeaker{
								Name:      name,
								Character: speaker.Name,
								Style:     style.Name,
								Id:        style.Id,
							})

							loadInfo.speakerIdxIdMap[style.Id] = idx
							loadInfo.speakerIdxNameMap[name] = idx

							// ノーマルの場合はデフォルト値を入れておく
							loadInfo.speakerIdxNameMap[speaker.Name] = idx
						}
					}

					status.info = loadInfo
				}()

			case req := <-statusQueueReceiver:
				req.receiver(status)

			case req := <-genQueueReceiver:
				if _, exist := status.info.speakerIdxIdMap[req.SpeakerId]; !exist {
					req.Receiver(nil, ErrUnknownSpeaker)
				} else {
					wav, err := client.Text2Speech(req.Text, req.SpeakerId)
					req.Receiver(wav, err)
				}

			}
		}
	}()

	return &v, nil
}

func (v *VoiceVox) Quit() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	v.quit <- &wg
	wg.Wait()
}

func (v *VoiceVox) GenerateVoice(text string, speakerId int, waitResume bool) (wav io.ReadCloser, err error) {

	if !waitResume {
		if v.restartLock.TryRLock() {
			defer v.restartLock.RUnlock()
		} else {
			return nil, ErrRestarting
		}
	}

	for v.genQueue == nil {
		time.Sleep(time.Millisecond)
	}

	wg := sync.WaitGroup{}
	req := generateSpeakerConfig{
		Text:       text,
		SpeakerId:  speakerId,
		waitResume: waitResume,
		Receiver: func(rc io.ReadCloser, genErr error) {
			wav, err = rc, genErr
			wg.Done()
		},
	}

	for {
		if v.genQueue == nil {
			time.Sleep(10 * time.Millisecond)
		} else {
			wg.Add(1)
			v.genQueue <- req
			wg.Wait()
			break
		}
	}

	if err != nil {
		return nil, err
	} else {
		return wav, nil
	}
}

func (v *VoiceVox) GetSpeakers(nameFilter string, waitResume bool) ([]VoiceSpeaker, error) {

	status, err := v.getStatus(waitResume)
	if err != nil {
		return nil, err
	}

	if nameFilter == "" {
		return status.info.speakers, nil
	} else {
		result := []VoiceSpeaker{}

		for _, speaker := range status.info.speakers {
			if strings.Contains(speaker.Name, nameFilter) {
				result = append(result, speaker)
			}
		}

		return result, nil
	}
}

func (v *VoiceVox) getStatus(waitResume bool) (s status, err error) {

	if !waitResume {
		if v.restartLock.TryRLock() {
			defer v.restartLock.RUnlock()
		} else {
			return status{}, ErrRestarting
		}
	}

	wg := sync.WaitGroup{}
	monitor := statusMonitor{
		receiver: func(status status) {
			s = status
			wg.Done()
		},
	}

	wg.Add(1)
	v.statusQueue <- monitor
	wg.Wait()
	return s, nil
}
