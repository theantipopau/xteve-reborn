<#
.SYNOPSIS
  Builds xteve-reborn release binaries for all 5 supported platforms and
  zips each one, ready to attach to a GitHub release.

.DESCRIPTION
  Bakes the given version into the binary via -ldflags "-X main.ReleaseTag=...",
  which is what lets a running binary know its own version well enough to
  check GitHub for a newer one (see xteve.go's ReleaseTag var and
  src/update.go). Forgetting this flag doesn't break the build - it just
  silently disables update checking for that release, which is exactly the
  kind of mistake this script exists to prevent.

.PARAMETER Version
  The release tag, with or without a leading "v" (e.g. "3.0.0-pre.3" or
  "v3.0.0-pre.3"). Used both as the ldflags value and in each output
  filename, matching this repo's asset-naming convention
  (xteve-reborn_<version>_<os>_<arch>.zip) that GetLatestRelease
  (src/internal/up2date/client/client.go) expects when checking for updates.

.EXAMPLE
  .\tools\release\build-release.ps1 -Version 3.0.0-pre.3
#>
param(
  [Parameter(Mandatory = $true)]
  [string]$Version
)

$ErrorActionPreference = "Stop"

# Normalize: strip a leading "v" if given, since it's added back below only
# where GitHub's tag format wants it (the ldflag) - the .zip/binary names
# themselves use the bare version, matching what release binaries have
# always been named.
$Version = $Version.TrimStart("v")
$Tag = "v$Version"

$RepoRoot = Resolve-Path "$PSScriptRoot\..\.."
$OutDir = Join-Path $RepoRoot ".devdata\release"

if (Test-Path $OutDir) {
  Remove-Item -Recurse -Force $OutDir
}
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$Targets = @(
  @{os = "windows"; arch = "amd64"; ext = ".exe" },
  @{os = "linux"; arch = "amd64"; ext = "" },
  @{os = "linux"; arch = "arm64"; ext = "" },
  @{os = "darwin"; arch = "amd64"; ext = "" },
  @{os = "darwin"; arch = "arm64"; ext = "" }
)

Push-Location $RepoRoot
try {

  $env:CGO_ENABLED = "0"

  foreach ($t in $Targets) {

    $env:GOOS = $t.os
    $env:GOARCH = $t.arch

    $dirName = "xteve-reborn_${Version}_$($t.os)_$($t.arch)"
    $targetDir = Join-Path $OutDir $dirName
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null

    $outFile = Join-Path $targetDir "xteve-reborn$($t.ext)"

    Write-Host "Building $($t.os)/$($t.arch)..."
    go build -ldflags "-X main.ReleaseTag=$Tag" -o $outFile .

    if ($LASTEXITCODE -ne 0) {
      throw "Build failed for $($t.os)/$($t.arch)"
    }

    $zipPath = Join-Path $OutDir "$dirName.zip"
    Compress-Archive -Path "$targetDir\*" -DestinationPath $zipPath -Force
    Write-Host "  -> $zipPath"

  }

}
finally {
  Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
  Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
  Pop-Location
}

Write-Host ""
Write-Host "Done. Zips are in $OutDir"
Write-Host "Next: git tag -a $Tag -m `"xteve-reborn $Tag`" && git push origin $Tag"
Write-Host "Then: gh release create $Tag $OutDir\*.zip --title `"xteve-reborn $Tag`" --notes-file <notes.md> --prerelease"
