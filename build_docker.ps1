
echo "`nGETTING MODULES`n"

go mod vendor -v

echo "`nBUILDING IN DOCKER`n"

docker run --rm `
  -v "$((Get-Location).Path):/usr/src/$((Get-Item .).BaseName)" `
  -v "$((Get-Location).Path)/.cache:/root/.cache/go-build" `
  -w "/usr/src/$((Get-Item .).BaseName)" `
  golang:1.15 `
  ./build.sh

echo "`nDONE`n"