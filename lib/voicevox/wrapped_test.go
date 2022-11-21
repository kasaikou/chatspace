package voicevox

import (
	"io"
	"log"
	"os"
	"testing"
)

func TestWrapper(t *testing.T) {

	client, err := LoadLib(
		"/workspace/lib/voicevox/libcore.so",
		"/workspace/lib/voicevox/open_jtalk_dic_utf_8-1.11",
	)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("library loaded")

	if _, err := client.Open(InitConfig{
		UseGPU:        false,
		NumThreads:    1,
		LoadAllModels: true,
	}); err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	log.Println(client.GetVoiceSpeakers())

	log.Println("voice loaded")

	wav, err := client.Text2Speech("ハローニューワールド．", 1)
	if err != nil {
		t.Fatal(err)
	}
	wav.Close()

	log.Println("voice generated")

	file, err := os.Create("new-world.wav")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if _, err := io.Copy(file, wav); err != nil {
		t.Fatal(err)
	}
	log.Println("succeed")
}
