#pragma once

#if __cplusplus
#include <cstdint>
#else
#include <stdbool.h>
#include <stdint.h>
#endif

bool initialize(void *ptr, bool useGpu, int numThreads, bool loadAllModels) {
  bool (*fn)(bool, int, bool) = ptr;
  return fn(useGpu, numThreads, loadAllModels);
}

bool loadModel(void *ptr, int64_t speakerId) {
  bool (*fn)(int64_t) = ptr;
  return fn(speakerId);
}

bool checkModelLoaded(void *ptr, int64_t speakerId) {
  bool (*fn)(int64_t) = ptr;
  return fn(speakerId);
}

void finalize(void *ptr) {
  void (*fn)() = ptr;
  fn();
}

const char *getMeta(void *ptr) {
  const char *(*fn)() = ptr;
  return fn();
}

const char *getErrMsg(void *ptr) {
  const char *(*fn)() = ptr;
  return fn();
}

int loadDict(void *ptr, const char *dictPath) {
  const int (*fn)(const char *) = ptr;
  return fn(dictPath);
}

uint8_t *text2Speech(void *ptr, const char *text, int64_t speakerId,
                     int *outputSize, int *result) {
  const int (*fn)(const char *, int64_t, int *, uint8_t **) = ptr;
  uint8_t *outputWav;
  (*result) = fn(text, speakerId, outputSize, &outputWav);
  return outputWav;
}

void freeWave(void *ptr, uint8_t *wav) {
  void (*fn)(uint8_t *) = ptr;
  return fn(wav);
}

int bool2int(bool input) { return input ? 1 : 0; }
