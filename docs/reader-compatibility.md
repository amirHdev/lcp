# Reader compatibility

This project emits Readium LCP licenses and protected publications. The server side is meant to stay reader-neutral, but compatibility still needs to be proven reader by reader.

| Reader | Status | Notes |
| --- | --- | --- |
| Thorium Reader | Verified | Local end-to-end flow tested with a generated `.lcpl`, signed download URL, and passphrase unlock. |
| Readium Swift Toolkit | Planned | The server follows the Readium LCP flow, but a checked example app and written demo are still needed. |
| Readium Kotlin Toolkit / Android | Planned | The server follows the Readium LCP flow, but an Android demo and verification notes are still needed. |

## Thorium Reader

The current local flow has been checked with Thorium Reader:

1. Start the local stack with `docker compose up --build`.
2. Run `sh scripts/demo-local.sh`.
3. Import the generated `.lcpl` into Thorium Reader.
4. Use the license passphrase from the demo flow.

The local Compose setup uses `127.0.0.1` in generated public links because Thorium rejects `localhost` URLs while importing a license.

## Readium Swift

Planned work:

- add a small iOS example flow
- document how to fetch the `.lcpl`
- verify download, fulfillment, and open behavior against the current API

## Android

Planned work:

- add a small Android example flow
- document `.lcpl` download and import
- verify fulfillment and protected-book opening against the current API
