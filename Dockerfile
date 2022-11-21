ARG voicevox_version=0.13.0
ARG onnxruntime_version=1.10.0
ARG codename=eternia

FROM golang:1.19-bullseye as builder
ENV GOARCH=amd64 \
    GOOS=linux \
    CGO_ENABLED=1

WORKDIR /opt
RUN apt-get update && \
    apt-get install -y \
    build-essential \
    apt-transport-https \
    ca-certificates \
    ffmpeg \
    gnupg \
    lsb-release \
    pkg-config \
    libogg0 \
    libopus0 \
    libopus-dev \
    opus-tools

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

ARG codename

RUN go build -o /bin/${codename} .

FROM debian:bullseye as downloader

ARG voicevox_version
ARG onnxruntime_version

WORKDIR /opt
RUN apt-get update && \
    apt-get install -y \
    unzip \
    tar \
    wget \
    apt-transport-https \
    ca-certificates && \
    wget https://github.com/VOICEVOX/voicevox_core/releases/download/${voicevox_version}/voicevox_core-linux-x64-cpu-${voicevox_version}.zip && \
    wget https://github.com/microsoft/onnxruntime/releases/download/v${onnxruntime_version}/onnxruntime-linux-x64-${onnxruntime_version}.tgz && \
    wget http://downloads.sourceforge.net/open-jtalk/open_jtalk_dic_utf_8-1.11.tar.gz && \
    unzip -j voicevox_core-linux-x64-cpu-${voicevox_version}.zip && \
    tar -zxvf onnxruntime-linux-x64-${onnxruntime_version}.tgz && \
    tar -zxvf open_jtalk_dic_utf_8-1.11.tar.gz

FROM debian:bullseye as runner
ARG workdir=/app
ENV PATH=${PATH}:${workdir}

WORKDIR ${workdir}
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    ffmpeg \
    gnupg \
    lsb-release \
    pkg-config \
    libogg0 \
    libopus0 \
    libopus-dev \
    opus-tools

ARG onnxruntime_version
ENV VOICEVOX_PATH=/lib/voicevox
ENV VOICEVOX_COREPATH=${VOICEVOX_PATH}/libcore.so
ENV VOICEVOX_JTALKDIR=/lib/open_jtalk_dic

COPY --from=downloader /opt/onnxruntime-linux-x64-${onnxruntime_version}/lib/libonnxruntime.so.${onnxruntime_version} ${VOICEVOX_PATH}/
COPY --from=downloader /opt/libcore.so ${VOICEVOX_PATH}/
COPY --from=downloader /opt/open_jtalk_dic_utf_8-1.11/ ${VOICEVOX_JTALKDIR}/

ARG codename

COPY --from=builder /bin/${codename} /bin/

ENV OMP_NUM_THREADS=2
RUN mv /bin/${codename} /bin/app

CMD [ "/bin/app" ]
