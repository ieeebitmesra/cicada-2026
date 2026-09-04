#!/bin/bash
set -e

USERNAME="$1"
if [ -z "$USERNAME" ]; then
    read -p "Enter your Docker Hub username / organization: " USERNAME
fi

if [ -z "$USERNAME" ]; then
    echo "Error: Docker Hub username cannot be empty."
    exit 1
fi

echo "========================================="
echo " Building and Pushing CTF Challenge Images"
echo " Target Namespace: $USERNAME"
echo "========================================="

CHALLENGES=(
    "pwn-maze-oob"
    "pwn-stonks-fmt"
    "crypto-sphinx-oracle"
    "web-global-megaphone"
    "web-secure-portal"
)

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for c in "${CHALLENGES[@]}"; do
    IMAGE_TAG="${USERNAME}/${c}:latest"
    echo -e "\n>>> Building ${c} -> ${IMAGE_TAG}..."
    docker build -t "${IMAGE_TAG}" -f "${DIR}/${c}/Dockerfile" "${DIR}/${c}"

    echo -e ">>> Pushing ${IMAGE_TAG}..."
    docker push "${IMAGE_TAG}"
done

echo -e "\n========================================="
echo " All 5 CTF images successfully pushed!"
echo "========================================="
