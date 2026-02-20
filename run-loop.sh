#!/bin/bash

cd ~/.picoclaw/workspace
while true; do
    picoclaw gateway
    echo "***** Sleep before rerunning - use Ctrl+C to stop *****"
    sleep 5
done
