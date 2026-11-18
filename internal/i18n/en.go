package i18n

// enMap — англійський словник. Ключі — українські рядки.
var enMap = map[string]string{
	// ── App / window ──────────────────────────────────────────────────────
	"Ziper VPN - Захищений VPN клієнт":        "Ziper VPN - Secure VPN Client",
	"Ziper VPN — підключено":                  "Ziper VPN — connected",
	"Ziper VPN — підключення виконується...":  "Ziper VPN — connecting...",
	"Ziper VPN — відключено":                  "Ziper VPN — disconnected",

	// ── Settings ──────────────────────────────────────────────────────────
	"Налаштування Ziper VPN": "Ziper VPN Settings",
	"Конфігурація автозапуску, безпеки та мережі": "Startup, security and network configuration",
	"Автозапуск та автоматизація": "Startup & Automation",
	"Автозапуск з Windows":        "Start with Windows",
	"Запускати Ziper VPN автоматично при старті операційної системи (Windows Registry)":
		"Start Ziper VPN automatically when Windows starts (Windows Registry)",
	"Автопідключення VPN": "Auto-connect VPN",
	"Автоматично підключатися до найшвидшого VPN сервера при відкритті програми":
		"Automatically connect to the fastest VPN server when the app starts",
	"Мережевий захист та протоколи": "Network Protection & Protocols",
	"Захист від витоку DNS (DNS Leak Protection)": "DNS Leak Protection",
	"Блокувати небезпечні зовнішні DNS запити повз VPN тунель (block-outside-dns)":
		"Block unsafe DNS requests outside the VPN tunnel (block-outside-dns)",
	"Аварійний вимикач (Kill Switch)": "Kill Switch",
	"Захист від раптового розкриття реального IP при аварійному обриві зв'язку":
		"Prevent real IP leaks if the connection drops unexpectedly",
	"Протокол OpenVPN:": "OpenVPN protocol:",
	"DNS Резолвер:":     "DNS resolver:",
	"Сповіщення":        "Notifications",
	"Спливаючі сповіщення (Notifications)": "Pop-up notifications",
	"Показувати повідомлення про успішне підключення, відключення та помилки":
		"Show notifications about successful connections, disconnections and errors",
	"Скинути за замовчуванням": "Reset to defaults",
	"Закрити":                  "Close",
	"Зберегти налаштування":    "Save settings",
	"Мова інтерфейсу":          "Interface language",
	"Оберіть мову інтерфейсу. Зміна мови перезавантажить вікно.":
		"Choose the interface language. Changing it will reload the window.",
	"Свій DNS (custom)": "Custom DNS",
	"Власні DNS адреси (напр. 1.1.1.1, 8.8.8.8)": "Custom DNS addresses (e.g. 1.1.1.1, 8.8.8.8)",
	"Некоректна IP-адреса (IP через кому або пробіл)": "Invalid IP address (separate IPs with commas or spaces)",
	"Активний DNS провайдер": "Active DNS provider",
	"Всі налаштування успішно збережено!": "All settings saved successfully!  ",
	"Налаштування скинуто до типових":       "Settings reset to defaults",
	"Збережено налаштування: {key} = {value}": "Setting saved: {key} = {value}",

	// ── Header / admin banner ─────────────────────────────────────────────
	"Процес має повні права адміністратора для Wintun": "The process has full administrator rights for Wintun",
	"Потрібні права адміністратора для створення Wintun адаптера":
		"Administrator rights are required to create the Wintun adapter",
	"Next-Gen Privacy & Anti-Censorship": "Next-Gen Privacy & Anti-Censorship",
	"Оновити зовнішній IP та геолокацію": "Refresh public IP and geolocation",
	"Налаштування програми та автозапуску": "App and startup settings",
	"Потрібні права адміністратора Windows": "Windows administrator rights required",
	"Wintun-драйвер та маршрутизація OpenVPN вимагають прав адміністратора. Натисніть для надання прав.":
		"The Wintun driver and OpenVPN routing require administrator rights. Click to grant them.",
	"Запустити від імені адміністратора": "Run as administrator",

	// ── Main status panel ─────────────────────────────────────────────────
	"ПУБЛІЧНА IP-АДРЕСА": "PUBLIC IP ADDRESS",
	"Визначення...":      "Detecting...",
	"Завантаження локації...": "Loading location...",
	"НЕ ЗАХИЩЕНО": "UNPROTECTED",
	"ЗАХИЩЕНО":    "PROTECTED",
	"ПІДКЛЮЧЕННЯ": "CONNECTING",
	"Натисніть для увімкнення безпечного шифрування": "Click to enable secure encryption",
	"Захищений тунель активний • Трафік зашифровано": "Secure tunnel active • Traffic encrypted",
	"Встановлення безпечного тунелю OpenVPN...": "Establishing a secure OpenVPN tunnel...",
	"З'ЄДНАННЯ...": "CONNECTING...",
	"Вузол: Швидкий авто-підбір (VPN Gate)": "Node: Fast auto-select (VPN Gate)",
	"Вузол: оберіть сервер зі списку нижче": "Node: choose a server from the list below",
	"Вузол: Власний профіль ({name})":       "Node: Custom profile ({name})",
	"Вузол: {value}":                        "Node: {value}",
	"Вузол:":                                "Node:",
	"завантажте .ovpn":                      "upload an .ovpn",
	"Вузол: {flag} {country} ({ip})":        "Node: {flag} {country} ({ip})",

	// ── Metrics ───────────────────────────────────────────────────────────
	"ПРОТОКОЛ":           "PROTOCOL",
	"ЗАТРИМКА (PING)":    "LATENCY (PING)",
	"DNS ЗАХИСТ":         "DNS PROTECTION",
	"TOR SOCKS5":         "TOR SOCKS5",
	"Вимкнено":           "Disabled",
	"АКТИВНИЙ":           "ACTIVE",

	// ── Tabs ──────────────────────────────────────────────────────────────
	"Сервери VPN":   "VPN Servers",
	"Власний .ovpn": "Custom .ovpn",
	"Tor Мережа":    "Tor Network",
	"DNS Безпека":   "DNS Security",
	"Логи тунелю":   "Tunnel Logs",

	// ── Server browser ────────────────────────────────────────────────────
	"РЕЖИМ ПІДКЛЮЧЕННЯ": "CONNECTION MODE",
	"Завантаження...":   "Loading...",
	"Оновити сервери з VPN Gate": "Refresh servers from VPN Gate",
	"⚡ Авто":    "⚡ Auto",
	"🌐 Мій сервер": "🌐 My server",
	"📂 Мій .ovpn":  "📂 My .ovpn",
	"Авто: обереться найкращий сервер; при відмові перебирає всі сервери по колу, поки не підключить":
		"Auto: picks the best server; if it fails, tries every server in turn until connected",
	"Мій сервер: підключення тільки до обраного вами сервера зі списку":
		"My server: connects only to the server you selected from the list",
	"Мій .ovpn: тільки ваш власний конфіг (імпортований, збережений або вбудований)":
		"My .ovpn: only your own config (imported, saved or built-in)",
	"Пошук за країною, IP або хостом (напр.: япон, сша, 219.100)...":
		"Search by country, IP or host (e.g. japan, usa, 219.100)...",
	"Застосувати пошук": "Apply search",
	"Всі країни":        "All countries",
	"Країна":            "Country",
	"За рейтингом":      "By rating",
	"За пінгом":         "By ping",
	"За швидкістю":      "By speed",
	"Сортування":        "Sorting",
	"Завантаження списку серверів від VPN Gate...": "Loading server list from VPN Gate...",
	"Серверів за обраними критеріями не знайдено":  "No servers match the selected filters",
	"Скинути фільтри": "Reset filters",
	"СКАСУВАТИ":       "CANCEL",
	"Вибрати":         "Select",
	"Натисніть, щоб відмінити вибір сервера та повернутися до авто-підбору":
		"Click to deselect the server and return to auto-select",
	"Знайдено: {count} серверів": "Found: {count} servers",
	"Помилка завантаження":        "Loading error",
	"Не вдалося завантажити список серверів": "Failed to load the server list",
	"Перевірте підключення до інтернету та оновіть список":
		"Check your internet connection and refresh the list",
	"Спробувати ще раз": "Try again",

	// ── Custom .ovpn tab ──────────────────────────────────────────────────
	"Імпорт власної OpenVPN конфігурації": "Import a custom OpenVPN configuration",
	"Ви можете завантажити власний .ovpn профіль — він буде збережений і доступний після перезапуску.":
		"You can upload your own .ovpn profile — it will be saved and available after restart.",
	"Перетягніть або виберіть .ovpn файл": "Drag and drop or choose an .ovpn file",
	"Мої збережені конфіги": "My saved configs",
	"Завантажені конфіги автоматично зберігаються в папці config/profiles. Видалення — через кнопку «Видалити» з підтвердженням.":
		"Uploaded configs are saved automatically in config/profiles. Use the Delete button with confirmation to remove one.",
	"Поки немає збережених конфігів — завантажте свій .ovpn файл вище.":
		"No saved configs yet — upload your .ovpn file above.",
	"Збережений власний профіль · config/profiles/{name}": "Saved custom profile · config/profiles/{name}",
	"Видалити конфіг?": "Delete config?",
	"Файл config/profiles/{name} буде видалено незворотно.":
		"The file config/profiles/{name} will be permanently deleted.",
	"Скасувати": "Cancel",
	"Видалити":  "Delete",

	// ── Tor tab ───────────────────────────────────────────────────────────
	"Tor SOCKS5 Проксі": "Tor SOCKS5 Proxy",
	"Локальна адреса: 127.0.0.1:{port}": "Local address: 127.0.0.1:{port}",
	"ЗУПИНЕНО": "STOPPED",
	"Запустити Tor": "Start Tor",
	"Зупинити Tor":  "Stop Tor",
	"SOCKS5 порт":   "SOCKS5 port",
	"Живі логи Tor": "Live Tor logs",
	"Копіювати":     "Copy",
	"Очистити":      "Clear",
	"Очікування запуску служби Tor...": "Waiting for the Tor service to start...",
	"Підказка: Після запуску служби Tor налаштуйте SOCKS5 проксі в Telegram, Firefox або Chrome на адресу, вказану вище, для анонімної маршрутизації всього веб-трафіку.":
		"Tip: After starting Tor, configure the SOCKS5 proxy in Telegram, Firefox or Chrome to the address shown above to route all web traffic anonymously.",
	"Запуск...": "Starting...",
	"АКТИВНИЙ (127.0.0.1:{port})": "ACTIVE (127.0.0.1:{port})",
	"ПОМИЛКА": "ERROR",
	"Службу Tor зупинено": "Tor service stopped",
	"Tor SOCKS5 успішно активовано на порту {port}!": "Tor SOCKS5 activated successfully on port {port}!",
	"Не вдалося запустити Tor": "Failed to start Tor",
	"Порт має бути у діапазоні 1024–65535 — повернено 9050":
		"Port must be between 1024 and 65535 — reset to 9050",
	"Порт змінено на {port} — зупиніть і запустіть Tor знову":
		"Port changed to {port} — stop and start Tor again",

	// ── DNS tab ───────────────────────────────────────────────────────────
	"Захист DNS та фільтрація загроз": "DNS Protection & Threat Filtering",
	"Захистіть свої інтернет-запити від відстеження інтернет-провайдером.":
		"Protect your internet requests from ISP tracking.",
	"Блокувати рекламу, трекери та небезпечні сайти": "Block ads, trackers and malicious sites",
	"Використовує DNS-фільтрацію на рівні тунелю для прискорення завантаження сторінок":
		"Uses tunnel-level DNS filtering to speed up page loading",
	"Перевірити поточний IP та DNS": "Check current IP and DNS",

	// ── Logs ──────────────────────────────────────────────────────────────
	"Системний термінал подій": "System Event Terminal",
	"Очікування ініціалізації тунелю...": "Waiting for tunnel initialization...",
	"Логи скопійовано в буфер обміну!  ": "Logs copied to clipboard!  ",
	"Консоль логів порожня":              "The log console is empty",
	"Логи очищено":                       "Logs cleared",
	"Логи Tor скопійовано в буфер обміну!  ": "Tor logs copied to clipboard!  ",
	"Консоль логів Tor порожня":              "The Tor log console is empty",
	"Логи Tor очищено":                       "Tor logs cleared",

	// ── Notifications / status messages ───────────────────────────────────
	"VPN успішно відключено": "VPN disconnected securely",
	"VPN успішно відключено.": "VPN disconnected successfully.",
	"Підключення відмінено":   "Connection cancelled",
	"Підключення відмінено.":  "Connection cancelled.",
	"Вичерпано всі сервери зі списку.": "All servers in the list have been exhausted.",
	"Список серверів порожній.":        "The server list is empty.",
	"⚡ Автопідключення VPN активовано...": "⚡ VPN auto-connect activated...",
	"VPN успішно підключено!":             "VPN connected successfully!",
	"Підключення вже запускається, зачекайте кілька секунд...":
		"Connection is already starting, please wait a few seconds...",
	"Підготовка конфігурації та вибір сервера...": "Preparing configuration and selecting a server...",
	"Помилка: Немає доступних серверів.": "Error: No servers available.",
	"Помилка зв'язку з VPN Gate. Перевірте з'єднання з мережею.":
		"VPN Gate connection error. Check your network connection.",
	"Сервер не обрано — підключаюсь через авто-підбір": "No server selected — connecting via auto-select",
	"Помилка: Не вдалося підключитися до {count} серверів.": "Error: Failed to connect to {count} servers.",
	"Усі {count} серверів недоступні. Спробуйте ручний вибір.":
		"All {count} servers are unavailable. Try selecting one manually.",
	"Не вибрано сервер та відсутній конфіг.": "No server selected and no config available.",
	"Будь ласка, виберіть сервер або увімкніть автопідбір!":
		"Please select a server or enable auto-select!",
	"Сервер відхилив підключення. Спробуйте пізніше або інший сервер.":
		"The server rejected the connection. Try again later or choose another server.",
	"Сервер відхилив авторизацію (ліміт підключень).": "The server rejected authorization (connection limit).",
	"Сервер не відповідає. Спробуйте інший вузол.": "The server is not responding. Try another node.",
	"Жоден сервер не відповідає. Спробуйте пізніше або використовуйте вбудований профіль.":
		"No server is responding. Try again later or use the built-in profile.",
	"Підключення не встановлено (недоступна мережа/конфіг).": "Connection failed (network/config unavailable).",
	"Не вдалося підключитися. Спробуйте пізніше або інший сервер.":
		"Could not connect. Try again later or choose another server.",
	"Оновлено: знайдено {count} активних серверів": "Updated: found {count} active servers",
	"Не вдалося завантажити список серверів!": "Failed to load the server list!",
	"IP оновлено: {ip} ({country})": "IP updated: {ip} ({country})",
	"Локацію не визначено": "Location not detected",
	"Оновлення IP...":      "Refreshing IP...",
	"Обрано сервер: {desc} (Ping: {ping} ms)": "Server selected: {desc} (Ping: {ping} ms)",
	"Вибір сервера {desc} скасовано — знову швидкий авто-підбір":
		"Server selection {desc} cancelled — auto-select restored",
	"Режим підключення: {mode}": "Connection mode: {mode}",
	"⚡ Авто-підбір сервера":     "⚡ Auto server selection",
	"🌐 Мій обраний сервер":      "🌐 My selected server",
	"📂 Власний .ovpn конфіг":    "📂 Custom .ovpn config",
	"Оберіть сервер у списку нижче — після цього режим «Мій сервер» активується":
		"Choose a server from the list below — then the My server mode will activate",
	"Спочатку завантажте свій .ovpn у вкладці «Власний .ovpn»":
		"First upload your .ovpn in the Custom .ovpn tab",
	"Конфіг завантажено, але не вдалося зберегти на диск": "Config uploaded, but could not be saved to disk",
	"Завантажено: {name}": "Uploaded: {name}",
	"Помилка читання файлу: {ex}": "File read error: {ex}",
	"Профіль не знайдено: {name}": "Profile not found: {name}",
	"Профіль {name} готовий до підключення": "Profile {name} is ready to connect",
	"Помилка читання профілю: {ex}": "Profile read error: {ex}",
	"Профіль {name} видалено": "Profile {name} deleted",
	"Не вдалося видалити профіль {name}": "Failed to delete profile {name}",
	"Конфіг збережено: {name}": "Config saved: {name}",
	"VPN безпечно відключено": "VPN disconnected securely",
	"VPN підключено через {country}!": "VPN connected through {country}!",
	"Failover → {desc}": "Failover → {desc}",
	"Помилка: {ex}": "Error: {ex}",
	"Помилка підключення: {ex}": "Connection error: {ex}",

	// ── Settings / misc ───────────────────────────────────────────────────
	"Налаштування": "Settings",
	"Профіль":      "Profile",
	"Мої конфіги":  "My configs",
	"МОЇ КОНФІГИ ({count})": "MY CONFIGS ({count})",
	" · активний": " · active",
	"Завантажити свій конфіг (.ovpn)": "Upload my config (.ovpn)",
	"Виберіть свій .ovpn конфіг":      "Choose your .ovpn config",
	"OpenVPN конфігурації (*.ovpn)":   "OpenVPN configurations (*.ovpn)",
	"Відкрити вікно": "Open window",
	"Підключити":     "Connect",
	"Відключити":     "Disconnect",
	"Вийти":          "Quit",
	"Завантажити свій .ovpn конфіг": "Upload my .ovpn config",
	"Оновити список серверів":       "Refresh server list",

	// ── Log phrase fragments ──────────────────────────────────────────────
	"Старі процеси openvpn.exe не завершились — можлива помилка адаптера.":
		"Old openvpn.exe processes did not exit — adapter error possible.",
	"Реєстрація системної служби OpenVPN Interactive Service...": "Registering the OpenVPN Interactive Service...",
	"Завантажую драйвер Wintun...": "Loading the Wintun driver...",
	"Завантажую OpenVPN MSI (~8 МБ)...": "Downloading the OpenVPN MSI (~8 MB)...",
	"Розпаковую OpenVPN...": "Extracting OpenVPN...",
	"Фанітомних Wintun пристроїв не виявлено.": "No phantom Wintun devices found.",
	"tapctl.exe не знайдено в папці openVPN.": "tapctl.exe was not found in the openVPN folder.",
	"Драйвер tap0901 вже встановлений в системі.": "The tap0901 driver is already installed.",
	"Завантаження офіційного драйвера tap0901 (tap-windows6)...": "Downloading the official tap0901 driver (tap-windows6)...",
	"Встановлення драйвера tap0901 (тихий режим)...": "Installing the tap0901 driver (silent mode)...",
	"Драйвер tap0901 (tap-windows6) встановлено.": "The tap0901 driver (tap-windows6) is installed.",
	"Не вдалося підготувати жодний мережевий адаптер.": "Could not prepare any network adapter.",
	"Сервер не відповідає на запити TLS handshake. Вузол може бути перевантажений або вимкнений...":
		"The server is not responding to TLS handshake requests. The node may be overloaded or disabled...",
	"Сервер відгукнувся! Перевірка авторизації (AUTH)...": "The server responded! Checking authorization (AUTH)...",
	"Отримання мережевих параметрів від сервера (GET_CONFIG)...": "Receiving network parameters from the server (GET_CONFIG)...",
	"Налаштування віртуального Wintun адаптера та маршрутів...": "Configuring the virtual Wintun adapter and routes...",
	"Втрата зв'язку з сервером (RECONNECTING)...": "Connection to the server lost (RECONNECTING)...",
	"Увага: Ziper VPN запущено без прав адміністратора. Створення Wintun адаптера вимагає підвищених привілеїв Windows.":
		"Warning: Ziper VPN is running without administrator rights. Creating a Wintun adapter requires elevated Windows privileges.",
	"Ініціалізація OpenVPN рушія...": "Initializing the OpenVPN engine...",
	"Адаптер підготувати не вдалося — віддаю вибір драйвера OpenVPN.":
		"Could not prepare the adapter — letting OpenVPN choose the driver.",
	"Очікую Management Interface...": "Waiting for the Management Interface...",
	"Сесію ініційовано. Встановлюю з'єднання...": "Session initialized. Establishing connection...",
	"Підключення відмінено користувачем": "Connection cancelled by user",
	"OpenVPN завершився з кодом ": "OpenVPN exited with code ",
	"Помилка підключення: ": "Connection error: ",
	"Wintun успішно встановлено: ": "Wintun installed successfully: ",
	"Не вдалося завантажити Wintun: ": "Failed to load Wintun: ",
	"OpenVPN готовий до роботи: ": "OpenVPN is ready: ",
	"Копіюю wintun.dll до ": "Copying wintun.dll to ",
	"Видалення фантомного Wintun пристрою ": "Removing phantom Wintun device ",
	"Не вдалося видалити фантомний пристрій ": "Failed to remove phantom device ",
	"Створено Wintun адаптер ": "Created Wintun adapter ",
	"WintunCreateAdapter відхилено (код помилки Windows: ": "WintunCreateAdapter rejected (Windows error code: ",
	"Помилка ініціалізації Wintun (": "Wintun initialization error (",
	") — перехід на tap-windows6.": ") — switching to tap-windows6.",
	"Створено ": "Created ",
	" адаптер \"Ziper VPN\" (tapctl).": " adapter \"Ziper VPN\" (tapctl).",
	"tapctl повернув код ": "tapctl returned code ",
	", але адаптер створено за GUID.": ", but the adapter was created by GUID.",
	"Драйвер tap0901 не встановлено (код ": "tap0901 driver was not installed (code ",
	"Не вдалося встановити драйвер tap0901: ": "Failed to install the tap0901 driver: ",
	"Використовую ": "Using ",
	" драйвер.": " driver.",
	"Профіль сам задає протокол ": "The profile defines protocol ",
	" — налаштування '": " — setting '",
	"' для цього профілю проігноровано.": "' is ignored for this profile.",
	"Пропущено некоректну DNS адресу: ": "Skipped invalid DNS address: ",
	"Використовується власний DNS: ": "Using custom DNS: ",
	"Запуск тунелю OpenVPN (порт management: ": "Starting OpenVPN tunnel (management port: ",
	"Завантажено ": "Loaded ",
	" серверів з онлайн-дзеркала [OK]": " servers from the online mirror [OK]",
	" серверів з VPN Gate [OK]": " servers from VPN Gate [OK]",
	" серверів через Tor [OK]": " servers via Tor [OK]",
	"Сервери завантажено з локального кешу (": "Servers loaded from local cache (",
	"[TOR] Служба Tor вже працює на 127.0.0.1:": "[TOR] Tor is already running at 127.0.0.1:",
	"[TOR ERROR] tor.exe не знайдено!": "[TOR ERROR] tor.exe was not found!",
	"[TOR] Запуск процесу Tor SOCKS5 (порт ": "[TOR] Starting Tor SOCKS5 process (port ",
	"[TOR] Службу Tor успішно запущено (SOCKS5 127.0.0.1:": "[TOR] Tor started successfully (SOCKS5 127.0.0.1:",
	"[TOR WARN] Tor запущено, але порт ще не відповідає.": "[TOR WARN] Tor is running, but the port is not responding yet.",
	"[TOR ERROR] Не вдалося запустити Tor: ": "[TOR ERROR] Failed to start Tor: ",
	"[TOR] Службу Tor зупинено.": "[TOR] Tor service stopped.",
	"Ініціація відключення VPN...": "Initiating VPN disconnect...",
	"VPN-тунель повністю зупинено (process openvpn завершено, маршрути знято).":
		"VPN tunnel fully stopped (openvpn exited, routes removed).",
	"Не всі процеси openvpn.exe завершились — перевірте Диспетчер задач.":
		"Not all openvpn.exe processes exited — check Task Manager.",
	"Користувач скасував підключення...": "User cancelled the connection...",
	"Режим: Автоматичний підбір сервера (VPN Gate)...": "Mode: Automatic server selection (VPN Gate)...",
	"Оновлюю список публічних серверів...": "Refreshing the public server list...",
	"Підключення за власним файлом конфігурації...": "Connecting using a custom configuration file...",
	"Використовую вбудований auto_config.ovpn...": "Using the built-in auto_config.ovpn...",
	"=== Запуск захищеної сесії Ziper VPN ===": "=== Starting secure Ziper VPN session ===",
	"VPN тунель успішно встановлено!  ": "VPN tunnel established successfully!  ",
	"Значок системного трея запущено.": "System tray icon started.",
	"Не вдалося ініціалізувати трей: ": "Failed to initialize the tray: ",
	"Обрано сервер: ": "Server selected: ",
	"Користувач обрав сервер: ": "User selected server: ",
	"Користувач скасував вибір сервера: ": "User deselected server: ",
	" (повернено авто-підбір)": " (auto-select restored)",
	"Завантажено власний .ovpn конфіг: ": "Custom .ovpn config loaded: ",
	"Збережено власний .ovpn конфіг: ": "Custom .ovpn config saved: ",
	"Завантажено збережений власний .ovpn профіль: ": "Saved custom .ovpn profile loaded: ",
	"Видалено власний .ovpn профіль: ": "Custom .ovpn profile deleted: ",
	"Збережено налаштування: ": "Setting saved: ",
	"Оновлено: знайдено ": "Updated: found ",
	" активних серверів": " active servers",
	"VPN відключено (": "VPN disconnected (",
	"[INFO] Вибраний сервер VPN Gate скинуто — активним став власний .ovpn конфіг.":
		"[INFO] Selected VPN Gate server cleared — the custom .ovpn config is now active.",
	"[INFO] Повторну спробу скасовано (стан змінився).": "[INFO] Retry cancelled (state changed).",
	"[INFO] Повторна спроба підключення...": "[INFO] Retrying connection...",
	"Перевірка зовнішньої IP-адреси та геопозиції...": "Checking external IP address and geolocation...",
	"[INFO] Ігноровано швидкий повторний клік — підключення запускається.":
		"[INFO] Fast double-click ignored — connection is starting.",
	"[WARN] Запущено без прав адміністратора — використовується tap-windows6 (Wintun недоступний).":
		"[WARN] Running without administrator rights — using tap-windows6 (Wintun unavailable).",
	"[INFO] Режим «Мій сервер»: сервер не обрано — використовую авто-підбір.":
		"[INFO] \"My server\" mode: no server selected — using automatic selection.",
	"[ERROR] Не вдалося завантажити список серверів з мережі!":
		"[ERROR] Failed to download the server list from the network!",
	"[ERROR] Порожній конфіг, пробуємо наступний сервер...": "[ERROR] Empty config, trying the next server...",
	"[ERROR] Не вдалося підключитися до ": "[ERROR] Failed to connect to ",
	" серверів з верху списку": " servers at the top of the list",
	"[ERROR] Не вдалося прочитати збережений .ovpn: ": "[ERROR] Failed to read the saved .ovpn: ",
	"[WARN] Збережений .ovpn не знайдено: ": "[WARN] Saved .ovpn not found: ",
	"[CRITICAL ERROR] Не вдалося підключитися: ": "[CRITICAL ERROR] Failed to connect: ",
	"[INFO] Failover скасовано (користувач сам змінив стан підключення).":
		"[INFO] Failover cancelled (the user changed the connection state).",
	"[ERROR] Failover помилка: ": "[ERROR] Failover error: ",
	"Завантаження списку публічних серверів...": "Downloading the public server list...",
	"[WARN] Перша спроба не вдалася — повторюю завантаження...": "[WARN] First attempt failed — retrying the download...",
	"[ERROR] Не вдалося завантажити список серверів. Перевірте інтернет.":
		"[ERROR] Failed to download the server list. Check your internet connection.",
	"[TOR] SOCKS5 порт змінено: ": "[TOR] SOCKS5 port changed: ",
	". Перезапустіть Tor, щоб застосувати.": ". Restart Tor to apply.",
	"Зупинка процесу Tor...": "Stopping the Tor process...",
	"Запуск Tor SOCKS5 проксі (127.0.0.1:": "Starting Tor SOCKS5 proxy (127.0.0.1:",
	"[ERROR] Не вдалося прочитати .ovpn: ": "[ERROR] Failed to read .ovpn: ",
	"[ERROR] Не вдалося прочитати збережений .ovpn ": "[ERROR] Failed to read the saved .ovpn ",
	"[ERROR] Не вдалося видалити профіль: ": "[ERROR] Failed to delete the profile: ",
	"[WARN] pystray/Pillow не встановлено — значок трея недоступний.":
		"[WARN] Tray icon is unavailable.",
	"[SELF-HEAL] Тунель активний, але інтерфейс цього не показував — виправляю стан.":
		"[SELF-HEAL] The tunnel is active but the UI did not show it — fixing the state.",
	"[SELF-HEAL] Стан CONNECTED застосовано до елементів UI.": "[SELF-HEAL] CONNECTED state applied to the UI elements.",
	"[SELF-HEAL] Тунель неактивний, а інтерфейс показував з'єднання — виправляю стан.":
		"[SELF-HEAL] The tunnel is inactive but the UI showed a connection — fixing the state.",
	"[SELF-HEAL] Стан DISCONNECTED застосовано до елементів UI.": "[SELF-HEAL] DISCONNECTED state applied to the UI elements.",
	"[AUTO] Налаштування Автопідключення АКТИВНЕ. Запускаю VPN...": "[AUTO] Auto-connect is ENABLED. Starting the VPN...",
	"[WARN] Сервер ": "[WARN] Server ",
	" відсутній client, пробуємо наступний...": " has no client config, trying the next one...",
	"[WARN] Помилка обробки сервера ": "[WARN] Error processing server ",
	"[ERROR] Не вдалося підключитися: ": "[ERROR] Failed to connect: ",
	"[ERROR] Імпорт конфіга не вдався: ": "[ERROR] Config import failed: ",
	": імпорт конфіга не вдався: ": ": config import failed: ",
	"[ERROR] Не вдалося зберегти конфіг ": "[ERROR] Failed to save config ",
	": не вдалося зберегти конфіг ": ": failed to save config ",
	"Трей": "Tray",
	"Імпорт": "Import",
	": збережено власний .ovpn конфіг: config/profiles/": ": custom .ovpn config saved: config/profiles/",
	"Автозавантажено власний .ovpn конфіг: ": "Custom .ovpn config auto-loaded: ",
	"Помилка: ": "Error: ",

	// ── WireGuard ─────────────────────────────────────────────────────────
	"=== Запуск WireGuard тунелю ===": "=== Starting the WireGuard tunnel ===",
	"Профіль: {name} | Адреса тунелю: {addr} | Пірів: {peers}":
		"Profile: {name} | Tunnel address: {addr} | Peers: {peers}",
	"Не вдалося розпізнати Endpoint «{ep}»: {ex}":
		"Failed to resolve Endpoint \"{ep}\": {ex}",
	"Endpoint {i}: {ep}": "Endpoint {i}: {ep}",
	"wintun.dll скопійовано до папки застосунку.": "wintun.dll copied to the app folder.",
	"Драйвер Wintun встановлено: ": "Wintun driver installed: ",
	"Не вдалося завантажити драйвер Wintun": "Failed to download the Wintun driver",
	"Не вдалося створити Wintun адаптер: {ex} (потрібні права адміністратора)":
		"Failed to create the Wintun adapter: {ex} (administrator privileges required)",
	"Wintun адаптер «{name}» створено (MTU {mtu}).": "Wintun adapter \"{name}\" created (MTU {mtu}).",
	"Помилка конфігурації тунелю: {ex}": "Tunnel configuration error: {ex}",
	"Не вдалося запустити тунель: {ex}": "Failed to start the tunnel: {ex}",
	"Тунель ініціалізовано, очікую хендшейк з сервером...":
		"Tunnel initialized, waiting for the handshake with the server...",
	"Хендшейк не вдався: сервер недоступний або ключі невірні.":
		"Handshake failed: the server is unreachable or the keys are invalid.",
	"Хендшейк ще не отримано — трафік через тунель не генерується...":
		"Handshake not received yet — no traffic is being generated through the tunnel...",
	"WireGuard хендшейк успішний — тунель активний! ✓": "WireGuard handshake succeeded — tunnel is active! ✓",
	"Закриття WireGuard тунелю...": "Closing the WireGuard tunnel...",
	"Хендшейк застарілий (>180с) — сервер може бути недоступним.":
		"Handshake is stale (>180s) — the server may be unreachable.",
	"Адресу {addr} призначено адаптеру «{name}».":
		"Address {addr} assigned to the \"{name}\" adapter.",
	"Метрику адаптера тунелю знижено (пріоритет над фізичним інтернетом).":
		"Tunnel adapter metric lowered (priority over the physical network).",
	"DNS тунелю: {servers}": "Tunnel DNS: {servers}",
	"Full-tunnel: маршрути 0.0.0.0/1 + 128.0.0.0/1 через тунель.":
		"Full-tunnel: routes 0.0.0.0/1 + 128.0.0.0/1 through the tunnel.",
	"Full-tunnel (IPv6): маршрути ::/1 + 8000::/1 через тунель.":
		"Full-tunnel (IPv6): routes ::/1 + 8000::/1 through the tunnel.",
	"Маршрут {p} не додано (можливо, вже існує).":
		"Route {p} was not added (it may already exist).",
	"Сокети не прив'язано до «{name}»: {ex}":
		"Sockets were not bound to \"{name}\": {ex}",
	"UDP сокети тунелю прив'язано до фізичного інтерфейсу «{name}».":
		"Tunnel UDP sockets bound to the physical interface \"{name}\".",
	"Без прав адміністратора маршрути не встановити.":
		"Routes cannot be installed without administrator privileges.",
	"Профіль WireGuard не обрано.": "No WireGuard profile is selected.",
	"Спочатку оберіть .conf профіль на вкладці WireGuard.":
		"First select a .conf profile on the WireGuard tab.",
	"WireGuard підключено через {country}!": "WireGuard connected via {country}!",
	"WireGuard відключено: {ex}": "WireGuard disconnected: {ex}",
	"WireGuard успішно відключено.": "WireGuard disconnected successfully.",
	"Некоректний WireGuard конфіг: {ex}": "Invalid WireGuard config: {ex}",
	"Виберіть свій .conf конфіг WireGuard": "Select your WireGuard .conf config",
	"WireGuard конфігурації (*.conf)": "WireGuard configurations (*.conf)",
	"Некоректний .conf конфіг: {ex}": "Invalid .conf config: {ex}",
}
