#!/bin/bash -e

export LIBP2P_ALLOW_WEAK_RSA_KEYS="" && export LIBP2P_SWARM_FD_LIMIT="10000" && ./libp2p-crawler $@
