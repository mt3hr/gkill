# ========== �ݒ� ==========
$goProjectPath = "../app/"
$nodeProjectPath = "../../"
$outputFile = "../../DEPENDENCE_LICENSES.txt"

# ========== �o�͏����� ==========
"=== �ˑ����C�Z���X�ꗗ ===`n" | Out-File -Encoding utf8 $outputFile

########################################
# Go ���W���[�����C�Z���X���o
########################################
if (Test-Path "$goProjectPath\go.mod") {
    Push-Location $goProjectPath
    Write-Host "? Go�ˑ����C�Z���X���擾��..."
    go mod tidy | Out-Null
    $goDepsRaw = go list -m -json all
    Pop-Location

    Add-Content $outputFile "`n=== [Go Modules] ==="

    # JSON�u���b�N���蓮�ŕ���
    $lines = $goDepsRaw -split "`n"
    $jsonBlock = ""
    $braceCount = 0

    foreach ($line in $lines) {
        $braceCount += ($line -split "{").Count - 1
        $braceCount -= ($line -split "}").Count - 1
        $jsonBlock += "$line`n"

        if ($braceCount -eq 0 -and $jsonBlock.Trim()) {
            try {
                $mod = $jsonBlock | ConvertFrom-Json
                $jsonBlock = ""

                if ($mod.Dir -and (Test-Path $mod.Dir)) {
                    $licenseFiles = @(
                        "LICENSE", "LICENSE.txt", "LICENSE.md",
                        "UNLICENSE", "COPYING", "NOTICE"
                    ) | ForEach-Object { Join-Path $mod.Dir $_ }

                    $found = $false
                    foreach ($file in $licenseFiles) {
                        if (Test-Path $file) {
                            Add-Content $outputFile "`n=== [$($mod.Path)] ==="
                            Add-Content $outputFile "License file: $(Split-Path $file -Leaf)"
                            Add-Content $outputFile "`n$(Get-Content $file)"
                            $found = $true
                        }
                    }

                    if (-not $found) {
                        Add-Content $outputFile "`n=== [$($mod.Path)] ==="
                        Add-Content $outputFile "No license or notice file found."
                    }
                }
            } catch {
                Add-Content $outputFile "`n[ERROR] Failed to parse Go module block (manual)."
                Add-Content $outputFile $jsonBlock
                $jsonBlock = ""
            }
        }
    }
} else {
    Add-Content $outputFile "`n[!] Go�v���W�F�N�g��������܂���F$goProjectPath"
}

########################################
# Node.js ���C�Z���X���o
########################################
if (Test-Path "$nodeProjectPath\package.json") {
    Add-Content $outputFile "`n=== [Node.js Modules] ==="

    Push-Location $nodeProjectPath
    Write-Host "? Node.js�ˑ����C�Z���X���擾��..."
    $jsonRaw = license-checker --json --production 2>$null
    Pop-Location

    if ($jsonRaw) {
        $packages = $jsonRaw | ConvertFrom-Json
        foreach ($pkg in $packages.PSObject.Properties) {
            $name = $pkg.Name
            $info = $pkg.Value

            Add-Content $outputFile "`n=== [$name] ==="
            Add-Content $outputFile "License: $($info.licenses)"
            if ($info.repository) {
                Add-Content $outputFile "Repository: $($info.repository)"
            }

            $licenseText = $info.licenseText
            $licenseFile = $info.licenseFile

            if ($licenseText) {
                Add-Content $outputFile "`n$licenseText"
            } elseif ($licenseFile -and (Test-Path $licenseFile)) {
                Add-Content $outputFile "`n$(Get-Content $licenseFile)"
            } else {
                Add-Content $outputFile "`n(���C�Z���X�{����������܂���ł���)"
            }

            # NOTICE���T��
            if ($info.path -and (Test-Path "$($info.path)\NOTICE")) {
                Add-Content $outputFile "`n[NOTICE] (from $($info.path)\NOTICE):"
                Add-Content $outputFile "$(Get-Content "$($info.path)\NOTICE" )"
            }
        }
    } else {
        Add-Content $outputFile "`n[!] license-checker ���� JSON �o�͂ł��܂���ł���"
    }
} else {
    Add-Content $outputFile "`n[!] Node.js �v���W�F�N�g��������܂���F$nodeProjectPath"
}

########################################
# ? ����
########################################
Write-Host "`n? �������C�Z���X���� $outputFile �ɏo�͂��܂����B" -ForegroundColor Green
