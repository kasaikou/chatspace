package voicevox

import (
	/*
		#cgo CPPFLAGS: -I.
		#cgo LDFLAGS: -ldl
		#include <dlfcn.h>
		#include "dllcore.h"
	*/
	"C"
)
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unsafe"
)

type Client struct {
	library          unsafe.Pointer
	initialize       unsafe.Pointer
	finalize         unsafe.Pointer
	loadModel        unsafe.Pointer
	checkModelLoaded unsafe.Pointer
	getMeta          unsafe.Pointer
	getErrMsg        unsafe.Pointer
	loadDict         unsafe.Pointer
	text2Speech      unsafe.Pointer
	freeWave         unsafe.Pointer
}

type InitConfig struct {
	UseGPU        bool
	NumThreads    int
	LoadAllModels bool
}

type waveData struct {
	rawPtr unsafe.Pointer
	client *Client
	bytes.Reader
}

var ErrLibraryNotLoaded = errors.New("library have not been loaded yet")

func LoadLib(libPath, dictPath string) (*Client, error) {

	// ライブラリ
	library := C.dlopen(C.CString(libPath), C.RTLD_LAZY)
	if library == nil {
		msg := C.GoString(C.dlerror())
		return nil, fmt.Errorf("cannot load library: %s: %s", libPath, msg)
	}

	// 関数ポインタ取得用の関数
	solveSymbol := func(symbolName string) (unsafe.Pointer, error) {
		symbol := C.dlsym(library, C.CString(symbolName))
		if symbol == nil {
			msg := C.GoString(C.dlerror())
			return nil, fmt.Errorf("cannot found symbol: %s: %s", symbolName, msg)
		}
		return symbol, nil
	}

	if initialize, err := solveSymbol("initialize"); err != nil {
		return nil, err
	} else if loadModel, err := solveSymbol("load_model"); err != nil {
		return nil, err
	} else if checkModelLoaded, err := solveSymbol("is_model_loaded"); err != nil {
		return nil, err
	} else if finalize, err := solveSymbol("finalize"); err != nil {
		return nil, err
	} else if getMeta, err := solveSymbol("metas"); err != nil {
		return nil, err
	} else if getErrMsg, err := solveSymbol("last_error_message"); err != nil {
		return nil, err
	} else if loadDict, err := solveSymbol("voicevox_load_openjtalk_dict"); err != nil {
		return nil, err
	} else if text2Speech, err := solveSymbol("voicevox_tts"); err != nil {
		return nil, err
	} else if freeWave, err := solveSymbol("voicevox_wav_free"); err != nil {
		return nil, err
	} else {
		client := &Client{
			initialize:       initialize,
			finalize:         finalize,
			loadModel:        loadModel,
			checkModelLoaded: checkModelLoaded,
			getMeta:          getMeta,
			getErrMsg:        getErrMsg,
			loadDict:         loadDict,
			text2Speech:      text2Speech,
			freeWave:         freeWave,
		}

		return client, client.loadDictionary(dictPath)
	}
}

func (c *Client) getError() error {
	result := C.getErrMsg(c.getErrMsg)
	return fmt.Errorf("voicevox error: %s", C.GoString(result))
}

func (c *Client) loadDictionary(dictPath string) error {
	result := C.loadDict(c.loadDict, C.CString(dictPath))
	if result == 1 {
		return c.getError()
	}
	return nil
}

func (c *Client) Open(config InitConfig) (*Client, error) {
	result := C.initialize(
		c.initialize,
		C.bool(config.UseGPU),
		C.int(config.NumThreads),
		C.bool(config.LoadAllModels),
	)
	if int(C.bool2int(result)) == 0 {
		return nil, c.getError()
	}

	return c, nil
}

func (c *Client) Close() error {
	C.finalize(c.finalize)
	return nil
}

func (c *Client) GetVoiceSpeakers() ([]VoiceSpeaker, error) {
	speakers := []VoiceSpeaker{}
	err := json.Unmarshal([]byte(C.GoString(C.getMeta(c.getMeta))), &speakers)
	return speakers, err
}

func (c *Client) Text2Speech(text string, speakerId int) (wav io.ReadCloser, err error) {

	var result C.int
	var size C.int
	rawWav := C.text2Speech(c.text2Speech, C.CString(text), C.int64_t(speakerId), &size, &result)
	wave := C.GoBytes(unsafe.Pointer(rawWav), size)

	if result == 1 {
		return nil, c.getError()
	}

	return &waveData{
		Reader: *bytes.NewReader(wave),
		rawPtr: unsafe.Pointer(rawWav),
		client: c,
	}, nil
}

func (w *waveData) Close() error {
	C.freeWave(w.client.freeWave, (*C.uint8_t)(w.rawPtr))
	return nil
}
