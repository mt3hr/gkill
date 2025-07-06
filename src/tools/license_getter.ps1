# ==========
# 設定
# ==========
$goProjectPath = "../app/"
$nodeProjectPath = "../../"
$outputFile = "../../DEPENDENCE_LICENSES.txt"

# ==========
# 出力初期化
# ==========
"=== 依存ライセンス一覧 ===`n" | Out-File -Encoding utf8 $outputFile

########################################
# ? Go モジュールライセンス抽出（vendor使わない版）
########################################
if (Test-Path "$goProjectPath\go.mod") {
    Push-Location $goProjectPath
    Write-Host "? Go依存ライセンスを取得中..."

    go mod tidy | Out-Null
    $goDepsRaw = go list -m -json all

    Pop-Location

    Add-Content $outputFile "`n=== [Go Modules] ==="

    $lines = $goDepsRaw -split "`r?`n"
    $buffer = ""
    $braceCount = 0

    foreach ($line in $lines) {
        $trimmed = $line.Trim()
        $braceCount += ($trimmed -split "{").Count - 1
        $braceCount -= ($trimmed -split "}").Count - 1
        $buffer += "$line`n"

        if ($braceCount -eq 0 -and $buffer.Trim() -ne "") {
            try {
                $mod = $buffer | ConvertFrom-Json
                $buffer = ""

                if ($mod.Dir -and (Test-Path $mod.Dir)) {
                    $licenseFiles = @(
                        "LICENSE", "LICENSE.txt", "LICENSE.md",
                        "UNLICENSE", "COPYING", "NOTICE"
                    )

                    $found = $false
                    foreach ($fileName in $licenseFiles) {
                        $filePath = Join-Path $mod.Dir $fileName
                        if (Test-Path $filePath) {
                            Add-Content $outputFile "`n=== [$($mod.Path)] ==="
                            Add-Content $outputFile "License file: $fileName"
                            Add-Content $outputFile "`n$(Get-Content $filePath -TotalCount 10)"
                            $found = $true
                            break
                        }
                    }

                    if (-not $found) {
                        Add-Content $outputFile "`n=== [$($mod.Path)] ==="
                        Add-Content $outputFile "No license file found."
                    }
                }
            } catch {
                Add-Content $outputFile "`n[ERROR] Failed to parse Go module block."
                $buffer = ""
            }
        }
    }
} else {
    Add-Content $outputFile "`n[!] Goプロジェクトが見つかりません：$goProjectPath"
}

########################################
# ? Node.js ライセンス抽出（前回のまま）
########################################
if (Test-Path "$nodeProjectPath\package.json") {
    Add-Content $outputFile "`n=== [Node.js Modules] ==="

    Push-Location $nodeProjectPath
    Write-Host "? Node.js依存ライセンスを取得中..."
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
                Add-Content $outputFile "`n$(Get-Content $licenseFile -TotalCount 10)"
            } else {
                Add-Content $outputFile "`n(ライセンス本文が見つかりませんでした)"
            }
        }
    } else {
        Add-Content $outputFile "`n[!] license-checker から JSON 出力できませんでした"
    }
} else {
    Add-Content $outputFile "`n[!] Node.js プロジェクトが見つかりません：$nodeProjectPath"
}

########################################
# ? 完了
########################################
Write-Host "`n? 統合ライセンス情報を $outputFile に出力しました。" -ForegroundColor Green
