
echo "\nGETTING MODULES\n"

go mod vendor -v

echo "\nBUILDING IN DOCKER\n"

docker run --rm \
  -v "$PWD:/usr/src/${PWD##*/}" \
  -v "$PWD/.cache:/root/.cache/go-build" \
  -w "/usr/src/${PWD##*/}" \
  golang:1.15 \
  ./build.sh

echo "\nDONE\n"