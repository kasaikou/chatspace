FROM debian:bullseye

LABEL maintainer="Kasai Kou"

ARG apt_get_server=ftp.jaist.ac.jp/pub/Linux
ARG golang_version=1.19
ARG voicevox_version=0.13.0
ARG onnxruntime_version=1.10.0
ARG username=vscode
ARG useruid=1000
ARG usergid=${useruid}

ENV DEBIAN_FRONTEND=nointeractive
ENV LANG=C.UTF-8

ENV GOPATH=/home/${username}/go
ENV PATH=${PATH}:/usr/local/go/bin
ENV PATH=${PATH}:${GOPATH}/bin
ENV SHELL=/bin/bash
ENV VOICEVOX_PATH=/home/${username}/voicevox
ENV VOICEVOX_COREPATH=${VOICEVOX_PATH}/libcore.so
ENV VOICEVOX_JTALKDIR=${VOICEVOX_PATH}/open_jtalk_dic_utf_8

WORKDIR /opt
RUN \
    sed -i s@archive.ubuntu.com@${apt_get_server}@g /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    apt-transport-https \
    build-essential \
    curl \
    ca-certificates \
    lsb-release \
    pkg-config \
    gnupg \
    git \
    zip \
    unzip \
    gnupg \
    lsb-release \
    ffmpeg \
    libogg0 \
    libopus0 \
    opus-tools \
    wget \
    sudo && \
    #
    # Install docker
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    apt-get install -y docker-ce docker-ce-cli containerd.io && \
    #
    #
    wget https://go.dev/dl/go${golang_version}.linux-$(dpkg --print-architecture).tar.gz && \
    tar -C /usr/local -xzf go${golang_version}.linux-$(dpkg --print-architecture).tar.gz && \
    #
    # Create non-root user: https://aka.ms/vscode-remote/containers/non-root-user
    groupadd --gid ${usergid} ${username} && \
    useradd --uid ${useruid} --gid ${usergid} -m ${username} && \
    echo ${username} ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/${username} && \
    chmod 0440 /etc/sudoers.d/${username} && \
    chown -R ${username} /opt

USER ${username}
RUN go install golang.org/x/tools/gopls@latest && \
    go install golang.org/x/lint/golint@latest && \
    go install github.com/go-delve/delve/cmd/dlv@master && \
    go install github.com/haya14busa/goplay/cmd/goplay@v1.0.0 && \
    go install github.com/fatih/gomodifytags@v1.16.0 && \
    go install github.com/josharian/impl@latest && \
    go install github.com/cweill/gotests/gotests@latest && \
    go install github.com/ramya-rao-a/go-outline@latest && \
    go install golang.org/x/tools/cmd/godoc@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    #
    # Install Voicevox
    wget https://github.com/VOICEVOX/voicevox_core/releases/download/${voicevox_version}/voicevox_core-linux-x64-cpu-${voicevox_version}.zip && \
    wget https://github.com/microsoft/onnxruntime/releases/download/v${onnxruntime_version}/onnxruntime-linux-x64-${onnxruntime_version}.tgz && \
    wget http://downloads.sourceforge.net/open-jtalk/open_jtalk_dic_utf_8-1.11.tar.gz && \
    unzip voicevox_core-linux-x64-cpu-${voicevox_version}.zip && \
    tar -zxvf onnxruntime-linux-x64-${onnxruntime_version}.tgz && \
    tar -zxvf open_jtalk_dic_utf_8-1.11.tar.gz && \
    mkdir -p ${VOICEVOX_PATH} && \
    cp \
    onnxruntime-linux-x64-${onnxruntime_version}/lib/libonnxruntime.so.${onnxruntime_version} \
    voicevox_core-linux-x64-cpu-${voicevox_version}/libcore.so \
    ${VOICEVOX_PATH} && \
    cp -r open_jtalk_dic_utf_8-1.11 ${VOICEVOX_JTALKDIR}