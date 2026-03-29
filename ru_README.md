# Awesomer

English version: [README.md](./README.md)
Документация для разработчиков: [docs/developer.ru.md](./docs/developer.ru.md)

`Awesomer` теперь разделён на два Linux-инструмента:

- `awesomerd`: root-only демон, который постоянно следит за процессами и автоматически помещает "тяжёлые" процессы в ограниченную группу ресурсов;
- `awesomerctl`: TUI-клиент, который запускается по требованию, локально читает процессы и общается с демоном через Unix socket, если тот доступен.

## Содержание

1. [Возможности](#возможности)
2. [Стек](#стек)
3. [Системные требования](#системные-требования)
4. [Структура проекта](#структура-проекта)
5. [Установка и запуск](#установка-и-запуск)
6. [Конфигурация](#конфигурация)
7. [Как работает интерфейс](#как-работает-интерфейс)
8. [Как работает демон](#как-работает-демон)
9. [Логи](#логи)
10. [Ограничения](#ограничения)

## Возможности

- Обновление списка процессов в реальном времени с настраиваемой частотой.
- Сортировка по PID, имени, CPU, памяти, числу потоков и пользователю.
- Просмотр краткой карточки процесса с командой запуска, владельцем и `nice`.
- Просмотр расширенной информации:
  - сетевые соединения,
  - открытые файлы,
  - дерево дочерних процессов.
- Управление процессами прямо из TUI:
  - остановка,
  - продолжение,
  - завершение,
  - завершение всего дерева процессов,
  - изменение CPU affinity,
  - изменение `RLIMIT_NOFILE`,
  - ручное помещение в `processJail` и извлечение из него.
- Фоновый демон для контроля ресурсов и помещения процессов в ограниченную группу ресурсов через `systemd` unit или `cgroup v2`.
- Раздельные логи для основного приложения и демона.

## Стек

- Go `1.25`
- Bubble Tea для терминального интерфейса
- Bubbles (`table`, `viewport`) для виджетов
- Lip Gloss для компоновки и оформления
- `gopsutil` для чтения информации о процессах
- `yaml.v3` для загрузки `config.yaml`
- `systemd` units или Linux `cgroup v2` для ограничения ресурсов

## Системные требования

Проект ориентирован на Linux.

Минимально требуется:

- Linux с доступным `/proc`
- Linux с `cgroup v2`, смонтированным в `/sys/fs/cgroup`
- `systemd`, если ограничения должны создаваться через unit-ы
- Go `1.25` или новее

Практически важно учитывать следующее:

- Для отправки сигналов чужим процессам нужны соответствующие права.
- Для создания и настройки ограничений через `systemd` или `cgroup v2` нужны повышенные привилегии.
- Без нужных прав просмотр части информации может работать, а управление процессами и работа демона с ограничениями ресурсов - нет.

## Структура проекта

```text
cmd/awesomerctl/main.go      Точка входа клиента `awesomerctl`
cmd/awesomerd/main.go        Точка входа демона `awesomerd`
internal/config              Общая конфигурация приложения
internal/daemon              Демон мониторинга, IPC и hot-reload конфигурации
internal/daemon/config       Модель конфигурации демона
internal/daemon/info         Общий API состояния "процесс в jail"
internal/service             Бизнес-логика TUI, сортировки и действия
internal/service/tui         Bubble Tea интерфейс
pkg/parser                   Парсинг процессов через gopsutil
pkg/cgroups                  Выбор backend-а ограничений: systemd unit или cgroup v2
pkg/logger                   Логгеры приложения и демона
pkg/mutation                 Низкоуровневые helpers для изменения процессов
```

## Установка и запуск

### Сборка

```bash
git clone https://github.com/igikawa/awesomer.git
cd awesomer
go build -o awesomerctl ./cmd/awesomerctl
go build -o awesomerd ./cmd/awesomerd
```

### Ручная Установка

```bash
sudo install -m 0755 awesomerctl /usr/bin/awesomerctl
sudo install -m 0755 awesomerd /usr/bin/awesomerd
sudo install -d -m 0755 /etc/awesomer
sudo install -m 0644 internal/config/config.yaml.example /etc/awesomer/config.yaml
sudo install -m 0644 deploy/systemd/awesomerd.service /etc/systemd/system/awesomerd.service
sudo systemctl daemon-reload
sudo systemctl enable --now awesomerd.service
```

### Быстрый старт

```bash
mkdir -p ~/.config/awesomer
cp internal/config/config.yaml.example ~/.config/awesomer/config.yaml
./awesomerctl
```

`awesomerctl` сам не поднимает демон.
Если существует `/run/awesomer.sock`, клиент подключается к `awesomerd`; иначе работает как offline monitor.

### Запуск без предварительной сборки

```bash
mkdir -p ~/.config/awesomer
cp internal/config/config.yaml.example ~/.config/awesomer/config.yaml
go run ./cmd/awesomerctl
sudo go run ./cmd/awesomerd
```

### Что происходит при старте

- `awesomerctl` читает `~/.config/awesomer/config.yaml` для обычного пользователя и `/etc/awesomer/config.yaml` для root.
- `awesomerd` - отдельный root-only процесс. Он читает `/etc/awesomer/config.yaml`, создаёт `/run/awesomer.sock` и перечитывает daemon-настройки на лету.
- `awesomerctl` никогда не владеет демоном и только подключается к нему, если socket доступен.

## Конфигурация

Пример `config.yaml`:

```yaml
tick: 1

logger:
  log_path: ./awesome.log
  daemon_log_path: ./awesome.daemon.log

daemon:
  run: true
  tick: 5
  cpu_limit: 85
  ram_limit: 60
  cpu_quota: 20
  ram_quota: 8G
  whitelist:
    - systemd
    - sshd

ui:
  table_width: 0
  info_width: 72
  border_color: "102"
  active_border_color: "62"
  selection_text_color: "229"
  selection_background_color: "57"
```

### Общие параметры

- `tick` - частота обновления списка процессов в секундах.
  - `0` отключает автообновление.
  - По умолчанию: `1`

### Логирование

- `logger.log_path` - путь к основному логу приложения.
  - По умолчанию: `./awesome.log`
- `logger.daemon_log_path` - путь к логу демона.
  - По умолчанию: `./awesome.daemon.log`

### Параметры демона

- `daemon.run` - должен ли `awesomerd` реально применять ограничения после старта.
  - По умолчанию: `true`
- `daemon.tick` - период опроса процессов демоном в секундах.
  - По умолчанию: `3`
- `daemon.cpu_limit` - порог CPU, после которого процесс получает замечание.
  - По умолчанию: `85`
- `daemon.ram_limit` - порог использования RAM в процентах, после которого процесс получает замечание.
  - По умолчанию: `60`
- `daemon.cpu_quota` - лимит CPU для ограничивающей группы ресурсов.
  - Для `systemd` преобразуется в `CPUQuota`
  - Для прямого `cgroup v2` записывается в `cpu.max`
  - По умолчанию: `20`
- `daemon.ram_quota` - лимит памяти для ограничивающей группы ресурсов.
  - Для `systemd` преобразуется в `MemoryMax`
  - Для прямого `cgroup v2` записывается в `memory.max`
  - По умолчанию: `8G`
- `daemon.whitelist` - имена процессов, которые демон не должен помещать в ограниченную группу ни при каких условиях.
  - Сопоставление нечувствительно к регистру и идёт по имени процесса, которое видно в таблице.
  - Подходит для системных и критичных сервисов, которые администратор хочет исключить из автоматического ограничения.
  - По умолчанию: `["systemd", "sshd"]`

`awesomerd` перечитывает `config.yaml` на лету, поэтому изменения конфигурации применяются без его перезапуска.

### systemd Service

Пример unit-файла лежит в `deploy/systemd/awesomerd.service`.

Типовая установка:

```bash
sudo cp deploy/systemd/awesomerd.service /etc/systemd/system/awesomerd.service
sudo systemctl daemon-reload
sudo systemctl enable --now awesomerd.service
```

Когда service уже запущен, `awesomerctl` подключается к daemon через `/run/awesomer.sock`.

Полезные операции:

```bash
systemctl status awesomerd.service
sudo systemctl restart awesomerd.service
journalctl -u awesomerd.service -f
awesomerctl
sudo awesomerctl
```

### Debian Пакет

В репозитории есть простой builder:

```bash
chmod +x packaging/deb/build.sh
./packaging/deb/build.sh 0.1.0
```

Он создаёт `dist/awesomer_0.1.0_<arch>.deb`.

Установка и запуск:

```bash
sudo dpkg -i dist/awesomer_0.1.0_$(dpkg --print-architecture).deb
sudo systemctl enable --now awesomerd.service
```

### Параметры интерфейса

- `ui.table_width` - предпочтительная ширина панели таблицы в символах.
  - `0` означает автоматический расчёт от общей ширины терминала.
  - По умолчанию: `0`
- `ui.info_width` - предпочтительная ширина панели деталей в символах.
  - Используется, когда `ui.table_width` не задан явно.
  - По умолчанию: `72`
- `ui.border_color` - цвет рамки неактивных панелей.
  - По умолчанию: `"102"`
- `ui.active_border_color` - цвет рамки панели в фокусе.
  - По умолчанию: `"62"`
- `ui.selection_text_color` - цвет текста выделенной строки таблицы.
  - По умолчанию: `"229"`
- `ui.selection_background_color` - цвет фона выделенной строки таблицы.
  - По умолчанию: `"57"`

## Как работает интерфейс

Интерфейс состоит из двух панелей:

- слева - таблица процессов;
- справа - панель деталей.

При старте справа выводится встроенная справка. Список процессов обновляется по таймеру `tick`, если `tick != 0`.

### Колонки таблицы

- `PID`
- `Name`
- `CPU`
- `Mem`
- `Threads`
- `User`
- `*` - процесс находится под контролем демона и уже помещён в ограниченную группу

### Навигация

- `↑` / `↓` - перемещение по списку процессов
- `Tab` - переключение фокуса между таблицей и панелью деталей
- `Esc` - вернуть фокус на таблицу
- `q` или `Ctrl+C` - корректное завершение приложения

### Сортировка

- `p` - по PID
- `n` - по имени
- `c` - по CPU
- `m` - по памяти
- `t` - по числу потоков
- `u` - по пользователю

Активная колонка сортировки отображается жирным заголовком.

### Действия над процессом

- `Enter` - показать краткую информацию по выбранному процессу
- `h` - показать расширенную информацию
- `a` - задать CPU affinity для выбранного процесса
- `l` - изменить `RLIMIT_NOFILE` для выбранного процесса
- `j` - вручную переместить выбранный процесс и его дерево в `processJail` или вернуть обратно
- `s` - отправить `SIGSTOP`
- `r` - отправить `SIGCONT`
- `k` - отправить `SIGKILL`
- `d` - завершить всё дерево процессов, корнем которого является выбранный PID

### Что показывается справа

Режим `Enter`:

- PID
- имя процесса
- пользователь
- `nice`
- CPU_AFFINITY
- RLIMIT_NOFILE
- полная команда запуска

Режим `h`:

- сетевые соединения
- открытые файлы
- дерево дочерних процессов

Режим `a`:

- ввод списка CPU-ядер в формате `0,1,3`
- применение через `SchedSetaffinity`

Режим `l`:

- ввод нового значения `RLIMIT_NOFILE`
- применение через `prlimit`

## Как работает демон

`awesomerd` — это отдельный процесс или `systemd` service. `awesomerctl` только подключается к нему.

Алгоритм работы:

1. Демон определяет backend ограничений.
2. Если в системе активен `systemd`, создаётся transient unit `processJail.service`; иначе создаётся прямой `cgroup` `processJail`.
3. Для группы задаются ограничения CPU и памяти.
4. Демон периодически сканирует все процессы.
5. Некоторые процессы пропускаются: PID `< 100`, а также `systemd` и `sshd`.
6. Если процесс превышает `daemon.cpu_limit` или `daemon.ram_limit`, он получает замечание.
7. После 3 замечаний процесс и его дочерние процессы переносятся в `processJail`.
8. В таблице такие процессы помечаются символом `*`.

Помимо автоматического режима, процесс можно вручную отправить в `processJail` клавишей `j`. Для ручного перевода используются те же значения `daemon.cpu_quota` и `daemon.ram_quota`.

Когда `awesomerd` завершается или `daemon.run` переключается в `false`, демон сначала возвращает jailed-процессы в root group, а затем удаляет созданный `systemd` unit или `cgroup`.

## Логи

По умолчанию проект пишет в два файла:

- `./awesome.log` - события основного приложения и TUI
- `./awesome.daemon.log` - события демона

Оба логгера используют стандартный `log.Logger` Go с датой, временем и короткой позицией в файле.

## Ограничения

- Проект работает только на Linux.
- Код зависит от `/proc`, POSIX-сигналов и механизма ограничения ресурсов Linux (`systemd` или `cgroup v2`).
- Для части процессов расширенная информация (`connections`, `open files`) может быть недоступна из-за прав.
- Демону нужны права на работу с `systemd` или доступ на запись в `/sys/fs/cgroup`.
