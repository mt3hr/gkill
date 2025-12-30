@echo off
pushd "%~dp0"

echo 【 gkillサンプルデータ 】
echo 1.下記のアドレスにアクセス
echo    http://localhost:8888/
echo.

echo 2.以下アカウントでログイン
echo    ユーザID: gkill_sample_data
echo    パスワード: sample
echo.

echo 3.この黒窓をクリックして CTRL+C で終了
echo.

start "" "http://localhost:8888/"
gkill_server --gkill_home_dir .

popd
