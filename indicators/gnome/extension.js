// Индикатор Tasky для GNOME Shell: время за сегодня и запущенная подзадача.
// Данные — через HTTP GET /status сервера `tasky serve` (по умолчанию
// http://127.0.0.1:9110, оверрайд — TASKY_HTTP_ADDR). Индикатор всегда видим;
// если сервер не запущен — показывает «—» и предлагает запустить. Клик
// открывает меню с деталями. Опрос — раз в секунду. Совместимо с GNOME
// Shell 45+ (ESM).

import Clutter from 'gi://Clutter';
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import Pango from 'gi://Pango';
import St from 'gi://St';

import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import { Button as PanelMenuButton } from 'resource:///org/gnome/shell/ui/panelMenu.js';
import * as PopupMenu from 'resource:///org/gnome/shell/ui/popupMenu.js';
import { Extension } from 'resource:///org/gnome/shell/extensions/extension.js';

const POLL_SECONDS = 1;
const DEFAULT_HTTP_ADDR = '127.0.0.1:9110';
const TITLE_MAX = 16; // символов названия подзадачи в панели
const MENU_MAX_WIDTH = '380px';

// GNOME Shell 45+ требует default-экспорт класса: создаёт экземпляр
// `new extensionModule.default({...})` и вызывает enable()/disable().
export default class TaskyIndicatorExtension extends Extension {
    enable() {
        this._indicator = new TaskyIndicator();
    }

    disable() {
        if (this._indicator) {
            this._indicator.destroy();
            this._indicator = null;
        }
    }
}

// fmtDur — как fmtDur в TUI Tasky: «1ч 05м» / «45м 12с» / «10с».
function fmtDur(seconds) {
    seconds = Math.max(0, Math.round(seconds));
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    if (h > 0)
        return `${h}ч ${m}м`;
    if (m > 0)
        return `${m}м ${s}с`;
    return `${s}с`;
}

// truncate обрезает текст до max кодовых точек (корректно для эмодзи и
// суррогатных пар) и добавляет «…», если текст был обрезан.
function truncate(text, max) {
    const chars = Array.from(text);
    if (chars.length <= max)
        return text;
    return chars.slice(0, max - 1).join('') + '…';
}

// httpAddr — адрес HTTP-сервера Tasky: TASKY_HTTP_ADDR или дефолт.
// Читается при каждом опросе/запуске, чтобы подхватить изменение env без
// перезахода (если переменная задана в ~/.config/environment.d/ или через
// `systemctl --user import-environment`). `export` в терминале до
// gnome-shell не дойдёт — это ограничение окружения процесса gnome-shell.
function httpAddr() {
    return GLib.getenv('TASKY_HTTP_ADDR') || DEFAULT_HTTP_ADDR;
}

// socketPath — путь сокета сервера: TASKY_SOCKET или TASKY_HOME/tasky.sock.
// null — дефолт (<XDG_DATA_HOME>/tasky/tasky.sock), передавать --socket не нужно.
function socketPath() {
    const s = GLib.getenv('TASKY_SOCKET');
    if (s)
        return s;
    const home = GLib.getenv('TASKY_HOME');
    if (home)
        return GLib.build_filenamev([home, 'tasky.sock']);
    return null;
}

// taskyBin — путь к бинарю tasky. Поддерживает все штатные места установки:
// `just install` (~/.local/bin), `PREFIX=/usr/local just install`,
// `go install` (~/go/bin), GOPATH/bin, /usr/bin.
// Сначала пробует PATH, затем явные фолбэки — GNOME Shell стартует с
// урезанным PATH и `~/.local/bin` там часто отсутствует.
function taskyBin() {
    const found = GLib.find_program_in_path('tasky');
    if (found)
        return found;
    const candidates = [];
    const home = GLib.get_home_dir();
    candidates.push(GLib.build_filenamev([home, '.local', 'bin', 'tasky']));
    candidates.push('/usr/local/bin/tasky');
    candidates.push(GLib.build_filenamev([home, 'go', 'bin', 'tasky']));
    const gopath = GLib.getenv('GOPATH');
    if (gopath)
        candidates.push(GLib.build_filenamev([gopath, 'bin', 'tasky']));
    candidates.push('/usr/bin/tasky');
    for (const p of candidates) {
        if (GLib.file_test(p, GLib.FileTest.EXISTS))
            return p;
    }
    return null;
}

class TaskyIndicator {
    constructor() {
        this._polling = false;
        this._server = null; // запущенный Gio.Subprocess сервера (null, если не запущен)
        this._serverReachable = false; // последний успешный GET /status
        this._lastRunning = null;
        this._lastToday = '—';

        this._button = new PanelMenuButton(0.0, 'Tasky', false);
        this._label = new St.Label({
            text: '',
            y_align: Clutter.ActorAlign.CENTER,
        });
        this._button.add_child(this._label);
        this._button.add_child(PopupMenu.arrowIcon(St.Side.RIGHT));

        Main.panel.addToStatusArea('tasky-indicator', this._button);
        this._button.hide(); // до первого успешного опроса

        this._timeout = GLib.timeout_add_seconds(
            GLib.PRIORITY_DEFAULT, POLL_SECONDS, () => {
                this._poll();
                return GLib.SOURCE_CONTINUE;
            });
        this._poll();
    }

    // _httpGET делает простой GET по HTTP/1.0 и отдаёт cb(JSON или null).
    // Соединение и таймаут — через Gio.SocketClient; один read достаточно
    // для маленького тела /status.
    _httpGET(path, cb) {
        const addr = httpAddr();
        const i = addr.lastIndexOf(':');
        if (i < 0) {
            cb(null);
            return;
        }
        const host = addr.slice(0, i);
        const port = parseInt(addr.slice(i + 1), 10);
        if (!Number.isInteger(port) || port <= 0 || port > 65535) {
            cb(null);
            return;
        }

        const client = new Gio.SocketClient();
        client.set_timeout(2_000_000); // 2 с (GLib.TimeSpan — микросекунды)
        const socketAddr = Gio.InetSocketAddress.new_from_string(host, port);
        client.connect_async(socketAddr, null, (c, res) => {
            let conn = null;
            try {
                conn = c.connect_finish(res);
                const bytes = new GLib.Bytes(
                    new TextEncoder().encode(`GET ${path} HTTP/1.0\r\n\r\n`));
                conn.get_output_stream().write_bytes(bytes, null);
                conn.get_input_stream().read_bytes_async(
                    65536, GLib.PRIORITY_DEFAULT, null, (s, r) => {
                        let data = null;
                        try {
                            const b = s.read_bytes_finish(r);
                            const text = b
                                ? new TextDecoder().decode(b.get_data())
                                : '';
                            const body = text.includes('\r\n\r\n')
                                ? text.split('\r\n\r\n')[1]
                                : text;
                            data = JSON.parse(body.trim());
                        } catch (e) {
                            // пустой/битый ответ — сервер недоступен
                            data = null;
                        }
                        conn.close(null);
                        cb(data);
                    });
            } catch (e) {
                if (conn)
                    conn.close(null);
                cb(null);
            }
        });
    }

    _poll() {
        if (this._polling)
            return; // прошлый запрос ещё в полёте
        this._polling = true;
        this._httpGET('/status', (data) => {
            this._polling = false;
            if (!data) {
                try {
                    this._update(null);
                } catch (e) {
                    console.log(`tasky-indicator: ${e}`);
                }
                return;
            }
            try {
                this._update(data);
            } catch (e) {
                // ошибка отрисовки не должна прятать индикатор
                console.log(`tasky-indicator: ${e}`);
            }
        });
    }

    _update(data) {
        // Индикатор всегда видим, чтобы из меню можно было запустить сервер.
        this._serverReachable = !!data;
        let today = '—';
        let running = null;
        if (data && typeof data.today_seconds === 'number') {
            today = fmtDur(data.today_seconds);
            if (data.subtask && typeof data.subtask.title === 'string')
                running = data.subtask.title;
        }
        this._lastRunning = running;
        this._lastToday = today;

        const parts = [];
        if (running)
            parts.push(truncate(running, TITLE_MAX));
        parts.push(today);
        this._label.set_text(parts.join(' · '));

        this._rebuildMenu(running, today);
        this._button.show();
    }

    // _controlItem добавляет кликабельный пункт меню с обработчиком.
    _controlItem(label, onActivate, sensitive) {
        const item = new PopupMenu.PopupMenuItem(label, {});
        item.setSensitive(sensitive);
        item.connect('activate', () => {
            try {
                onActivate();
            } catch (e) {
                console.log(`tasky-indicator: ${e}`);
            }
        });
        return item;
    }

    _rebuildMenu(running, today) {
        const menu = this._button.menu;
        const reachable = this._serverReachable;
        const owned = this._server !== null;
        // Запустить — только когда сервер недоступен (ни внешний, ни свой) и мы
        // сами не в процессе запуска. Остановить — только свой процесс.
        const canStart = !reachable && !owned;
        const canStop = reachable && owned;

        const items = [];
        if (running) {
            try {
                const item = new PopupMenu.PopupBaseMenuItem({reactive: false});
                const label = new St.Label({
                    text: `Сейчас: ${running}`,
                    x_expand: true,
                    style: `max-width: ${MENU_MAX_WIDTH}`,
                });
                label.clutter_text.ellipsize = Pango.EllipsizeMode.END;
                item.add_child(label);
                items.push(item);
            } catch (e) {
                // пункт «Сейчас» опционален — не должен ломать меню
                console.log(`tasky-indicator: ${e}`);
            }
        }

        const total = new PopupMenu.PopupBaseMenuItem({reactive: false});
        total.add_child(new St.Label({
            text: `За сегодня: ${today}`,
            x_expand: true,
        }));
        items.push(total);

        // Управление сервером.
        items.push(this._controlItem(
            'Запустить сервер', () => this._startServer(), canStart));
        items.push(this._controlItem(
            'Остановить сервер', () => this._stopServer(), canStop));
        items.push(this._controlItem(
            'Открыть в браузере', () => this._openBrowser(), true));

        menu.removeAll();
        for (const item of items)
            menu.addMenuItem(item);
    }

    // _startServer запускает `tasky serve` отвязанным процессом.
    _startServer() {
        if (this._server !== null)
            return;
        if (this._serverReachable)
            return; // уже есть внешний сервер — не дублируем
        const bin = taskyBin();
        if (!bin) {
            console.log('tasky-indicator: tasky не найден (checked PATH, ~/.local/bin/tasky, /usr/local/bin/tasky, ~/go/bin/tasky)');
            return;
        }
        const argv = [bin, 'serve'];
        const addr = httpAddr();
        const rawAddr = GLib.getenv('TASKY_HTTP_ADDR');
        if (rawAddr || addr !== DEFAULT_HTTP_ADDR)
            argv.push('--http-addr', addr);
        const sp = socketPath();
        if (sp)
            argv.push('--socket', sp);
        try {
            this._server = Gio.Subprocess.new(
                argv,
                Gio.SubprocessFlags.STDOUT_SILENCE | Gio.SubprocessFlags.STDERR_SILENCE);
        } catch (e) {
            console.log(`tasky-indicator: не удалось запустить ${argv.join(' ')}: ${e}`);
            this._server = null;
            return;
        }
        // Следим за выходом своего процесса — если упал сразу (порт занят и т.п.),
        // сбросим _server и обновим меню на следующем _poll.
        try {
            this._server.wait_async(null, (proc, res) => {
                try {
                    proc.wait_finish(res);
                } catch (_e) {
                    // процесс уже завершён
                }
                if (this._server === proc) {
                    this._server = null;
                    // меню обновится на следующем _poll; форсируем
                    this._poll();
                }
            });
        } catch (_e) {
            // wait_async недоступен — деградация: состояние сбросится по HTTP-failure
        }
        // Не затираем today/running — ждём первого успешного _poll (≤1с).
        this._rebuildMenu(this._lastRunning, this._lastToday);
    }

    // _stopServer останавливает только свой сервер (SIGTERM). Внешний не трогаем.
    _stopServer() {
        if (this._server === null)
            return;
        try {
            this._server.send_signal(15); // SIGTERM
        } catch (e) {
            console.log(`tasky-indicator: ${e}`);
        }
        this._server = null;
        // Меню обновится когда HTTP начнёт отвечать null; покажем текущий reachable
        // но с canStop=false.
        this._rebuildMenu(this._lastRunning, this._lastToday);
        // Форсируем опрос, чтобы быстрее обнаружить остановку.
        this._poll();
    }

    // _openBrowser открывает веб-интерфейс в браузере по умолчанию.
    _openBrowser() {
        const uri = `http://${httpAddr()}`;
        Gio.AppInfo.launch_default_for_uri(uri, null);
    }

    destroy() {
        if (this._server !== null) {
            try {
                this._server.send_signal(15);
            } catch (e) {
                // сервер уже завершился
            }
            this._server = null;
        }
        if (this._timeout)
            GLib.source_remove(this._timeout);
        this._button.destroy();
        this._button = null;
        this._label = null;
    }
}