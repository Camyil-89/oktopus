/**
 * Примеры Lua для правил инспекции исходящих HTTP-запросов (MITM / http://).
 * Каждый скрипт обязан определять function inspect(ctx).
 * Возврат false / "allow" — правило не сработало; true / "match" — сработало (действие из настроек правила).
 * Порядок: при совпадении «Запретить» — сразу блок; «Разрешить» — проверяем следующие правила.
 */

export type InspectLuaExample = {
  id: string;
  title: string;
  description: string;
  script: string;
};

export const INSPECT_LUA_EXAMPLES: InspectLuaExample[] = [
  {
    id: "any-file-upload-heuristic",
    title: "Похоже на отправку файла (эвристика по заголовкам)",
    description:
      "POST/PUT/PATCH: multipart, бинарные и медиа Content-Type, Content-Disposition, имена файлов из тела (лимит 4 МБ). Пустой Content-Type без признаков файла не блокируется.",
    script: `local function is_plausible_filename(s)
  return type(s) == "string"
    and s ~= ""
    and #s <= 255
    and not s:find("[\\r\\n%z]")
end

local function collect_names(raw)
  local raw_names, clean_names = {}, {}
  if type(raw) == "table" then
    for i = 1, #raw do
      local v = raw[i]
      if type(v) == "string" and v ~= "" then
        raw_names[#raw_names + 1] = v
        if is_plausible_filename(v) then
          clean_names[#clean_names + 1] = v
        end
      end
    end
  elseif type(raw) == "string" and raw ~= "" then
    raw_names[1] = raw
    if is_plausible_filename(raw) then
      clean_names[1] = raw
    end
  end
  return raw_names, clean_names
end

function inspect(ctx)
  local method = ctx.method or ""
  if method ~= "POST" and method ~= "PUT" and method ~= "PATCH" then
    return false
  end

  local raw_names, clean_names = collect_names(ctx.upload_filenames)

  local ct = string.lower(tostring(ctx.content_type or ""))
  local base = ct:match("^%s*([^;]+)") or ""
  base = (base:gsub("%s+$", ""))
  local clen = tonumber(ctx.content_length) or -1

  local header_cd = ctx:header("Content-Disposition") or ""
  local header_has_filename =
    header_cd ~= "" and string.lower(header_cd):find("filename=", 1, true) ~= nil

  ctx:log("upload_method", method)
  ctx:log("upload_content_type", base)
  ctx:log("upload_content_length", clen)
  ctx:log("upload_filenames_raw_count", #raw_names)
  ctx:log("upload_filenames", clean_names)          -- без мусора
  ctx:log("upload_header_has_filename", header_has_filename)

  local function match(reason)
    ctx:log("upload_block_reason", reason)
    return true
  end

  -- A. Имена файлов: детект по сырому списку (как в исходном правиле).
  if #raw_names > 0 then
    return match("multipart_with_filenames")
  end

  -- B. Явный filename= в заголовке запроса.
  if header_has_filename then
    return match("content_disposition_header_filename")
  end

  -- C. multipart без имён — обычная форма.
  if base:find("multipart/", 1, true) then
    return false
  end

  -- D. Нет Content-Type.
  if base == "" then
    return false
  end

  -- E. Пустое тело.
  if clen == 0 then
    return false
  end

  local binary_exact = {
    ["application/octet-stream"]     = true,
    ["application/pdf"]              = true,
    ["application/zip"]              = true,
    ["application/x-zip-compressed"] = true,
    ["application/gzip"]             = true,
    ["application/x-gzip"]           = true,
    ["application/x-tar"]            = true,
    ["application/x-7z-compressed"]  = true,
    ["application/vnd.rar"]          = true,
    ["application/msword"]           = true,
  }
  if binary_exact[base] then
    return match("binary_content_type:" .. base)
  end

  if base:find("^image/") or base:find("^video/") or base:find("^audio/") then
    return match("media_content_type:" .. base)
  end

  if base:find("^application/vnd%.openxmlformats%-officedocument") then
    return match("ooxml_content_type:" .. base)
  end
  if base:find("^application/vnd%.ms%-") then
    return match("ms_office_content_type:" .. base)
  end
  if base:find("^application/vnd%.oasis%.opendocument") then
    return match("odf_content_type:" .. base)
  end

  return false
end`,
  },
  {
    id: "post-multipart",
    title: "POST с multipart (быстро)",
    description:
      "Только метод и Content-Type, без разбора тела. Подходит для грубой блокировки форм upload.",
    script: `function inspect(ctx)
  if ctx.method ~= "POST" and ctx.method ~= "PUT" then
    return false
  end
  local ct = ctx.content_type
  if ct == nil or ct == "" then
    return false
  end
  -- plain search (4-й аргумент true) быстрее regex
  if ct:find("multipart/form-data", 1, true) then
    return true
  end
  return false
end`,
  },
  {
    id: "host-upload-path",
    title: "Путь upload на хосте",
    description: "SNI/Host + PATH. Ранний выход, если хост не тот.",
    script: `function inspect(ctx)
  local host = ctx.host or ""
  if not host:find("dropbox%.com", 1) then
    return false
  end
  local path = ctx.path or "/"
  if path:find("/upload", 1, true) then
    return true
  end
  return false
end`,
  },
  {
    id: "large-body",
    title: "Большой исходящий body",
    description:
      "Content-Length из заголовка (−1 если chunked). Не читает тело.",
    script: `local LIMIT = 50 * 1024 * 1024

function inspect(ctx)
  if ctx.method ~= "POST" and ctx.method ~= "PUT" then
    return false
  end
  local n = ctx.content_length
  if n == nil or n < 0 then
    return false
  end
  if n >= LIMIT then
    return true
  end
  return false
end`,
  },
  {
    id: "group-external",
    title: "Группа + внешний хост",
    description: "ctx.has_group и простая проверка домена.",
    script: `function inspect(ctx)
  if not ctx:has_group("intern") then
    return false
  end
  local host = (ctx.host or ""):lower()
  if host == "" then
    return false
  end
  if host:find("%.corp%.local$") then
    return false
  end
  if ctx.method == "POST" or ctx.method == "PUT" then
    return true
  end
  return false
end`,
  },
  {
    id: "header-api",
    title: "Заголовок X-Requested-With",
    description: "ctx.header(name) — один заголовок без копирования всех.",
    script: `function inspect(ctx)
  if ctx.method ~= "POST" then
    return false
  end
  local xrw = ctx:header("X-Requested-With")
  if xrw == "XMLHttpRequest" then
    local ct = ctx.content_type or ""
    if ct:find("application/json", 1, true) then
      return false
    end
    if ct:find("multipart/", 1, true) then
      return true
    end
  end
  return false
end`,
  },
  {
    id: "whitelist-octet-stream",
    title: "Разрешить только octet-stream для пользователя",
    description:
      "При совпадении выберите в правиле «Разрешить» — запрос пройдёт (раньше других deny-правил не должно быть).",
    script: `function inspect(ctx)
  if ctx.user == nil or ctx.user == "" then
    return false
  end
  local ct = ctx.content_type or ""
  if ct:find("application/octet-stream", 1, true) then
    return true
  end
  return false
end`,
  },
];

export const INSPECT_LUA_API_REFERENCE = `Поля ctx (исходящий запрос):
  ctx.method          — GET, POST, …
  ctx.host            — SNI / Host
  ctx.path            — путь URL
  ctx.url             — полный URL
  ctx.content_type    — Content-Type
  ctx.content_length  — число (−1 если неизвестно)
  ctx.user            — имя proxy-auth
  ctx.client_ip       — адрес клиента
  ctx.groups          — таблица групп (1..n)
  ctx:header("Name")  — заголовок
  ctx:has_group("x")  — bool
  ctx.upload_filenames — таблица имён файлов (из заголовков и multipart-тела, лимит 4 МБ)
  ctx:log("key", value) — записать в extra журнала под id этого правила
  ctx:log({ key = value, ... }) — несколько полей сразу

Возврат inspect:
  false / nil / "allow" — правило не подошло
  true / "match"        — подошло (в правиле: запретить или разрешить)

Порядок правил: «Запретить» при совпадении останавливает проверку; «Разрешить» — только отмечает совпадение и идёт дальше. Whitelist: несколько «Разрешить» сверху, внизу жёсткие «Запретить».

Работает только когда прокси видит HTTP (режим MITM или http://). В tunnel HTTPS внутри туннеля не анализируется.`;
