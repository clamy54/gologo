# Compile GoLogo pour Windows, avec icone et metadonnees embarquees dans l'exe.
# Sortie : tools/build/bin/windows/gologo.exe
# Prerequis : Go 1.25+, un compilateur C (ex. MSYS2/mingw-w64), CGO active.
$ErrorActionPreference = "Stop"

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$src  = (Resolve-Path (Join-Path $here "..\..\src")).Path
$icon = Join-Path $here "icons\gologo.ico"
$bin  = Join-Path $here "bin\windows"
New-Item -ItemType Directory -Force -Path $bin | Out-Null

# Genere la ressource Windows (.syso) : icone + table de version.
$syso = Join-Path $src "cmd\gologo\resource_windows.syso"
# -64 : genere un .syso amd64 (sinon il est 32 bits et le linker l'ignore
# silencieusement sur un build amd64 -> ni icone ni metadonnees embarquees).
& go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 `
    -64 -icon "$icon" -o "$syso" (Join-Path $here "versioninfo.json")
if ($LASTEXITCODE -ne 0) { throw "goversioninfo a echoue" }

# Build (application GUI : -H=windowsgui supprime la console).
Push-Location $src
try {
    $env:CGO_ENABLED = "1"
    & go build -trimpath -ldflags "-H=windowsgui -s -w" -o (Join-Path $bin "gologo.exe") ./cmd/gologo
    if ($LASTEXITCODE -ne 0) { throw "go build a echoue" }
}
finally {
    Remove-Item $syso -ErrorAction SilentlyContinue
    Pop-Location
}
Write-Host "OK -> $bin\gologo.exe"
