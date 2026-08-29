// Package gnome содержит встроенные файлы GNOME Shell-расширения, чтобы
// команда `tasky upgrade` могла обновить их на месте (если расширение уже
// установлено), не обращаясь к сети. На Windows эти файлы не используются,
// но безвредно встроены в бинарник.
package gnome

import "embed"

// Files — встроенные файлы расширения (extension.js + metadata.json).
//
//go:embed extension.js metadata.json
var Files embed.FS
