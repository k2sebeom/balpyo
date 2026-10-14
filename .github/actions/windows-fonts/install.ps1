# 내장 · 예제 테마의 Noto 글꼴을 윈도우에 깐다 — CI의 윈도우 잡과 로컬 개발자가 같은 파일을 쓴다.
#
# 리눅스 잡이 까는 패키지(fonts-noto-core · fonts-noto-mono · fonts-noto-cjk)와 같은 가족 ·
# 같은 굵기다. 측정 브라우저에 테마 글꼴이 없으면 잰 빌드가 글꼴 경고를 내고, 예제 게이트는
# 경고 0이어야 하므로 두 호스트가 같은 글꼴을 가져야 같은 게이트다. 윈도우에는 패키지가 없으므로
# Noto 저장소에서 받는다. 커밋을 고정하므로 받는 바이트가 바뀌지 않고, 그래서 캐시해도 된다.
#
# 라틴 가족은 noto-fonts의 v20201206-phase3 태그에서 받는다 — 데비안 · 우분투 패키지와 같은
# 세대다. 그 뒤의 커밋은 Display를 "Noto Sans"의 하위 가족(이름 표의 typographic family)으로
# 옮겨서, 가족을 그 이름으로 묶는 DirectWrite에는 "Noto Sans Display"라는 가족이 없어진다.
#
# 설치는 글꼴 폴더에 복사하고 레지스트리에 적는 것이다 — 그 뒤에 뜨는 브라우저(DirectWrite)가
# 그 글꼴을 본다. Machine은 시스템 폴더와 HKLM(관리자 권한, CI 러너), User는 사용자 폴더와
# HKCU다(권한이 필요 없고 그 사용자에게만 보인다 — 개발 머신의 기본값).
#
# 윈도우 PowerShell 5.1과 PowerShell 7 둘 다에서 돈다. 5.1은 BOM이 없는 스크립트를 ANSI로
# 읽으므로 이 파일은 UTF-8 BOM으로 저장한다.
param(
  [ValidateSet('User', 'Machine')] [string] $Scope = 'User',
  [string] $CacheDir = (Join-Path ([IO.Path]::GetTempPath()) 'balpyo-noto')
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$latin = 'https://raw.githubusercontent.com/notofonts/noto-fonts/20bc5918912503bc1537a407a694738c33c048aa/hinted/ttf'
$cjk = 'https://raw.githubusercontent.com/notofonts/noto-cjk/f8d157532fbfaeda587e826d4cd5b21a49186f7c'
$files = @()
foreach ($family in 'NotoSans', 'NotoSerif', 'NotoSansDisplay') {
  foreach ($style in 'Regular', 'Bold', 'Italic', 'BoldItalic') { $files += "$latin/$family/$family-$style.ttf" }
}
foreach ($style in 'Regular', 'Bold') { $files += "$latin/NotoSansMono/NotoSansMono-$style.ttf" }
foreach ($style in 'Regular', 'Bold') {
  $files += "$cjk/Sans/OTC/NotoSansCJK-$style.ttc"
  $files += "$cjk/Serif/OTC/NotoSerifCJK-$style.ttc"
}

if ($Scope -eq 'Machine') {
  $fontDir = Join-Path $env:WINDIR 'Fonts'
  $registry = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts'
} else {
  $fontDir = Join-Path $env:LOCALAPPDATA 'Microsoft\Windows\Fonts'
  $registry = 'HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Fonts'
}
New-Item -ItemType Directory -Force $CacheDir, $fontDir | Out-Null
if (-not (Test-Path $registry)) { New-Item -Path $registry -Force | Out-Null }

foreach ($url in $files) {
  $name = Split-Path $url -Leaf
  $file = Join-Path $CacheDir $name
  if (-not (Test-Path $file)) { Invoke-WebRequest -Uri $url -OutFile $file }
  $dest = Join-Path $fontDir $name
  # 설치된 글꼴 파일은 글꼴 캐시 서비스가 잡고 있어 덮어쓰지 못한다 — 같은 바이트면 그대로 둔다.
  $same = (Test-Path $dest) -and ((Get-FileHash $dest).Hash -eq (Get-FileHash $file).Hash)
  if (-not $same) { Copy-Item $file $dest -Force }
  # 시스템 폴더의 글꼴은 파일 이름으로, 사용자 폴더의 글꼴은 전체 경로로 등록한다.
  $value = if ($Scope -eq 'Machine') { $name } else { $dest }
  New-ItemProperty -Path $registry -Name "$([IO.Path]::GetFileNameWithoutExtension($name)) (TrueType)" -Value $value -PropertyType String -Force | Out-Null
}
Write-Output "installed $($files.Count) font files into $fontDir ($Scope)"
