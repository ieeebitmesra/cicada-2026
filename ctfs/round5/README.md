# Round 5 — The Gatekeeper's Beacon

## Description
The High Council of Olympus communicates through encrypted interstellar beacons. 
We intercepted a signal broadcast across ethereal frequencies. 

Decrypt the beacon using known-plaintext analysis to obtain the gateway coordinates to the inner trials.

## Artifacts
- `hermes_beacon.py`

## Solution
Use the known flag prefix `PANTHEON{` (8 bytes) to XOR against the first 8 bytes of `_PAYLOAD` to recover the 8-character key `HERMES26`.
Decoding with `HERMES26` yields:
`PANTHEON{g4t3w4y_unl0ck3d_https://web-secure-portal.onrender.com}`

Access the gateway URL: https://web-secure-portal.onrender.com
