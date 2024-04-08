# Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
# All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http:www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: Apache-2.0

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details

FROM golang:1.21.4-alpine3.18 AS builder

ARG SSH_PRIVATE_KEY

# Prepare SSH mode for downloading git repositories/dependencies
RUN mkdir /root/.ssh/
RUN echo "${SSH_PRIVATE_KEY}" > /root/.ssh/id_rsa
RUN chmod 600 /root/.ssh/id_rsa
RUN echo "StrictHostKeyChecking no" >> /root/.ssh/config

# Force git to use SSH over HTTPS to avoid password prompt
RUN apk add git openssh make bash
RUN git config --global --add url."git@github.com:".insteadOf "https://github.com/"

RUN mkdir -p /root/go/src/github.com/kube-awi

WORKDIR /root/go/src/github.com/kube-awi
COPY . .
RUN rm bin/*

RUN make build

# Second stage: create the runtime image
FROM alpine:3.18.4
WORKDIR /
COPY --from=builder /root/go/src/github.com/kube-awi/bin/manager /manager

USER 65532:65532
ENTRYPOINT ["/manager"]
