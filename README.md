# Практическая работа 3: надёжная доставка поверх UDP

Проект разделён на модули `protocol` (пакеты), `transport` (UDP), `telemetry` (метрики работы №2) и `reliability` (ACK, повторы, дедупликация и RTO).

## Быстрый запуск

Откройте два терминала в корне репозитория.

Сервер:

```powershell
go run -buildvcs=false ./cmd/reliable-server -mode baseline
```

Клиент:

```powershell
go run -buildvcs=false ./cmd/reliable-client -experiment baseline
```

Режимы: `baseline`, `loss_5`, `loss_10`, `loss_20`, `delay_100_loss_5`, `jitter_loss_10`. Для каждой новой серии перезапустите сервер с нужным `-mode`, а клиент запустите с таким же `-experiment` и флагом `-append` (кроме первой серии).

Отчёт и графики: `go run -buildvcs=false ./cmd/reliability-report`.

Тесты: `go test -buildvcs=false ./...`.

Команды работы №2 сохранены в `cmd/server`, `cmd/client` и `cmd/report`.
