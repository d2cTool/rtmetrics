# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Конфигурация

Примеры JSON без секретов лежат в `configs/server.example.json` и `configs/agent.example.json`. Скопируйте их в рабочие файлы и заполните DSN, ключи и токены локально:

```
cp configs/server.example.json configs/server.json
cp configs/agent.example.json configs/agent.json
```

```
./server -c configs/server.json
./agent -c configs/agent.json
```

Реальные `config.json`, `server.json`, `agent.json`, `configs/server.json` и `configs/agent.json` в git не попадают — см. `.gitignore`.

## Генерация RSA-ключей

Асимметричное шифрование тел запросов (`-crypto-key` / `CRYPTO_KEY`): серверу нужен приватный ключ, агенту — публичный.

```
make gen-keys
```

Эквивалент без Make:

```
mkdir -p keys
openssl genrsa -out keys/private.pem 4096
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

Дальше:

```
./server -crypto-key keys/private.pem
./agent -crypto-key keys/public.pem
```

Каталог `keys/*.pem` игнорируется git. Не коммитьте приватный ключ.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки

Горячие пути покрыты бенчмарками:

| Пакет | Что измеряем |
|---|---|
| `internal/agent` | сбор runtime-метрик, сборка батча |
| `internal/storage` | запись/чтение in-memory хранилища |
| `internal/hash` | HMAC-SHA256 подпись и проверка |
| `internal/handler/update` | JSON-хендлер `/update` |
| `internal/handler/html` | HTML-дашборд |

```
go test -run '^$' -bench . -benchmem ./internal/agent ./internal/storage ./internal/hash ./internal/handler/update ./internal/handler/html
```

После оптимизации `BuildBatch` (указатели на поля структуры вместо `NewGauge` на каждую метрику):

```
BenchmarkBuildBatch-8    1845003    637.9 ns/op    2280 B/op    3 allocs/op
```

Было: `2067 ns/op`, `6376 B/op`, `31 allocs/op`.

Дополнительно убраны лишние буферы в HTML- и JSON-хендлерах и промежуточный слайс в `hash.Sign`.

## Профиль памяти

Профили — heap живого сервера **во время нагрузки**, не микробенчмарк. Вход — реалистичный батч агента (29 метрик: runtime gauges + `PollCount`) в `profiles/testdata/batch.json`.

Нагрузка: [hey](https://github.com/rakyll/hey), параллельно `POST /updates/` и `GET /` (дашборд уже непустой после warmup). Снимок `GET /debug/pprof/heap` через 2 с после старта hey, пока запросы ещё идут.

```
go install github.com/rakyll/hey@latest
powershell -File profiles/capture.ps1 -OutFile profiles/base.pprof
# оптимизация
powershell -File profiles/capture.ps1 -OutFile profiles/result.pprof
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

`top` / `list` / `peek` по `base.pprof`: удерживаемая память на горячем пути — `slog` (`Logger.Info`, `Logger.With`, `buffer.Write`) и `chi` middleware. На каждый запрос писался access-лог уровня Info и создавался `log.With`.

Что убрано под эту нагрузку:

- access-лог и успешные «data saved» / «html rendered» переведены на Debug; при уровне Info `With` не вызывается;
- `gzip.Writer` и карта compressible-типов больше не создаются на каждый ответ;
- слайс имён для аудита собирается, только если аудит включён.

Вывод `go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`:

```
File: server.exe
Type: inuse_space
Showing nodes accounting for -561.54kB, 17.95% of 3128.59kB total
      flat  flat%   sum%        cum   cum%
-1056.33kB 33.76% 33.76% -1056.33kB 33.76%  log/slog/internal/buffer.(*Buffer).Write
 -532.26kB 17.01% 50.78%  -532.26kB 17.01%  log/slog/internal/buffer.(*Buffer).WriteString
    -514kB 16.43% 50.71%     -514kB 16.43%  bufio.NewReaderSize
         0     0% 17.95% -1588.59kB 50.78%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 17.95% -1056.33kB 33.76%  log/slog.(*Logger).Info
         0     0% 17.95%  -532.26kB 17.01%  log/slog.(*Logger).With
```

Отрицательные значения — меньше удерживаемой памяти на том же сценарии. Под нагрузкой latency `POST /updates/` упала с ~430 ms до ~2 ms (hey перестал упираться в синхронный лог в консоль).
