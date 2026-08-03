# このファイルについて

`embed.go` の `//go:embed embed` は、対象ディレクトリに1件もファイルが無いと
`pattern embed: no matching files found` でコンパイルエラーになる。

このディレクトリの中身（`html/` `i18n/` `manual/` `version.json`）はすべて
`npm run prepare_install` が生成するもので `.gitignore` の対象なので、
クリーンな clone では空になり `go build ./...` すら通らなくなる。
それを避けるために、このファイルだけを追跡対象にしている
（`.gitignore` に `!/src/server/gkill/api/embed/PLACEHOLDER.md` の除外を書いている）。

`npm run clean_app_embed` は `html` / `i18n` / `manual` / `version.json` を
個別に消すだけなので、このファイルは残る。

配信経路は `fs.Sub(api.EmbedFS, "embed/html")` と `"embed/manual"` を通るため、
このファイルがHTTPで見えることはない。

なお `api` パッケージの `init()` は `embed/i18n/locales` を読んで
見つからなければ panic する。つまりこのファイルがあっても
`go test` を通すには locales の配置が要る。CI では
`npm run copy_i18n_to_app_embed` を先に実行している。
