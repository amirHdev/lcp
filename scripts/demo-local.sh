#!/bin/sh
set -eu

base_url="${BASE_URL:-http://localhost:8080}"
book_path="${BOOK_PATH:-examples/pride-and-prejudice/pride-and-prejudice.epub}"

token="$(
  curl -fsS "$base_url/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin","twoFactor":"123456"}' |
    jq -r .token
)"

book_b64="$(base64 < "$book_path" | tr -d '\n')"

publication_id="$(
  curl -fsS "$base_url/graphql" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$(jq -cn --arg file "$book_b64" '{
      query: "mutation UploadPublication($title: String!, $file: Upload!) { uploadPublication(title: $title, file: $file) { id } }",
      variables: {
        title: "Pride and Prejudice",
        file: $file
      }
    }')" |
    jq -r .data.uploadPublication.id
)"

license_id="$(
  curl -fsS "$base_url/graphql" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "$(jq -cn --arg publicationID "$publication_id" '{
      query: "mutation CreateLicense($publicationID: ID!, $userID: ID!, $passphrase: String!, $hint: String!) { createLicense(publicationID: $publicationID, userID: $userID, passphrase: $passphrase, hint: $hint) { id } }",
      variables: {
        publicationID: $publicationID,
        userID: "reader-01",
        passphrase: "open-sesame",
        hint: "demo"
      }
    }')" |
    jq -r .data.createLicense.id
)"

echo "publication_id=$publication_id"
echo "license_id=$license_id"
echo "license_url=$base_url/licenses/$license_id.lcpl"
