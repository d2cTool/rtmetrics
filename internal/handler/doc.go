// Package handler собирает HTTP-хендлеры сервера метрик.
//
// Эндпоинты практического трека:
//
//	POST /update                  — одна метрика, JSON
//	POST /update/{mtype}/{name}/{value}
//	POST /updates/                — пачка метрик, JSON
//	GET  /value/{mtype}/{name}    — значение текстом
//	POST /value                   — значение JSON
//	GET  /                        — HTML-дашборд
//	GET  /ping                    — проверка БД
package handler
