package main

import "embed"

// webDist — собранный фронтенд (web/dist), встроенный в бинарник. Путь
// относителен корню репозитория (пакет main), т.к. //go:embed не выходит за
// пределы каталога пакета.
//
//go:embed all:web/dist
var webDist embed.FS
